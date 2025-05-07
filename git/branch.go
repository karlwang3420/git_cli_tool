// filepath: h:\code_base\git_cli_tool\git\branch.go
package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"git_cli_tool/config"
)

// GetCurrentBranch gets the current branch name of the repository
func GetCurrentBranch(repoPath string) (string, error) {
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve absolute path: %v", err)
	}

	// Check if repository exists
	if _, err := os.Stat(filepath.Join(absPath, ".git")); os.IsNotExist(err) {
		return "", fmt.Errorf("not a git repository or directory does not exist")
	}

	cmd := exec.Command("git", "-C", absPath, "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %v", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// CheckBranchExists checks if a branch exists locally
func CheckBranchExists(repoPath string, branch string) (bool, error) {
	cmd := exec.Command("git", "-C", repoPath, "show-ref", "--verify", "--quiet", "refs/heads/"+branch)
	err := cmd.Run()
	
	if err != nil {
		// Exit code 1 means branch doesn't exist, which is not an error for our purposes
		if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() == 1 {
			return false, nil
		}
		return false, err
	}
	
	return true, nil
}

// CheckRemoteBranchExists checks if a branch exists on the remote
func CheckRemoteBranchExists(repoPath string, branch string) (bool, error) {
	cmd := exec.Command("git", "-C", repoPath, "show-ref", "--verify", "--quiet", "refs/remotes/origin/"+branch)
	err := cmd.Run()
	
	if err != nil {
		// Exit code 1 means branch doesn't exist, which is not an error for our purposes
		if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() == 1 {
			return false, nil
		}
		return false, err
	}
	
	return true, nil
}

// SwitchBranchWithFallback tries to switch to each branch in the given order
func SwitchBranchWithFallback(repoPath string, branches []string) error {
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %v", err)
	}

	// Check if repository exists
	if _, err := os.Stat(filepath.Join(absPath, ".git")); os.IsNotExist(err) {
		return fmt.Errorf("not a git repository or directory does not exist")
	}

	// Try each branch in order
	var lastError error
	for i, branch := range branches {
		// Check if branch exists locally
		branchExists, err := CheckBranchExists(absPath, branch)
		if err != nil {
			lastError = fmt.Errorf("failed to check if branch %s exists: %v", branch, err)
			continue
		}

		// If branch exists locally, switch to it
		if branchExists {
			cmd := exec.Command("git", "-C", absPath, "checkout", branch)
			output, err := cmd.CombinedOutput()
			if err != nil {
				lastError = fmt.Errorf("git checkout failed for branch %s: %v\n%s", branch, err, output)
				continue
			}
			fmt.Printf("Successfully switched to branch %s in %s\n", branch, repoPath)
			return nil
		}

		// If branch doesn't exist locally, try to fetch and check remote
		fmt.Printf("Branch %s not found locally in %s, fetching from remote...\n", branch, repoPath)
		
		// Fetch from remote
		fetchCmd := exec.Command("git", "-C", absPath, "fetch")
		output, err := fetchCmd.CombinedOutput()
		if err != nil {
			lastError = fmt.Errorf("git fetch failed: %v\n%s", err, output)
			continue
		}
		
		// Check if remote branch exists
		remoteBranchExists, err := CheckRemoteBranchExists(absPath, branch)
		if err != nil {
			lastError = fmt.Errorf("failed to check if remote branch %s exists: %v", branch, err)
			continue
		}
		
		if remoteBranchExists {
			// Create tracking branch
			trackCmd := exec.Command("git", "-C", absPath, "checkout", "-b", branch, "--track", "origin/"+branch)
			_, err := trackCmd.CombinedOutput()
			if err != nil {
				// If branch creation fails, try direct checkout of remote branch
				checkoutCmd := exec.Command("git", "-C", absPath, "checkout", branch)
				checkoutOutput, err := checkoutCmd.CombinedOutput()
				if err != nil {
					lastError = fmt.Errorf("failed to checkout remote branch %s: %v\n%s", branch, err, checkoutOutput)
					continue
				}
			}
			fmt.Printf("Successfully switched to branch %s in %s\n", branch, repoPath)
			return nil
		}

		// If we're on the last branch and none have worked, log that we're trying the next branch
		if i < len(branches)-1 {
			fmt.Printf("Branch %s not found locally or remotely in %s, trying next branch...\n", branch, repoPath)
		}
	}

	// If we get here, none of the branches worked
	if lastError != nil {
		return fmt.Errorf("failed to switch to any branch in repository %s: %v", repoPath, lastError)
	}
	return fmt.Errorf("none of the specified branches exist in repository %s", repoPath)
}

