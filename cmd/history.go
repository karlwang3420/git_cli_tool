// filepath: h:\code_base\git_cli_tool\cmd\history.go
package cmd

import (
	"fmt"
	"os"

	"git_cli_tool/config"
	
	"github.com/spf13/cobra"
)

// historyCmd represents the history command
var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "List branch history",
	Run:   runHistoryCmd,
}

// initHistoryCmd initializes the history command with its flags
func initHistoryCmd() {
	// No specific flags for history command
}

// runHistoryCmd is the main function for the history command
func runHistoryCmd(cmd *cobra.Command, args []string) {
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
}