// filepath: h:\code_base\git_cli_tool\cmd\revert.go
package cmd

import (
	"os"
	"strconv"

	"git_cli_tool/config"
	"git_cli_tool/git"
	"git_cli_tool/log"

	"github.com/spf13/cobra"
)

var (
	applyStashes bool
)

// revertCmd represents the revert command
var revertCmd = &cobra.Command{
	Use:   "revert [index]",
	Short: "Revert to a previous branch state (defaults to latest if no index provided)",
	Args:  cobra.MaximumNArgs(1),
	Run:   runRevertCmd,
}

// initRevertCmd initializes the revert command with its flags
func initRevertCmd() {
	revertCmd.Flags().BoolVar(&applyStashes, "apply-stashes", true, "Apply stashes when reverting (if any exist)")
}

// runRevertCmd is the main function for the revert command
func runRevertCmd(cmd *cobra.Command, args []string) {
	// Parse the history index argument, default to 0 (most recent) if not provided
	index := 0
	if len(args) > 0 {
		var err error
		index, err = strconv.Atoi(args[0])
		if err != nil {
			log.PrintError(log.ErrInvalidArgument, "Error parsing index", err)
			os.Exit(1)
		}
	}

	history, err := config.LoadBranchHistory()
	if err != nil {
		log.PrintError(log.ErrHistoryReadFailed, "Error loading branch history", err)
		os.Exit(1)
	}

	if len(history.States) == 0 {
		log.PrintInfo("No branch history found.")
		return
	}

	// Convert the user-provided index to the actual array index
	// User sees newest first (index 0), but array stores oldest first
	actualIndex := len(history.States) - 1 - index

	if actualIndex < 0 || actualIndex >= len(history.States) {
		log.PrintError(log.ErrHistoryIndexInvalid, "Invalid index", nil)
		log.PrintInfo("Valid range: 0-" + strconv.Itoa(len(history.States)-1))
		os.Exit(1)
	}

	// Get the state to revert to
	state := history.States[actualIndex]

	// Revert to the selected state
	err = git.RevertToState(state, applyStashes)
	if err != nil {
		log.PrintError(log.ErrOperationFailed, "Error during revert", err)
		os.Exit(1)
	}

	log.PrintSuccess("Successfully reverted to state [" + strconv.Itoa(index) + "] from " + state.Timestamp)
	if state.Description != "" {
		log.PrintInfo("Description: " + state.Description)
	}
}
