// filepath: h:\code_base\git_cli_tool\cmd\revert.go
package cmd

import (
	"fmt"
	"os"
	"strconv"

	"git_cli_tool/config"
	"git_cli_tool/git"
	
	"github.com/spf13/cobra"
)

var (
	applyStashes bool
)

// revertCmd represents the revert command
var revertCmd = &cobra.Command{
	Use:   "revert [index]",
	Short: "Revert to a previous branch state",
	Args:  cobra.ExactArgs(1),
	Run:   runRevertCmd,
}

// initRevertCmd initializes the revert command with its flags
func initRevertCmd() {
	revertCmd.Flags().BoolVar(&applyStashes, "apply-stashes", true, "Apply stashes when reverting (if any exist)")
}

// runRevertCmd is the main function for the revert command
func runRevertCmd(cmd *cobra.Command, args []string) {
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
}