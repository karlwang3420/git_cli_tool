// filepath: h:\code_base\git_cli_tool\cmd\switch.go
package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"git_cli_tool/config"
	"git_cli_tool/git"
	"git_cli_tool/log"

	"github.com/spf13/cobra"
)

var (
	autostash          string
	storeHistory       bool
	historyDescription string
	dryRun             bool
)

// switchCmd represents the switch command
var switchCmd = &cobra.Command{
	Use:   "switch",
	Short: "Switch branches based on configuration",
	Run:   runSwitchCmd,
}

// initSwitchCmd initializes the switch command with its flags
func initSwitchCmd() {
	switchCmd.Flags().StringVarP(&autostash, "autostash", "a", "", "Stash changes with the provided name before switching branches")
	switchCmd.Flags().BoolVar(&storeHistory, "store-history", true, "Store branch state in history before switching")
	switchCmd.Flags().StringVar(&historyDescription, "description", "", "Description for the history entry")
	switchCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview what branches would be switched to without making changes")
}

// runSwitchCmd is the main function for the switch command
func runSwitchCmd(cmd *cobra.Command, args []string) {
	// Read the configuration file
	configObj, err := config.ReadConfig(configFile)
	if err != nil {
		log.PrintError(log.ErrConfigReadFailed, "Error reading config", err)
		os.Exit(1)
	}

	// Ensure we have branches to switch to (check new field first, then legacy)
	configBranches := configObj.SwitchBranchesFallback
	if len(configBranches) == 0 {
		configBranches = configObj.Branches // backwards compatibility
	}

	if len(configBranches) == 0 && len(args) == 0 {
		log.PrintError(log.ErrNoConfigBranches, "No branches specified in the configuration file", nil)
		os.Exit(1)
	}

	// Get the repositories from the config
	repositories := configObj.FlattenRepositories()
	if len(repositories) == 0 {
		log.PrintError(log.ErrNoConfigRepos, "No repositories found in the configuration file", nil)
		os.Exit(1)
	}

	// Determine branches to try
	var branches []string
	if len(args) > 0 {
		branches = args
	} else {
		branches = configBranches
	}

	// Handle dry-run mode
	if dryRun {
		runDryRun(repositories, branches)
		return
	}

	// If recording history is enabled, save the current state
	if configObj.RecordHistory {
		_, history, err := config.ReadHistory()
		if err == nil || os.IsNotExist(err) {
			// Attempt to save the current state
			state, err := collectCurrentState(repositories)
			if err != nil {
				log.PrintWarning("Error saving branch history: " + err.Error())
			} else {
				config.SaveStateToHistory(state, history)
				log.PrintSuccess("Current branch state saved to history")
			}
		}
	}

	stashName := autostash
	stash := autostash != ""

	// If no stashName was provided, use first branch name
	if stash && stashName == "" && len(branches) > 0 {
		stashName = branches[0]
	}

	// Actually switch branches now
	log.PrintOperation("Switching repositories to branches: " + strings.Join(branches, ", "))
	log.PrintInfo("")

	// If stashing, remember which repositories had changes stashed
	stashedRepos := make(map[string]bool)

	// Perform the branch switching
	if stash {
		stashedRepos = git.SwitchBranchesWithStash(repositories, branches, stashName)
		_ = stashedRepos // used for history if needed
		log.PrintInfo("")
		log.PrintSuccess("Branch switch completed")
	} else {
		// Process with real-time output
		successCount := 0
		failCount := 0

		// Parallel execution with channel for results
		resultsChan := make(chan git.SwitchResult, len(repositories))
		
		// Launch goroutines
		for _, repo := range repositories {
			go func(r config.Repository) {
				resultsChan <- git.SwitchBranchWithResult(r.Path, branches)
			}(repo)
		}

		// Collect results as they come in
		for i := 0; i < len(repositories); i++ {
			result := <-resultsChan
			
			if result.Success {
				successCount++
				if result.AlreadyOnIt {
					log.PrintSuccess(fmt.Sprintf("%-30s %s → [ALREADY ON TARGET]", result.RepoName, result.ToBranch))
				} else if result.FromRemote {
					log.PrintSuccess(fmt.Sprintf("%-30s %s → %s (from remote)", result.RepoName, result.FromBranch, result.ToBranch))
				} else {
					log.PrintSuccess(fmt.Sprintf("%-30s %s → %s", result.RepoName, result.FromBranch, result.ToBranch))
				}
			} else {
				failCount++
				log.PrintWarning(fmt.Sprintf("%-30s %s → [FAILED: %s]", result.RepoName, result.FromBranch, result.Message))
			}
		}

		log.PrintInfo("")
		if failCount == 0 {
			log.PrintSuccess(fmt.Sprintf("All %d repositories switched successfully!", successCount))
		} else {
			log.PrintWarning(fmt.Sprintf("%d succeeded, %d failed", successCount, failCount))
		}
	}
}

