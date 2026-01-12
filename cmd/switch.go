// filepath: h:\code_base\git_cli_tool\cmd\switch.go
package cmd

import (
	"fmt"
	"os"
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
}

// runSwitchCmd is the main function for the switch command
func runSwitchCmd(cmd *cobra.Command, args []string) {
	// Read the configuration file
	configObj, err := config.ReadConfig(configFile)
	if err != nil {
		log.PrintError(log.ErrConfigReadFailed, "Error reading config", err)
		os.Exit(1)
	}

	// Ensure we have branches to switch to
	if len(configObj.Branches) == 0 && len(args) == 0 {
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
		branches = configObj.Branches
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

	// If stashing, remember which repositories had changes stashed
	stashedRepos := make(map[string]bool)

	// Perform the branch switching
	if parallel {
		if stash {
			stashedRepos = git.SwitchBranchesParallelWithStash(repositories, branches, stashName)
			log.PrintDebug(fmt.Sprintf("Stashed repositories (parallel): %v", stashedRepos))
		} else {
			git.SwitchBranchesParallel(repositories, branches)
		}
	} else {
		if stash {
			stashedRepos = git.SwitchBranchesSequentialWithStash(repositories, branches, stashName)
			log.PrintDebug(fmt.Sprintf("Stashed repositories (sequential): %v", stashedRepos))
		} else {
			git.SwitchBranchesSequential(repositories, branches)
		}
	}

	log.PrintSuccess("Branch switch completed")
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
