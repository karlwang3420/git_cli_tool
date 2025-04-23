package main

import (
	"git_cli_tool/cmd"
)

func main() {
	// Initialize and execute the root command
	cmd.Initialize()
	cmd.Execute()
}