// filepath: h:\code_base\git_cli_tool\cmd\tags.go
package cmd

import (
	"os"

	"git_cli_tool/config"
	"git_cli_tool/git"
	"git_cli_tool/log"

	"github.com/spf13/cobra"
)

// tagsCmd represents the tags command
var tagsCmd = &cobra.Command{
	Use:   "tags",
	Short: "Delete local tags and fetch tags from remote for all repositories",
	Long:  `Delete all local tags and fetch remote tags for all repositories defined in the configuration file.`,
	Run:   runTagsCmd,
}

// initTagsCmd initializes the tags command with its flags
func initTagsCmd() {
	// The tags command already has access to the global configFile and parallel flags
}

// runTagsCmd is the main function for the tags command
func runTagsCmd(cmd *cobra.Command, args []string) {
	// Read the configuration file
	configObj, err := config.ReadConfig(configFile)
	if err != nil {
		log.PrintError(log.ErrConfigReadFailed, "Error reading config", err)
		os.Exit(1)
	}

	// Get the flattened repositories
	repositories := configObj.FlattenRepositories()

	if len(repositories) == 0 {
		log.PrintError(log.ErrNoConfigRepos, "No repositories found in the configuration file", nil)
		os.Exit(1)
	}

	log.PrintOperation("Refreshing tags in all repositories")

	if parallel {
		git.ProcessTagsParallel(repositories)
	} else {
		git.ProcessTagsSequential(repositories)
	}

	log.PrintSuccess("Tags refresh completed")
}