// collectCurrentState collects the current branch state of all repositories
func collectCurrentState(repositories []config.Repository) (*config.BranchState, error) {
	state := &config.BranchState{
		Timestamp:    time.Now().Format(time.RFC3339),
		Description:  historyDescription,
		Repositories: make(map[string]config.RepositoryState),
	}

	for _, repo := range repositories {
		currentBranch, err := git.GetCurrentBranch(repo.Path)
		if err != nil {
			log.PrintWarning("Could not get current branch for " + repo.Path + ": " + err.Error())
			continue
		}

		state.Repositories[repo.Path] = config.RepositoryState{
			Branch:    currentBranch,
			StashName: "",
		}
	}

	return state, nil
}

// runDryRun performs a dry-run of the switch command, showing what would happen
func runDryRun(repositories []config.Repository, branches []string) {
	log.PrintOperation("Dry-run: Checking which branches would be used...")
	log.PrintInfo("")

	for _, repo := range repositories {
		repoName := filepath.Base(repo.Path)
		currentBranch, err := git.GetCurrentBranch(repo.Path)
		if err != nil {
			log.PrintErrorNoExit("", fmt.Sprintf("%-30s [ERROR: %s]", repoName, err.Error()), nil)
			continue
		}

		// Find which branch would be used
		targetBranch, source := findTargetBranch(repo.Path, branches)

		if targetBranch == "" {
			log.PrintWarning(fmt.Sprintf("%-30s %s → [NO MATCH] (none of %v found)", repoName, currentBranch, branches))
		} else if targetBranch == currentBranch {
			log.PrintSuccess(fmt.Sprintf("%-30s %s → [ALREADY ON TARGET]", repoName, currentBranch))
		} else {
			sourceInfo := ""
			if source == "remote" {
				sourceInfo = " (from remote)"
			}
			log.PrintInfo(fmt.Sprintf("%-30s %s → %s%s", repoName, currentBranch, targetBranch, sourceInfo))
		}
	}

	log.PrintInfo("")
	log.PrintOperation("Dry-run complete. No changes were made.")
}

// findTargetBranch finds which branch would be used for a repository
// Returns the branch name and source ("local" or "remote"), or empty string if none found
func findTargetBranch(repoPath string, branches []string) (string, string) {
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return "", ""
	}

	// Try each branch in order
	for _, branch := range branches {
		// Check if branch exists locally
		exists, err := git.CheckBranchExists(absPath, branch)
		if err == nil && exists {
			return branch, "local"
		}
	}

	// None found locally, try fetching and checking remote
	log.PrintDebug(fmt.Sprintf("Fetching remote for %s...", filepath.Base(repoPath)))
	fetchCmd := exec.Command("git", "-C", absPath, "fetch")
	fetchCmd.CombinedOutput() // Ignore errors, just try

	for _, branch := range branches {
		// Check if remote branch exists
		exists, err := git.CheckRemoteBranchExists(absPath, branch)
		if err == nil && exists {
			return branch, "remote"
		}
	}

	return "", ""
}

