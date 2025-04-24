package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"time"

	"gopkg.in/yaml.v3"
)

// Configuration represents the YAML configuration file structure
type Configuration struct {
	Branches     []string                 `yaml:"branches"`
	Repositories []map[string][]string    `yaml:"repositories"`
}

// Repository represents a Git repository configuration
type Repository struct {
	Path string
}

// FlattenRepositories converts the hierarchical parent-subfolders structure 
// into a flat list of Repository objects with full paths
func (c *Configuration) FlattenRepositories() []Repository {
	var flatRepos []Repository

	for _, parentRepoMap := range c.Repositories {
		for parentPath, subFolders := range parentRepoMap {
			for _, subFolder := range subFolders {
				fullPath := filepath.Join(parentPath, subFolder)
				flatRepos = append(flatRepos, Repository{Path: fullPath})
			}
		}
	}

	return flatRepos
}

// ReadConfig reads and parses the configuration file
func ReadConfig(configPath string) (*Configuration, error) {
	absPath, err := filepath.Abs(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve absolute path: %v", err)
	}

	// Read the file as plain text first
	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	// Convert content to string
	content := string(data)

	// On Windows, ensure backslashes in paths are properly handled
	// by doubling them in the YAML content before parsing
	if filepath.Separator == '\\' {
		// Use regex to find paths in the format "X:\path\to\something"
		re := regexp.MustCompile(`"([A-Za-z]:(?:\\[^"\\]+)+)"`)
		content = re.ReplaceAllStringFunc(content, func(match string) string {
			// Remove the surrounding quotes
			path := match[1 : len(match)-1]
			// Convert to forward slashes which YAML handles better
			normalizedPath := filepath.ToSlash(path)
			// Return with quotes
			return `"` + normalizedPath + `"`
		})
	}

	// Now parse the modified content as YAML
	var config Configuration
	if err := yaml.Unmarshal([]byte(content), &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	return &config, nil
}

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
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %v", err)
	}
	exeDir := filepath.Dir(exePath)
	
	// Use history file in the same directory as the executable
	return filepath.Join(exeDir, "gitswitch-history.yml"), nil
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
		// Use empty string for branch if there's an error getting it
		// The error will be logged elsewhere
		branchName := ""
		
		// We'll retry getting the current branch with standard git command if needed
		gitCmd := fmt.Sprintf("git -C \"%s\" rev-parse --abbrev-ref HEAD", repo.Path)
		cmdOut, err := execCommand("cmd", "/c", gitCmd)
		if err == nil {
			branchName = string(cmdOut)
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

// Helper function to execute commands
func execCommand(command string, args ...string) ([]byte, error) {
	cmd := exec.Command(command, args...)
	return cmd.Output()
}