// filepath: h:\code_base\git_cli_tool\cmd\history.go
package cmd

import (
	"fmt"
	"os"

	"git_cli_tool/config"
	"git_cli_tool/log"

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
		log.PrintError(log.ErrHistoryReadFailed, "Error loading branch history", err)
		os.Exit(1)
	}

	if len(history.States) == 0 {
		log.PrintInfo("No branch history found.")
		return
	}

	log.PrintInfo("Branch history:")
	log.PrintInfo("--------------")

	// Display history entries from newest to oldest
	for i := len(history.States) - 1; i >= 0; i-- {
		state := history.States[i]
		historyIndex := len(history.States) - 1 - i // Reverse index for display

		message := fmt.Sprintf("[%d] %s", historyIndex, state.Timestamp)
		if state.Description != "" {
			message += fmt.Sprintf(" - %s", state.Description)
		}
		log.PrintInfo(message)

		// Display a summary of branches in this state
		repoCount := len(state.Repositories)
		if repoCount > 0 {
			summaryMsg := fmt.Sprintf("    %d repositories", repoCount)

			// Count repositories with stashes
			stashCount := 0
			for _, repoState := range state.Repositories {
				if repoState.StashName != "" {
					stashCount++
				}
			}

			if stashCount > 0 {
				summaryMsg += fmt.Sprintf(", %d with stashes", stashCount)
			} else {
				summaryMsg += ", no stashes"
			}

			log.PrintInfo(summaryMsg)
		} else {
			log.PrintInfo("    No repository information")
		}
	}

	log.PrintInfo("\nUse 'git_cli_tool revert <index>' to revert to a specific state")
}
