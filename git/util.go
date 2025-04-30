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