// filepath: h:\code_base\git_cli_tool\git\stash.go
package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

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