// SwitchBranchesSequential switches branches in the provided repositories sequentially
func SwitchBranchesSequential(repositories []config.Repository, branches []string) {
	for _, repo := range repositories {
		err := SwitchBranchWithFallback(repo.Path, branches)
		if err != nil {
			fmt.Printf("Error switching branch in %s: %v\n", repo.Path, err)
		}
	}
}

// SwitchBranchesParallel switches branches in the provided repositories in parallel
func SwitchBranchesParallel(repositories []config.Repository, branches []string) {
	var wg sync.WaitGroup
	wg.Add(len(repositories))

	for _, repo := range repositories {
		go func(r config.Repository) {
			defer wg.Done()
			err := SwitchBranchWithFallback(r.Path, branches)
			if err != nil {
				fmt.Printf("Error switching branch in %s: %v\n", r.Path, err)
			}
		}(repo)
	}

	wg.Wait()
}

// SwitchBranchWithFallbackAndStash tries to switch to each branch in the given order, stashing changes if requested
func SwitchBranchWithFallbackAndStash(repoPath string, branches []string, stashName string) error {
	// If stashName is not empty, stash changes first
	if stashName != "" {
		err := StashChanges(repoPath, stashName)
		if err != nil {
			return fmt.Errorf("failed to stash changes: %v", err)
		}
	}
	
	// Proceed with normal branch switching
	return SwitchBranchWithFallback(repoPath, branches)
}

// SwitchBranchesSequentialWithStash switches branches in the provided repositories sequentially, with optional stashing
func SwitchBranchesSequentialWithStash(repositories []config.Repository, branches []string, stashName string) {
	for _, repo := range repositories {
		var err error
		if stashName != "" {
			err = SwitchBranchWithFallbackAndStash(repo.Path, branches, stashName)
		} else {
			err = SwitchBranchWithFallback(repo.Path, branches)
		}
		
		if err != nil {
			fmt.Printf("Error switching branch in %s: %v\n", repo.Path, err)
		}
	}
}

// SwitchBranchesParallelWithStash switches branches in the provided repositories in parallel, with optional stashing
func SwitchBranchesParallelWithStash(repositories []config.Repository, branches []string, stashName string) {
	var wg sync.WaitGroup
	wg.Add(len(repositories))

	for _, repo := range repositories {
		go func(r config.Repository) {
			defer wg.Done()
			
			var err error
			if stashName != "" {
				err = SwitchBranchWithFallbackAndStash(r.Path, branches, stashName)
			} else {
				err = SwitchBranchWithFallback(r.Path, branches)
			}
			
			if err != nil {
				fmt.Printf("Error switching branch in %s: %v\n", r.Path, err)
			}
		}(repo)
	}

	wg.Wait()
}

