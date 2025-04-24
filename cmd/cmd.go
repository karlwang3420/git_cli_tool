package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"

	"git_cli_tool/config"
	"git_cli_tool/git"
	
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gitswitch",
	Short: "Switch branches in multiple Git repositories",
	Long:  `A CLI tool that switches branches in multiple Git repositories based on a YAML configuration file.`,
}

// Initialize adds all child commands to the root command
func Initialize() {
	var configFile string
	var parallel bool
	var autostash string
	var storeHistory bool
	var historyDescription string
	var applyStashes bool

	// Switch command
	switchCmd := &cobra.Command{
		Use:   "switch",
		Short: "Switch branches based on configuration",
		Run: func(cmd *cobra.Command, args []string) {
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
		},
	}

	// List command
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List repositories and their current branches",
		Run: func(cmd *cobra.Command, args []string) {
			configObj, err := config.ReadConfig(configFile)
			if (err != nil) {
				fmt.Printf("Error reading config: %v\n", err)
				os.Exit(1)
			}

			// Get the flattened repositories
			repositories := configObj.FlattenRepositories()

			fmt.Println("Configured repositories:")
			fmt.Println("------------------------")
			for _, repo := range repositories {
				currentBranch, err := git.GetCurrentBranch(repo.Path)
				if err != nil {
					fmt.Printf("- %s: Error - %v\n", repo.Path, err)
				} else {
					preferredBranch := "none"
					if len(configObj.Branches) > 0 {
						preferredBranch = configObj.Branches[0]
					}
					status := "✓"
					if currentBranch != preferredBranch {
						status = "✗"
					}
					fmt.Printf("- %s: Current: %s, Preferred: %s %s\n", 
						repo.Path, 
						currentBranch, 
						preferredBranch,
						status)
				}
			}
			
			fmt.Println("\nBranch priority order:")
			for i, branch := range configObj.Branches {
				fmt.Printf("%d. %s\n", i+1, branch)
			}
		},
	}
	
	// Tags command
	tagsCmd := &cobra.Command{
		Use:   "tags",
		Short: "Delete local tags and fetch tags from remote for all repositories",
		Long:  `Delete all local tags and fetch remote tags for all repositories defined in the configuration file.`,
		Run: func(cmd *cobra.Command, args []string) {
			configObj, err := config.ReadConfig(configFile)
			if err != nil {
				fmt.Printf("Error reading config: %v\n", err)
				os.Exit(1)
			}

			// Get the flattened repositories
			repositories := configObj.FlattenRepositories()
			
			if len(repositories) == 0 {
				fmt.Println("No repositories found in the configuration file.")
				os.Exit(1)
			}

			fmt.Println("Refreshing tags in all repositories...")
			
			if parallel {
				git.ProcessTagsParallel(repositories)
			} else {
				git.ProcessTagsSequential(repositories)
			}
			
			fmt.Println("Tags refresh completed.")
		},
	}
	
	// History command - new command to list branch history
	historyCmd := &cobra.Command{
		Use:   "history",
		Short: "List branch history",
		Run: func(cmd *cobra.Command, args []string) {
			history, err := config.LoadBranchHistory()
			if err != nil {
				fmt.Printf("Error loading branch history: %v\n", err)
				os.Exit(1)
			}
			
			if len(history.States) == 0 {
				fmt.Println("No branch history found.")
				return
			}
			
			fmt.Println("Branch history:")
			fmt.Println("--------------")
			
			// Display history entries from newest to oldest
			for i := len(history.States) - 1; i >= 0; i-- {
				state := history.States[i]
				historyIndex := len(history.States) - 1 - i  // Reverse index for display
				
				fmt.Printf("[%d] %s", historyIndex, state.Timestamp)
				if state.Description != "" {
					fmt.Printf(" - %s", state.Description)
				}
				fmt.Println()
				
				// Display a summary of branches in this state
				repoCount := len(state.Repositories)
				if repoCount > 0 {
					fmt.Printf("    %d repositories, ", repoCount)
					
					// Count repositories with stashes
					stashCount := 0
					for _, repoState := range state.Repositories {
						if repoState.StashName != "" {
							stashCount++
						}
					}
					
					if stashCount > 0 {
						fmt.Printf("%d with stashes\n", stashCount)
					} else {
						fmt.Println("no stashes")
					}
				} else {
					fmt.Println("    No repository information")
				}
			}
			
			fmt.Println("\nUse 'gitswitch revert <index>' to revert to a specific state")
		},
	}
	
	// Revert command - new command to revert to a previous branch state
	revertCmd := &cobra.Command{
		Use:   "revert [index]",
		Short: "Revert to a previous branch state",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// Parse the history index argument
			index, err := strconv.Atoi(args[0])
			if err != nil {
				fmt.Printf("Invalid index: %v\n", err)
				os.Exit(1)
			}
			
			history, err := config.LoadBranchHistory()
			if err != nil {
				fmt.Printf("Error loading branch history: %v\n", err)
				os.Exit(1)
			}
			
			if len(history.States) == 0 {
				fmt.Println("No branch history found.")
				return
			}
			
			// Convert the user-provided index to the actual array index
			// User sees newest first (index 0), but array stores oldest first
			actualIndex := len(history.States) - 1 - index
			
			if actualIndex < 0 || actualIndex >= len(history.States) {
				fmt.Printf("Invalid index: %d (valid range: 0-%d)\n", index, len(history.States)-1)
				os.Exit(1)
			}
			
			// Get the state to revert to
			state := history.States[actualIndex]
			
			// Revert to the selected state
			err = git.RevertToState(state, applyStashes)
			if err != nil {
				fmt.Printf("Error during revert: %v\n", err)
				os.Exit(1)
			}
			
			fmt.Printf("Successfully reverted to state [%d] from %s\n", index, state.Timestamp)
			if state.Description != "" {
				fmt.Printf("Description: %s\n", state.Description)
			}
		},
	}

	// Add flags to commands
	switchCmd.Flags().StringVarP(&configFile, "config", "c", "gitswitch.yml", "Path to configuration file")
	switchCmd.Flags().BoolVarP(&parallel, "parallel", "p", false, "Switch branches in parallel")
	switchCmd.Flags().StringVarP(&autostash, "autostash", "a", "", "Stash changes with the provided name before switching branches")
	switchCmd.Flags().BoolVar(&storeHistory, "store-history", true, "Store branch state in history before switching")
	switchCmd.Flags().StringVar(&historyDescription, "description", "", "Description for the history entry")
	
	listCmd.Flags().StringVarP(&configFile, "config", "c", "gitswitch.yml", "Path to configuration file")
	
	tagsCmd.Flags().StringVarP(&configFile, "config", "c", "gitswitch.yml", "Path to configuration file")
	tagsCmd.Flags().BoolVarP(&parallel, "parallel", "p", false, "Process tags in parallel")
	
	revertCmd.Flags().BoolVar(&applyStashes, "apply-stashes", true, "Apply stashes when reverting (if any exist)")
	
	// Add commands to root command
	rootCmd.AddCommand(switchCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(tagsCmd)
	rootCmd.AddCommand(historyCmd)
	rootCmd.AddCommand(revertCmd)
}

// Execute executes the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}