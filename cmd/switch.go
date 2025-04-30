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
	configObj, err := config.ReadConfig(configFile)
	if err != nil {
		fmt.Printf("Error reading config: %v\n", err)
		os.Exit(1)
	}

	if len(configObj.Branches) == 0 {
		fmt.Println("No branches specified in the configuration file.")
		os.Exit(1)
	}

	// Get the flattened repositories
	repositories := configObj.FlattenRepositories()
	
	if len(repositories) == 0 {
		fmt.Println("No repositories found in the configuration file.")
		os.Exit(1)
	}

	// Create stash name map to track which repositories had stashes created
	stashNameByRepo := make(map[string]string)

	// Store current branch state before switching if requested
	if storeHistory {
		if historyDescription == "" {
			historyDescription = fmt.Sprintf("Switch to %s", configObj.Branches[0])
		}
		
		// Save current state before switching
		_, err := config.CreateBranchStateSnapshot(repositories, historyDescription, nil)
		if err != nil {
			fmt.Printf("Warning: Failed to save branch history: %v\n", err)
		} else {
			fmt.Println("Current branch state saved to history")
		}
	}

	if parallel {
		if autostash != "" {
			// For parallel stashing, we need to capture stash names for history
			if storeHistory {
				var wg sync.WaitGroup
				wg.Add(len(repositories))
				
				for _, repo := range repositories {
					go func(r config.Repository) {
						defer wg.Done()
						
						// Check if there are changes to stash
						statusCmd := exec.Command("git", "-C", r.Path, "status", "--porcelain")
						statusOutput, err := statusCmd.CombinedOutput()
						if err == nil && len(strings.TrimSpace(string(statusOutput))) > 0 {
							// Only record repositories that will actually have a stash created
							stashNameByRepo[r.Path] = autostash
						}
						
						// The actual stashing happens in the SwitchBranchesParallelWithStash function
					}(repo)
				}
				
				wg.Wait()
			}
			
			git.SwitchBranchesParallelWithStash(repositories, configObj.Branches, autostash)
		} else {
			git.SwitchBranchesParallel(repositories, configObj.Branches)
		}
	} else {
		if autostash != "" {
			// For sequential stashing, we can capture stash names as we go
			for _, repo := range repositories {
				// Check if there are changes to stash
				statusCmd := exec.Command("git", "-C", repo.Path, "status", "--porcelain")
				statusOutput, err := statusCmd.CombinedOutput()
				if err == nil && len(strings.TrimSpace(string(statusOutput))) > 0 {
					// Only record repositories that will actually have a stash created
					stashNameByRepo[repo.Path] = autostash
				}
			}
			
			git.SwitchBranchesSequentialWithStash(repositories, configObj.Branches, autostash)
		} else {
			git.SwitchBranchesSequential(repositories, configObj.Branches)
		}
	}
	
	// Store branch state after switching if history is enabled and stashing was used
	if storeHistory && autostash != "" {
		// Create a post-switch snapshot that includes stash information
		postDescription := fmt.Sprintf("After switch to %s with stash '%s'", configObj.Branches[0], autostash)
		_, err := config.CreateBranchStateSnapshot(repositories, postDescription, stashNameByRepo)
		if err != nil {
			fmt.Printf("Warning: Failed to save post-switch branch history: %v\n", err)
		} else {
			fmt.Println("Post-switch branch state with stash information saved to history")
		}
	}
}