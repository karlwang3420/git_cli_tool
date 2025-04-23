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
			trackOutput, err := trackCmd.CombinedOutput()
			if err != nil {
				// If branch creation fails, try direct checkout of remote branch
				checkoutCmd := exec.Command("git", "-C", absPath, "checkout", branch)
				checkoutOutput, err := checkoutCmd.CombinedOutput()
				if err != nil {
					lastError = fmt.Errorf("failed to checkout remote branch %s: %v\n%s", branch, err, checkoutOutput)
					continue
				}
			} else {
				_ = trackOutput // Use output to avoid unused variable warning
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

// DeleteLocalTags deletes all local tags in the repository
func DeleteLocalTags(repoPath string) error {
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %v", err)
	}

	// Check if repository exists
	if _, err := os.Stat(filepath.Join(absPath, ".git")); os.IsNotExist(err) {
		return fmt.Errorf("not a git repository or directory does not exist")
	}

	// Get all local tags
	listCmd := exec.Command("git", "-C", absPath, "tag")
	tagList, err := listCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to list tags: %v", err)
	}

	// If there are no tags, we're done
	tagListStr := strings.TrimSpace(string(tagList))
	if len(tagListStr) == 0 {
		fmt.Printf("No tags to delete in %s\n", repoPath)
		return nil
	}

	// Split tags by newline
	tags := strings.Split(tagListStr, "\n")
	fmt.Printf("Found %d tags to delete in %s\n", len(tags), repoPath)

	// Delete tags one by one
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		
		deleteCmd := exec.Command("git", "-C", absPath, "tag", "-d", tag)
		deleteOutput, err := deleteCmd.CombinedOutput()
		if err != nil {
			fmt.Printf("Warning: failed to delete tag %s: %v\n%s\n", tag, err, deleteOutput)
			// Continue with other tags even if one fails
		} else {
			fmt.Printf("Deleted tag %s in %s\n", tag, repoPath)
		}
	}

	fmt.Printf("Completed tag deletion for %s\n", repoPath)
	return nil
}

// FetchTags fetches all tags from the remote
func FetchTags(repoPath string) error {
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %v", err)
	}

	// Check if repository exists
	if _, err := os.Stat(filepath.Join(absPath, ".git")); os.IsNotExist(err) {
		return fmt.Errorf("not a git repository or directory does not exist")
	}

	// Fetch tags from remote
	fetchCmd := exec.Command("git", "-C", absPath, "fetch", "--tags")
	fetchOutput, err := fetchCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to fetch tags: %v\n%s", err, fetchOutput)
	}

	fmt.Printf("Successfully fetched tags in %s\n", repoPath)
	return nil
}

// ProcessTagsSequential deletes local tags and fetches remote tags for all repositories sequentially
func ProcessTagsSequential(repositories []config.Repository) {
	for _, repo := range repositories {
		fmt.Printf("Processing tags for %s\n", repo.Path)
		
		err := DeleteLocalTags(repo.Path)
		if err != nil {
			fmt.Printf("Error deleting tags in %s: %v\n", repo.Path, err)
			continue
		}
		
		err = FetchTags(repo.Path)
		if err != nil {
			fmt.Printf("Error fetching tags in %s: %v\n", repo.Path, err)
		}
	}
}

// ProcessTagsParallel deletes local tags and fetches remote tags for all repositories in parallel
func ProcessTagsParallel(repositories []config.Repository) {
	var wg sync.WaitGroup
	wg.Add(len(repositories))

	for _, repo := range repositories {
		go func(r config.Repository) {
			defer wg.Done()
			
			fmt.Printf("Processing tags for %s\n", r.Path)
			
			err := DeleteLocalTags(r.Path)
			if err != nil {
				fmt.Printf("Error deleting tags in %s: %v\n", r.Path, err)
				return
			}
			
			err = FetchTags(r.Path)
			if err != nil {
				fmt.Printf("Error fetching tags in %s: %v\n", r.Path, err)
			}
		}(repo)
	}

	wg.Wait()
}