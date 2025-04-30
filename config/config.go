package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

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