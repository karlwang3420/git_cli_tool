// filepath: h:\code_base\git_cli_tool\cmd\list.go
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"git_cli_tool/config"
	"git_cli_tool/git"
	"git_cli_tool/log"

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
	if err != nil {
		log.PrintError(log.ErrConfigReadFailed, "Error reading config", err)
		os.Exit(1)
	}

	// Get the flattened repositories
	repositories := configObj.FlattenRepositories()

	// Determine configured branches (same logic as switch command)
	configBranches := configObj.SwitchBranchesFallback
	if len(configBranches) == 0 {
		configBranches = configObj.Branches // backwards compatibility
	}

	// First branch in the list is the preferred one
	preferredBranch := ""
	if len(configBranches) > 0 {
		preferredBranch = configBranches[0]
	}

	log.PrintOperation("Repository Status")
	log.PrintInfo("")

	matchCount := 0
	mismatchCount := 0
	errorCount := 0

	// Column widths
	const repoWidth = 30
	const branchWidth = 40

	for _, repo := range repositories {
		repoName := filepath.Base(repo.Path)
		currentBranch, err := git.GetCurrentBranch(repo.Path)
		if err != nil {
			errorCount++
			log.PrintErrorNoExit("", fmt.Sprintf("%s [ERROR: %s]", padRight(repoName, repoWidth), err.Error()), nil)
		} else {
			onPreferred := preferredBranch != "" && currentBranch == preferredBranch
			repoPadded := padRight(repoName, repoWidth)
			branchPadded := padRight(currentBranch, branchWidth)
			if onPreferred {
				matchCount++
				log.PrintSuccess(fmt.Sprintf("%s on %s [ON TARGET]", repoPadded, branchPadded))
			} else {
				mismatchCount++
				targetInfo := ""
				if preferredBranch != "" {
					targetInfo = fmt.Sprintf(" (target: %s)", preferredBranch)
				}
				log.PrintWarning(fmt.Sprintf("%s on %s%s", repoPadded, branchPadded, targetInfo))
			}
		}
	}

	// Print summary
	log.PrintInfo("")
	if errorCount > 0 {
		log.PrintWarning(fmt.Sprintf("Summary: %d on target, %d off target, %d errors", matchCount, mismatchCount, errorCount))
	} else if mismatchCount > 0 {
		log.PrintWarning(fmt.Sprintf("Summary: %d on target, %d off target", matchCount, mismatchCount))
	} else {
		log.PrintSuccess(fmt.Sprintf("All %d repositories on target branch!", matchCount))
	}

	// Print branch priority if configured
	if len(configBranches) > 0 {
		log.PrintInfo("")
		log.PrintInfo("Branch priority: " + strings.Join(configBranches, " → "))
	}
}

// displayWidth returns the visual width of a string, accounting for wide (CJK) characters
func displayWidth(s string) int {
	width := 0
	for _, r := range s {
		if r >= 0x1100 && (r <= 0x115F || // Hangul Jamo
			r == 0x2329 || r == 0x232A || // Angle brackets
			(r >= 0x2E80 && r <= 0x303E) || // CJK Radicals, Kangxi Radicals, etc.
			(r >= 0x3040 && r <= 0xA4CF) || // Hiragana, Katakana, Bopomofo, etc.
			(r >= 0xAC00 && r <= 0xD7A3) || // Hangul Syllables
			(r >= 0xF900 && r <= 0xFAFF) || // CJK Compatibility Ideographs
			(r >= 0xFE10 && r <= 0xFE1F) || // Vertical Forms
			(r >= 0xFE30 && r <= 0xFE6F) || // CJK Compatibility Forms
			(r >= 0xFF00 && r <= 0xFF60) || // Fullwidth Forms
			(r >= 0xFFE0 && r <= 0xFFE6) || // Fullwidth Forms
			(r >= 0x20000 && r <= 0x2FFFF)) { // CJK Extension B and beyond
			width += 2
		} else {
			width += 1
		}
	}
	return width
}

// padRight pads a string with spaces to reach the target display width
func padRight(s string, targetWidth int) string {
	currentWidth := displayWidth(s)
	if currentWidth >= targetWidth {
		return s
	}
	return s + strings.Repeat(" ", targetWidth-currentWidth)
}
