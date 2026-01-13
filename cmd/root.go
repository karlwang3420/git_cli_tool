// filepath: h:\code_base\git_cli_tool\cmd\root.go
package cmd

import (
	"fmt"
	"os"
	
	"github.com/spf13/cobra"
)

// Global flags used across multiple commands
var (
	configFile string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "git_cli_tool",
	Short: "Switch branches in multiple Git repositories",
	Long:  `A CLI tool that switches branches in multiple Git repositories based on a YAML configuration file.`,
}

// Initialize adds all child commands to the root command
func Initialize() {
	// Global flags
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "git_cli_tool.yml", "Path to configuration file")
	
	// Add all subcommands
	initSwitchCmd()
	initListCmd()
	initTagsCmd()
	initHistoryCmd()
	initRevertCmd()
	initPullCmd()
	initStatusCmd()
	initSyncCmd()
	
	// Add commands to root command
	rootCmd.AddCommand(switchCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(tagsCmd)
	rootCmd.AddCommand(historyCmd)
	rootCmd.AddCommand(revertCmd)
	rootCmd.AddCommand(pullCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(syncCmd)
}

// Execute executes the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}