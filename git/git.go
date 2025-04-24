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

// StashChanges stashes changes in a repository with the given name
func StashChanges(repoPath string, stashName string) error {
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %v", err)
	}

	// Check if repository exists
	if _, err := os.Stat(filepath.Join(absPath, ".git")); os.IsNotExist(err) {
		return fmt.Errorf("not a git repository or directory does not exist")
	}

	// Check if there are changes to stash
	statusCmd := exec.Command("git", "-C", absPath, "status", "--porcelain")
	statusOutput, err := statusCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to get git status: %v", err)
	}

	// If there are no changes, skip stashing
	if len(strings.TrimSpace(string(statusOutput))) == 0 {
		fmt.Printf("No changes to stash in %s\n", repoPath)
		return nil
	}

	// Create a detailed message with the stash name
	message := fmt.Sprintf("GitSwitch: %s", stashName)
	
	// Stash changes with the provided name, include untracked files
	// Use --include-untracked to ensure all files are included, even new ones
	stashCmd := exec.Command("git", "-C", absPath, "stash", "push", "--include-untracked", "-m", message)
	stashOutput, err := stashCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to stash changes: %v\n%s", err, stashOutput)
	}

	fmt.Printf("Successfully stashed changes in %s with message '%s'\n", repoPath, message)
	fmt.Printf("To view stashed changes: git -C \"%s\" stash list\n", absPath)
	fmt.Printf("To apply the stash: git -C \"%s\" stash apply\n", absPath)
	
	return nil
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

// ApplyStash applies a specific stash in a repository
func ApplyStash(repoPath string, stashName string) error {
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %v", err)
	}

	// Check if repository exists
	if _, err := os.Stat(filepath.Join(absPath, ".git")); os.IsNotExist(err) {
		return fmt.Errorf("not a git repository or directory does not exist")
	}

	// Find stash with matching name
	listCmd := exec.Command("git", "-C", absPath, "stash", "list")
	listOutput, err := listCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to list stashes: %v", err)
	}

	stashLines := strings.Split(string(listOutput), "\n")
	stashIndex := ""

	// Format of stash line: stash@{0}: On branch: message
	for _, line := range stashLines {
		if strings.Contains(line, stashName) {
			// Extract the stash index (e.g., stash@{0})
			parts := strings.SplitN(line, ":", 2)
			if len(parts) > 0 {
				stashIndex = strings.TrimSpace(parts[0])
				break
			}
		}
	}

	if stashIndex == "" {
		return fmt.Errorf("no stash found with name '%s'", stashName)
	}

	// Apply the stash
	applyCmd := exec.Command("git", "-C", absPath, "stash", "apply", stashIndex)
	applyOutput, err := applyCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to apply stash %s: %v\n%s", stashIndex, err, applyOutput)
	}

	fmt.Printf("Successfully applied stash %s in %s\n", stashIndex, repoPath)
	return nil
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
		trackOutput, err := trackCmd.CombinedOutput()
		if err != nil {
			// If branch creation fails, try direct checkout of remote branch
			checkoutCmd := exec.Command("git", "-C", absPath, "checkout", branch)
			checkoutOutput, err := checkoutCmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("failed to checkout remote branch %s: %v\n%s", branch, err, checkoutOutput)
			}
		} else {
			_ = trackOutput // Use output to avoid unused variable warning
		}
		fmt.Printf("Successfully switched to branch %s in %s\n", branch, repoPath)
		return nil
	}

	return fmt.Errorf("branch %s not found locally or remotely in %s", branch, repoPath)
}

// RevertToState reverts repositories to their state in a given BranchState, including stash application
func RevertToState(state config.BranchState, applyStashes bool) error {
	fmt.Printf("Reverting to branch state from %s\n", state.Timestamp)
	if state.Description != "" {
		fmt.Printf("Description: %s\n", state.Description)
	}
	
	for repoPath, repoState := range state.Repositories {
		// Skip empty branches (there was probably an error when recording it)
		if repoState.Branch == "" {
			fmt.Printf("Skipping %s: no branch recorded in history\n", repoPath)
			continue
		}
		
		// Switch to the recorded branch
		err := SwitchToBranch(repoPath, repoState.Branch)
		if err != nil {
			fmt.Printf("Error switching branch in %s: %v\n", repoPath, err)
			continue
		}
		
		// Apply stash if needed
		if applyStashes && repoState.StashName != "" {
			err := ApplyStash(repoPath, repoState.StashName)
			if err != nil {
				fmt.Printf("Error applying stash in %s: %v\n", repoPath, err)
			}
		}
	}
	
	return nil
}