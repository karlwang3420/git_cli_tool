// filepath: h:\code_base\git_cli_tool\config\history.go
package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// RepositoryState represents the state of a repository at a specific time
type RepositoryState struct {
	Branch    string `yaml:"branch"`
	StashName string `yaml:"stash,omitempty"` // Will be empty if no stash was created
}

// BranchState represents a snapshot of all repositories at a specific time
type BranchState struct {
	Timestamp    string                      `yaml:"timestamp"`
	Description  string                      `yaml:"description,omitempty"`
	Repositories map[string]RepositoryState  `yaml:"repositories"`
}

// BranchHistory stores the history of branch states
type BranchHistory struct {
	States []BranchState `yaml:"states"`
}

// GetHistoryFilePath returns the path to the branch history file
func GetHistoryFilePath() (string, error) {
	// Get executable directory
	exePath, err := os.Executable()
	if (err != nil) {
		return "", fmt.Errorf("failed to get executable path: %v", err)
	}
	exeDir := filepath.Dir(exePath)
	
	// Use history file in the same directory as the executable
	return filepath.Join(exeDir, "git_cli_tool-history.yml"), nil
}

// LoadBranchHistory loads the branch history from file
func LoadBranchHistory() (*BranchHistory, error) {
	historyPath, err := GetHistoryFilePath()
	if err != nil {
		return nil, err
	}

	// Check if file exists
	if _, err := os.Stat(historyPath); os.IsNotExist(err) {
		// If file doesn't exist, return an empty history
		return &BranchHistory{States: []BranchState{}}, nil
	}

	// Read file
	data, err := os.ReadFile(historyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read history file: %v", err)
	}

	// Unmarshal YAML
	var history BranchHistory
	if err := yaml.Unmarshal(data, &history); err != nil {
		return nil, fmt.Errorf("failed to parse history file: %v", err)
	}

	return &history, nil
}

// SaveBranchHistory saves the branch history to file
func SaveBranchHistory(history *BranchHistory) error {
	historyPath, err := GetHistoryFilePath()
	if err != nil {
		return err
	}

	// Marshal to YAML
	data, err := yaml.Marshal(history)
	if err != nil {
		return fmt.Errorf("failed to marshal history to YAML: %v", err)
	}

	// Write to file
	if err := os.WriteFile(historyPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write history file: %v", err)
	}

	return nil
}

// CreateBranchStateSnapshot creates a snapshot of the current branch state for all repositories
func CreateBranchStateSnapshot(repositories []Repository, description string, stashNameByRepo map[string]string) (*BranchState, error) {
	state := BranchState{
		Timestamp:    time.Now().Format(time.RFC3339),
		Description:  description,
		Repositories: make(map[string]RepositoryState),
	}

	for _, repo := range repositories {
		// Use the GetCurrentBranch function which properly trims the output
		branchName := ""
		currentBranch, err := GetCurrentBranch(repo.Path)
		if err == nil {
			branchName = currentBranch
		} else {
			// Fallback to direct command if the GetCurrentBranch function fails
			gitCmd := fmt.Sprintf("git -C \"%s\" rev-parse --abbrev-ref HEAD", repo.Path)
			cmdOut, cmdErr := execCommand("cmd", "/c", gitCmd)
			if cmdErr == nil {
				// Make sure to trim any whitespace or newlines
				branchName = strings.TrimSpace(string(cmdOut))
			}
		}
		
		stashName := ""
		if stashNameByRepo != nil {
			stashName = stashNameByRepo[repo.Path]
		}
		
		state.Repositories[repo.Path] = RepositoryState{
			Branch:    branchName,
			StashName: stashName,
		}
	}

	// Load existing history
	history, err := LoadBranchHistory()
	if err != nil {
		return nil, fmt.Errorf("failed to load branch history: %v", err)
	}

	// Add the new state to history
	history.States = append(history.States, state)

	// Save the updated history
	if err := SaveBranchHistory(history); err != nil {
		return nil, fmt.Errorf("failed to save branch history: %v", err)
	}

	return &state, nil
}

// SaveStateToHistory adds a branch state to history and saves it to file
func SaveStateToHistory(state *BranchState, history *BranchHistory) error {
	// Add the new state to history
	history.States = append(history.States, *state)

	// Save the updated history
	return SaveBranchHistory(history)
}

// ReadHistory loads the branch history from file
func ReadHistory() (string, *BranchHistory, error) {
	historyPath, err := GetHistoryFilePath()
	if err != nil {
		return "", nil, err
	}

	history, err := LoadBranchHistory()
	return historyPath, history, err
}

// Helper function to execute commands
func execCommand(command string, args ...string) ([]byte, error) {
	cmd := exec.Command(command, args...)
	return cmd.Output()
}

// GetCurrentBranch gets the current branch name of a repository
// This is a duplicate of the function in the git package to avoid import cycles
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