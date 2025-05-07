// filepath: h:\code_base\git_cli_tool\git\util.go
package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"git_cli_tool/config"
)

// ValidateRepository checks if a path is a valid git repository
func ValidateRepository(repoPath string) error {
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %v", err)
	}

	// Check if repository exists
	if _, err := os.Stat(filepath.Join(absPath, ".git")); os.IsNotExist(err) {
		return fmt.Errorf("not a git repository or directory does not exist")
	}

	return nil
}

// RunGitCommand runs a git command in the specified repository path
func RunGitCommand(repoPath string, args ...string) (string, error) {
	if err := ValidateRepository(repoPath); err != nil {
		return "", err
	}

	// Add the -C flag and repository path to the beginning of the arguments
	cmdArgs := append([]string{"-C", repoPath}, args...)
	cmd := exec.Command("git", cmdArgs...)
	output, err := cmd.CombinedOutput()
	
	return string(output), err
}

// RevertToState reverts all repositories to the state described in the history
func RevertToState(state config.BranchState, applyStashes bool) error {
	fmt.Printf("Reverting to branch state from %s\n", state.Timestamp)
	
	if state.Description != "" {
		fmt.Printf("Description: %s\n", state.Description)
	}

	// Process each repository in state
	for repoPath, branchInfo := range state.Repositories {
		// Skip if there's no branch info (shouldn't happen, but just in case)
		if branchInfo.Branch == "" {
			fmt.Printf("Skipping %s: no branch recorded in history\n", repoPath)
			continue
		}
		
		// Switch to the recorded branch
		err := SwitchToBranch(repoPath, branchInfo.Branch)
		if err != nil {
			fmt.Printf("Error switching branch in %s: %v\n", repoPath, err)
			continue
		}
		
		// If there was a stash recorded and applyStashes is true, try to apply it
		if branchInfo.StashName != "" && applyStashes {
			err = ApplyStash(repoPath, branchInfo.StashName)
			if err != nil {
				fmt.Printf("Error applying stash in %s: %v\n", repoPath, err)
			}
		}
	}
	
	return nil
}