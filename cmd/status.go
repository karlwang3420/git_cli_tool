// filepath: git_cli_tool/cmd/status.go
package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"git_cli_tool/config"
	"git_cli_tool/log"

	"github.com/spf13/cobra"
)

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show quick status of repositories with changes or sync issues",
	Long: `Show a quick overview of repositories that have uncommitted changes,
are ahead/behind their remote tracking branch, or have other notable states.

This is a filtered view - clean repositories that are in sync are not shown.

Example:
  git_cli_tool status
  git_cli_tool status --all   # Show all repositories, not just those with issues`,
	Run: runStatusCmd,
}

var showAll bool

// initStatusCmd initializes the status command with its flags
func initStatusCmd() {
	statusCmd.Flags().BoolVar(&showAll, "all", false, "Show all repositories, not just those with issues")
}

// RepoStatus holds the status information for a repository
type RepoStatus struct {
	Path            string
	Branch          string
	HasChanges      bool
	UntrackedFiles  int
	StagedChanges   int
	UnstagedChanges int
	Ahead           int
	Behind          int
	Error           string
}

// runStatusCmd is the main function for the status command
func runStatusCmd(cmd *cobra.Command, args []string) {
	configObj, err := config.ReadConfig(configFile)
	if err != nil {
		log.PrintError(log.ErrConfigReadFailed, "Error reading config", err)
		os.Exit(1)
	}

	repositories := configObj.FlattenRepositories()
	if len(repositories) == 0 {
		log.PrintError(log.ErrNoConfigRepos, "No repositories found in the configuration file", nil)
		os.Exit(1)
	}

	log.PrintOperation("Checking repository status...")

	var statuses []RepoStatus
	issueCount := 0

	for _, repo := range repositories {
		status := getRepoStatus(repo.Path)
		statuses = append(statuses, status)

		if status.Error != "" || status.HasChanges || status.Ahead > 0 || status.Behind > 0 {
			issueCount++
		}
	}

	// Print results
	if issueCount == 0 && !showAll {
		log.PrintSuccess(fmt.Sprintf("All %d repositories are clean and in sync!", len(repositories)))
		return
	}

	log.PrintInfo("")
	for _, status := range statuses {
		hasIssue := status.Error != "" || status.HasChanges || status.Ahead > 0 || status.Behind > 0

		if !showAll && !hasIssue {
			continue
		}

		printRepoStatus(status)
	}

	log.PrintInfo("")
	if issueCount > 0 {
		log.PrintWarning(fmt.Sprintf("%d of %d repositories need attention", issueCount, len(repositories)))
	} else {
		log.PrintSuccess(fmt.Sprintf("All %d repositories are clean and in sync!", len(repositories)))
	}
}

func getRepoStatus(repoPath string) RepoStatus {
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return RepoStatus{Path: repoPath, Error: "failed to resolve path"}
	}

	status := RepoStatus{Path: absPath}

	// Check if it's a git repository
	if _, err := os.Stat(filepath.Join(absPath, ".git")); os.IsNotExist(err) {
		status.Error = "not a git repository"
		return status
	}

	// Get current branch
	branchCmd := exec.Command("git", "-C", absPath, "rev-parse", "--abbrev-ref", "HEAD")
	branchOutput, err := branchCmd.CombinedOutput()
	if err != nil {
		status.Error = "failed to get branch"
		return status
	}
	status.Branch = strings.TrimSpace(string(branchOutput))

	// Get status --porcelain for changes
	statusCmd := exec.Command("git", "-C", absPath, "status", "--porcelain")
	statusOutput, err := statusCmd.CombinedOutput()
	if err != nil {
		status.Error = "failed to get status"
		return status
	}

	// Parse status output
	lines := strings.Split(strings.TrimSpace(string(statusOutput)), "\n")
	for _, line := range lines {
		if len(line) < 2 {
			continue
		}
		status.HasChanges = true
		indexStatus := line[0]
		workTreeStatus := line[1]

		if indexStatus == '?' {
			status.UntrackedFiles++
		} else if indexStatus != ' ' {
			status.StagedChanges++
		}

		if workTreeStatus != ' ' && workTreeStatus != '?' {
			status.UnstagedChanges++
		}
	}

	// Get ahead/behind count
	revListCmd := exec.Command("git", "-C", absPath, "rev-list", "--left-right", "--count", "@{upstream}...HEAD")
	revOutput, err := revListCmd.CombinedOutput()
	if err == nil {
		parts := strings.Fields(strings.TrimSpace(string(revOutput)))
		if len(parts) == 2 {
			fmt.Sscanf(parts[0], "%d", &status.Behind)
			fmt.Sscanf(parts[1], "%d", &status.Ahead)
		}
	}
	// If error, it might not have an upstream - that's ok, leave ahead/behind as 0

	return status
}

func printRepoStatus(status RepoStatus) {
	// Get just the repo name for display
	repoName := filepath.Base(status.Path)

	if status.Error != "" {
		log.PrintErrorNoExit("", fmt.Sprintf("%-30s [ERROR: %s]", repoName, status.Error), nil)
		return
	}

	// Build status line
	var parts []string

	// Branch info
	branchInfo := fmt.Sprintf("on %s", status.Branch)
	parts = append(parts, branchInfo)

	// Changes info
	if status.HasChanges {
		var changesParts []string
		if status.StagedChanges > 0 {
			changesParts = append(changesParts, fmt.Sprintf("%d staged", status.StagedChanges))
		}
		if status.UnstagedChanges > 0 {
			changesParts = append(changesParts, fmt.Sprintf("%d unstaged", status.UnstagedChanges))
		}
		if status.UntrackedFiles > 0 {
			changesParts = append(changesParts, fmt.Sprintf("%d untracked", status.UntrackedFiles))
		}
		parts = append(parts, strings.Join(changesParts, ", "))
	}

	// Ahead/behind info
	if status.Ahead > 0 || status.Behind > 0 {
		var syncParts []string
		if status.Ahead > 0 {
			syncParts = append(syncParts, fmt.Sprintf("↑%d ahead", status.Ahead))
		}
		if status.Behind > 0 {
			syncParts = append(syncParts, fmt.Sprintf("↓%d behind", status.Behind))
		}
		parts = append(parts, strings.Join(syncParts, ", "))
	}

	// Determine color/status
	if status.HasChanges || status.Ahead > 0 || status.Behind > 0 {
		log.PrintWarning(fmt.Sprintf("%-30s %s", repoName, strings.Join(parts, " | ")))
	} else {
		log.PrintSuccess(fmt.Sprintf("%-30s %s | clean", repoName, parts[0]))
	}
}