// SwitchToBranch switches to a specific branch in a repository
func SwitchToBranch(repoPath string, branch string) error {
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %v", err)
	}

	// Check if repository exists
	if _, err := os.Stat(filepath.Join(absPath, ".git")); os.IsNotExist(err) {
		return fmt.Errorf("not a git repository or directory does not exist")
	}

	// Check if branch exists locally
	branchExists, err := CheckBranchExists(absPath, branch)
	if err != nil {
		return fmt.Errorf("failed to check if branch %s exists: %v", branch, err)
	}

	// If branch exists locally, switch to it
	if branchExists {
		cmd := exec.Command("git", "-C", absPath, "checkout", branch)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("git checkout failed for branch %s: %v\n%s", branch, err, output)
		}
		fmt.Printf("Successfully switched to branch %s in %s\n", branch, repoPath)
		return nil
	}

	// If branch doesn't exist locally, try to find and check it out from remote
	fmt.Printf("Branch %s not found locally in %s, checking remote...\n", branch, repoPath)
	
	// Fetch from remote
	fetchCmd := exec.Command("git", "-C", absPath, "fetch")
	fetchOutput, err := fetchCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git fetch failed: %v\n%s", err, fetchOutput)
	}
	
	// Check if remote branch exists
	remoteBranchExists, err := CheckRemoteBranchExists(absPath, branch)
	if err != nil {
		return fmt.Errorf("failed to check if remote branch %s exists: %v", branch, err)
	}
	
	if remoteBranchExists {
		// Create tracking branch
		trackCmd := exec.Command("git", "-C", absPath, "checkout", "-b", branch, "--track", "origin/"+branch)
		_, err := trackCmd.CombinedOutput()
		if err != nil {
			// If branch creation fails, try direct checkout of remote branch
			checkoutCmd := exec.Command("git", "-C", absPath, "checkout", branch)
			checkoutOutput, err := checkoutCmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("failed to checkout remote branch %s: %v\n%s", branch, err, checkoutOutput)
			}
		}
		fmt.Printf("Successfully switched to branch %s in %s\n", branch, repoPath)
		return nil
	}

	return fmt.Errorf("branch %s not found locally or remotely in %s", branch, repoPath)
}

// SwitchBranch attempts to switch to the given branch in the specified repository
func SwitchBranch(repoPath string, branch string, stashChanges bool) error {
	// Check if we need to stash changes
	if stashChanges {
		if err := StashChanges(repoPath, branch); err != nil {
			return fmt.Errorf("failed to stash changes: %v", err)
		}
	}

	// Try to check out the branch directly first
	cmd := exec.Command("git", "-C", repoPath, "checkout", branch)
	if _, err := cmd.CombinedOutput(); err == nil {
		fmt.Printf("Successfully switched to branch %s in %s\n", branch, repoPath)
		return nil
	} else {
		// Branch doesn't exist locally, check if it exists remotely
		fmt.Printf("Branch %s not found locally in %s, checking remote...\n", branch, repoPath)
		
		// Fetch from remote to get latest branches
		fetchCmd := exec.Command("git", "-C", repoPath, "fetch")
		if _, err := fetchCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to fetch from remote: %v", err)
		}
		
		// Check if the branch exists as a remote branch
		lsRemoteCmd := exec.Command("git", "-C", repoPath, "ls-remote", "--heads", "origin", branch)
		output, _ := lsRemoteCmd.CombinedOutput()
		
		if len(output) > 0 {
			// Remote branch exists, check it out
			checkoutCmd := exec.Command("git", "-C", repoPath, "checkout", "-b", branch, "--track", "origin/"+branch)
			_, err := checkoutCmd.CombinedOutput()
			
			if err != nil {
				// If that failed, maybe the branch already exists locally but is tracking a different remote
				// Try a simple checkout with tracking
				checkoutTrackCmd := exec.Command("git", "-C", repoPath, "checkout", "--track", "origin/"+branch)
				output, err = checkoutTrackCmd.CombinedOutput()
				if err != nil {
					return fmt.Errorf("failed to checkout branch %s: %v\n%s", branch, err, string(output))
				}
			}
			
			fmt.Printf("Successfully switched to branch %s in %s\n", branch, repoPath)
			return nil
		} else {
			// Branch doesn't exist remotely either
			fmt.Printf("Branch %s not found locally or remotely in %s\n", branch, repoPath)
			return fmt.Errorf("branch %s not found locally or remotely", branch)
		}
	}
}