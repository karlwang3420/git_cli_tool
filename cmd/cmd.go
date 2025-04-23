package cmd

import (
	"fmt"
	"os"

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

	// Switch command
	switchCmd := &cobra.Command{
		Use:   "switch",
		Short: "Switch branches based on configuration",
		Run: func(cmd *cobra.Command, args []string) {
			config, err := config.ReadConfig(configFile)
			if err != nil {
				fmt.Printf("Error reading config: %v\n", err)
				os.Exit(1)
			}

			if len(config.Branches) == 0 {
				fmt.Println("No branches specified in the configuration file.")
				os.Exit(1)
			}

			// Get the flattened repositories
			repositories := config.FlattenRepositories()
			
			if len(repositories) == 0 {
				fmt.Println("No repositories found in the configuration file.")
				os.Exit(1)
			}

			if parallel {
				git.SwitchBranchesParallel(repositories, config.Branches)
			} else {
				git.SwitchBranchesSequential(repositories, config.Branches)
			}
		},
	}

	// List command
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List repositories and their current branches",
		Run: func(cmd *cobra.Command, args []string) {
			config, err := config.ReadConfig(configFile)
			if (err != nil) {
				fmt.Printf("Error reading config: %v\n", err)
				os.Exit(1)
			}

			// Get the flattened repositories
			repositories := config.FlattenRepositories()

			fmt.Println("Configured repositories:")
			fmt.Println("------------------------")
			for _, repo := range repositories {
				currentBranch, err := git.GetCurrentBranch(repo.Path)
				if err != nil {
					fmt.Printf("- %s: Error - %v\n", repo.Path, err)
				} else {
					preferredBranch := "none"
					if len(config.Branches) > 0 {
						preferredBranch = config.Branches[0]
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
			for i, branch := range config.Branches {
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
			config, err := config.ReadConfig(configFile)
			if err != nil {
				fmt.Printf("Error reading config: %v\n", err)
				os.Exit(1)
			}

			// Get the flattened repositories
			repositories := config.FlattenRepositories()
			
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

	// Add flags to commands
	switchCmd.Flags().StringVarP(&configFile, "config", "c", "gitswitch.yml", "Path to configuration file")
	switchCmd.Flags().BoolVarP(&parallel, "parallel", "p", false, "Switch branches in parallel")
	listCmd.Flags().StringVarP(&configFile, "config", "c", "gitswitch.yml", "Path to configuration file")
	tagsCmd.Flags().StringVarP(&configFile, "config", "c", "gitswitch.yml", "Path to configuration file")
	tagsCmd.Flags().BoolVarP(&parallel, "parallel", "p", false, "Process tags in parallel")
	
	// Add commands to root command
	rootCmd.AddCommand(switchCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(tagsCmd)
}

// Execute executes the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}