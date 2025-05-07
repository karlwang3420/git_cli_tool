// filepath: h:\code_base\git_cli_tool\cmd\switch.go
package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"

	"git_cli_tool/config"
	"git_cli_tool/git"
	
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
		fmt.Printf("Error reading config: %v\n", err)
		os.Exit(1)
	}

	// Ensure we have branches to switch to
	if len(configObj.DefaultBranches) == 0 && len(args) == 0 {
		fmt.Println("No branches specified in the configuration file.")
		os.Exit(1)
	}

	// Get the repositories from the config
	repositories := configObj.FlattenRepositories()
	if len(repositories) == 0 {
		fmt.Println("No repositories found in the configuration file.")
		os.Exit(1)
	}

	// Determine branches to try
	var branches []string
	if len(args) > 0 {
		branches = args
	} else {
		branches = configObj.DefaultBranches
	}

	// If recording history is enabled, save the current state
	if configObj.RecordHistory && !dryRun {
		_, history, err := config.ReadHistory()
		if err == nil || os.IsNotExist(err) {
			// Attempt to save the current state
			state, err := collectCurrentState(repositories)
			if err != nil {
				fmt.Printf("Error saving branch history: %v\n", err)
			} else {
				config.SaveStateToHistory(state, history)
				fmt.Println("Current branch state saved to history")
			}
		}
	}

	description := description
	stashName := stashName
	
	// If no description was provided, use generic one
	if description == "" {
		description = fmt.Sprintf("Manual switch to %s", strings.Join(branches, ", "))
	}
	
	// If no stashName was provided, use first branch name
	if stash && stashName == "" && len(branches) > 0 {
		stashName = branches[0]
	}

	// If it's a dry run, just print what would happen
	if dryRun {
		fmt.Println("Dry run: Showing branch switching plan (not executing)")
		fmt.Printf("Would switch repositories to branches: %s\n", strings.Join(branches, ", "))
		if stash {
			fmt.Printf("Would stash changes with name: %s\n", stashName)
		}
		for _, repo := range repositories {
			fmt.Printf("Would process repository: %s\n", repo.Path)
		}
		return
	}
	
	// Actually switch branches now
	fmt.Printf("Switching repositories to branches: %s\n", strings.Join(branches, ", "))
	
	// If stashing, remember which repositories had changes stashed
	stashedRepos := make(map[string]bool)
	
	// Perform the branch switching
	if parallel {
		if stash {
			git.SwitchBranchesParallelWithStash(repositories, branches, stashName)
		} else {
			git.SwitchBranchesParallel(repositories, branches)
		}
	} else {
		if stash {
			git.SwitchBranchesSequentialWithStash(repositories, branches, stashName)
		} else {
			git.SwitchBranchesSequential(repositories, branches)
		}
	}
	
	fmt.Println("Branch switch completed.")
	
	// If recording history, save the post-switch state with stash information
	if configObj.RecordHistory && stash {
		_, history, err := config.ReadHistory()
		if err == nil || os.IsNotExist(err) {
			state, err := collectPostSwitchState(repositories, stashName, stashedRepos)
			if err != nil {
				fmt.Printf("Error saving post-switch branch history: %v\n", err)
			} else {
				state.Description = description
				config.SaveStateToHistory(state, history)
				fmt.Println("Post-switch branch state with stash information saved to history")
			}
		}
	}
}