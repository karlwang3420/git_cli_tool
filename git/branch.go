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
	"git_cli_tool/log"
)

// SwitchResult holds the result of switching a branch in a repository
type SwitchResult struct {
	RepoPath     string
	RepoName     string
	FromBranch   string
	ToBranch     string
	Success      bool
	Message      string
	FromRemote   bool
	AlreadyOnIt  bool
}


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
			log.PrintSuccess(fmt.Sprintf("Successfully switched to branch %s in %s", branch, repoPath))
			return nil
		}

		// If branch doesn't exist locally, try to fetch and check remote
		log.PrintInfo(fmt.Sprintf("Branch %s not found locally in %s, fetching from remote...", branch, repoPath))

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
			log.PrintSuccess(fmt.Sprintf("Successfully switched to branch %s in %s", branch, repoPath))
			return nil
		}

		// If we're on the last branch and none have worked, log that we're trying the next branch
		if i < len(branches)-1 {
			log.PrintInfo(fmt.Sprintf("Branch %s not found locally or remotely in %s, trying next branch...", branch, repoPath))
		}
	}

	// If we get here, none of the branches worked
	if lastError != nil {
		return fmt.Errorf("failed to switch to any branch in repository %s: %v", repoPath, lastError)
	}
	return fmt.Errorf("none of the specified branches exist in repository %s", repoPath)
}

// SwitchBranchWithResult switches to a branch and returns the result (no logging)
func SwitchBranchWithResult(repoPath string, branches []string) SwitchResult {
	absPath, err := filepath.Abs(repoPath)
	repoName := filepath.Base(repoPath)
	
	result := SwitchResult{
		RepoPath: repoPath,
		RepoName: repoName,
	}

	if err != nil {
		result.Message = "failed to resolve path"
		return result
	}

	// Check if repository exists
	if _, err := os.Stat(filepath.Join(absPath, ".git")); os.IsNotExist(err) {
		result.Message = "not a git repository"
		return result
	}

	// Get current branch
	result.FromBranch, _ = GetCurrentBranch(absPath)

	// Try each branch in order
	for _, branch := range branches {
		// Check if already on this branch
		if result.FromBranch == branch {
			result.ToBranch = branch
			result.Success = true
			result.AlreadyOnIt = true
			result.Message = "already on target"
			return result
		}

		// Check if branch exists locally
		branchExists, err := CheckBranchExists(absPath, branch)
		if err != nil {
			continue
		}

		if branchExists {
			cmd := exec.Command("git", "-C", absPath, "checkout", branch)
			_, err := cmd.CombinedOutput()
			if err != nil {
				continue
			}
			result.ToBranch = branch
			result.Success = true
			result.Message = "switched"
			return result
		}

		// Try to fetch and check remote
		fetchCmd := exec.Command("git", "-C", absPath, "fetch")
		fetchCmd.CombinedOutput()

		remoteBranchExists, err := CheckRemoteBranchExists(absPath, branch)
		if err != nil || !remoteBranchExists {
			continue
		}

		// Create tracking branch
		trackCmd := exec.Command("git", "-C", absPath, "checkout", "-b", branch, "--track", "origin/"+branch)
		_, err = trackCmd.CombinedOutput()
		if err != nil {
			// Try direct checkout
			checkoutCmd := exec.Command("git", "-C", absPath, "checkout", branch)
			_, err = checkoutCmd.CombinedOutput()
			if err != nil {
				continue
			}
		}
		result.ToBranch = branch
		result.Success = true
		result.FromRemote = true
		result.Message = "switched (from remote)"
		return result
	}

	result.Message = fmt.Sprintf("no matching branch found")
	return result
}

// SwitchBranchWithFallbackAndStash tries to switch to each branch in the given order, stashing changes if requested
func SwitchBranchWithFallbackAndStash(repoPath string, branches []string, stashName string) (bool, error) {
	wasStashed := false
	// If stashName is not empty, stash changes first
	if stashName != "" {
		var err error
		wasStashed, err = StashChanges(repoPath, stashName)
		if err != nil {
			return false, fmt.Errorf("failed to stash changes: %v", err)
		}
	}

	// Proceed with normal branch switching
	return wasStashed, SwitchBranchWithFallback(repoPath, branches)
}

// SwitchBranchesWithStash switches branches in the provided repositories in parallel, with optional stashing
func SwitchBranchesWithStash(repositories []config.Repository, branches []string, stashName string) map[string]bool {
	var wg sync.WaitGroup
	var mutex sync.Mutex // Mutex to protect the stashedRepos map from concurrent writes
	stashedRepos := make(map[string]bool)

	wg.Add(len(repositories))

	for _, repo := range repositories {
		go func(r config.Repository) {
			defer wg.Done()

			var err error
			if stashName != "" {
				var wasStashed bool
				wasStashed, err = SwitchBranchWithFallbackAndStash(r.Path, branches, stashName)
				if err == nil && wasStashed {
					mutex.Lock()
					stashedRepos[r.Path] = true
					mutex.Unlock()
				}
			} else {
				err = SwitchBranchWithFallback(r.Path, branches)
			}

			if err != nil {
				log.PrintErrorNoExit(log.ErrGitCheckoutFailed, fmt.Sprintf("Error switching branch in %s", r.Path), err)
			}
		}(repo)
	}

	wg.Wait()
	return stashedRepos
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
		log.PrintSuccess(fmt.Sprintf("Successfully switched to branch %s in %s", branch, repoPath))
		return nil
	}

	// If branch doesn't exist locally, try to find and check it out from remote
	log.PrintInfo(fmt.Sprintf("Branch %s not found locally in %s, checking remote...", branch, repoPath))

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
		log.PrintSuccess(fmt.Sprintf("Successfully switched to branch %s in %s", branch, repoPath))
		return nil
	}

	return fmt.Errorf("branch %s not found locally or remotely in %s", branch, repoPath)
}

// SwitchBranch attempts to switch to the given branch in the specified repository
func SwitchBranch(repoPath string, branch string, stashChanges bool) error {
	// Check if we need to stash changes
	if stashChanges {
		if _, err := StashChanges(repoPath, branch); err != nil {
			return fmt.Errorf("failed to stash changes: %v", err)
		}
	}

	// Try to check out the branch directly first
	cmd := exec.Command("git", "-C", repoPath, "checkout", branch)
	if _, err := cmd.CombinedOutput(); err == nil {
		log.PrintSuccess(fmt.Sprintf("Successfully switched to branch %s in %s", branch, repoPath))
		return nil
	} else {
		// Branch doesn't exist locally, check if it exists remotely
		log.PrintInfo(fmt.Sprintf("Branch %s not found locally in %s, checking remote...", branch, repoPath))

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

			log.PrintSuccess(fmt.Sprintf("Successfully switched to branch %s in %s", branch, repoPath))
			return nil
		} else {
			// Branch doesn't exist remotely either
			log.PrintWarning(fmt.Sprintf("Branch %s not found locally or remotely in %s", branch, repoPath))
			return fmt.Errorf("branch %s not found locally or remotely", branch)
		}
	}
}
