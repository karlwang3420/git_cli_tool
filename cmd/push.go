package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"git_cli_tool/config"
	"git_cli_tool/git"
	"git_cli_tool/log"

	"github.com/spf13/cobra"
)

// PushResult holds the result of pushing a single repository
type PushResult struct {
	RepoPath    string
	RepoName    string
	Branch      string
	Success     bool
	Message     string
	Published   bool
}

// pushCmd represents the push command
var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push all repositories to remote",
	Long: `Push all repositories to their remote origins in parallel.
If the current branch has no upstream, it will be published (set upstream).

Example:
  git_cli_tool push`,
	Run: runPushCmd,
}

// initPushCmd initializes the push command with its flags
func initPushCmd() {
	// No specific flags needed for push command
}

// runPushCmd is the main function for the push command
func runPushCmd(cmd *cobra.Command, args []string) {
	configObj, err := config.ReadConfig(configFile)
	if err != nil {
		log.PrintError(log.ErrConfigReadFailed, "Error reading config", err)
		os.Exit(1)
	}

	repositories := configObj.FlattenRepositories()
	if len(repositories) == 0 {
		log.PrintError(log.ErrNoConfigRepos, "No repositories found in the configuration file", nil)
		os.Exit(1)
	}

	log.PrintOperation("Pushing all repositories to remote")
	log.PrintInfo("")

	resultsChan := make(chan PushResult, len(repositories))

	// Launch goroutines for parallel push
	for _, repo := range repositories {
		go func(r config.Repository) {
			resultsChan <- pushRepository(r.Path)
		}(repo)
	}

	// Collect results
	successCount := 0
	failCount := 0

	for i := 0; i < len(repositories); i++ {
		result := <-resultsChan

		if result.Success {
			successCount++
			if result.Published {
				log.PrintSuccess(fmt.Sprintf("%-30s %s (published)", result.RepoName, result.Branch))
			} else {
				log.PrintSuccess(fmt.Sprintf("%-30s %s", result.RepoName, result.Branch))
			}
		} else {
			failCount++
			log.PrintWarning(fmt.Sprintf("%-30s [FAILED: %s]", result.RepoName, result.Message))
		}
	}

	log.PrintInfo("")
	if failCount == 0 {
		log.PrintSuccess(fmt.Sprintf("All %d repositories pushed successfully!", successCount))
	} else {
		log.PrintWarning(fmt.Sprintf("%d succeeded, %d failed", successCount, failCount))
	}
}

// pushRepository pushes a single repository
func pushRepository(repoPath string) PushResult {
	absPath, err := filepath.Abs(repoPath)
	repoName := filepath.Base(repoPath)

	result := PushResult{
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
	branch, err := git.GetCurrentBranch(absPath)
	if err != nil {
		result.Message = "failed to get current branch"
		return result
	}
	result.Branch = branch

	// Check if upstream is set
	upstreamCmd := exec.Command("git", "-C", absPath, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}")
	upstreamOutput, upstreamErr := upstreamCmd.CombinedOutput()

	if upstreamErr != nil || strings.TrimSpace(string(upstreamOutput)) == "" {
		// No upstream set, publish the branch
		pushCmd := exec.Command("git", "-C", absPath, "push", "-u", "origin", branch)
		output, err := pushCmd.CombinedOutput()
		if err != nil {
			result.Message = strings.TrimSpace(string(output))
			if result.Message == "" {
				result.Message = err.Error()
			}
			return result
		}
		result.Success = true
		result.Published = true
		result.Message = "published and pushed"
		return result
	}

	// Upstream exists, regular push
	pushCmd := exec.Command("git", "-C", absPath, "push")
	output, err := pushCmd.CombinedOutput()
	if err != nil {
		result.Message = strings.TrimSpace(string(output))
		if result.Message == "" {
			result.Message = err.Error()
		}
		return result
	}

	result.Success = true
	outputStr := strings.TrimSpace(string(output))
	if strings.Contains(outputStr, "Everything up-to-date") {
		result.Message = "up to date"
	} else {
		result.Message = "pushed"
	}

	return result
}

// Mutex for thread-safe output
var pushOutputMutex sync.Mutex
