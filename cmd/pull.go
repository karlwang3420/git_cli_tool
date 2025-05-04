package cmd

import (
	"fmt"
	"os"

	"git_cli_tool/config"
	"git_cli_tool/git"
	
	"github.com/spf13/cobra"
)

// pullCmd represents the pull command
var pullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull latest changes from remote for all repositories",
	Long: `Pull the latest changes from remote repositories for all repositories
specified in the configuration file.

Example:
  git_cli_tool pull
  git_cli_tool pull --parallel`,
	Run: runPullCmd,
}

// initPullCmd initializes the pull command with its flags
func initPullCmd() {
	// No specific flags needed for pull command beyond the global ones
}

// runPullCmd is the main function for the pull command
func runPullCmd(cmd *cobra.Command, args []string) {
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

	fmt.Println("Pulling latest changes from remote repositories...")
	
	if parallel {
		git.PullRepositoriesParallel(repositories)
	} else {
		git.PullRepositoriesSequential(repositories)
	}
	
	fmt.Println("Pull operation completed.")
}