// filepath: h:\code_base\git_cli_tool\cmd\list.go
package cmd

import (
	"fmt"
	"os"

	"git_cli_tool/config"
	"git_cli_tool/git"
	
	"github.com/spf13/cobra"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List repositories and their current branches",
	Run:   runListCmd,
}

// initListCmd initializes the list command with its flags
func initListCmd() {
	// The list command already has access to the global configFile flag
}

// runListCmd is the main function for the list command
func runListCmd(cmd *cobra.Command, args []string) {
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
}