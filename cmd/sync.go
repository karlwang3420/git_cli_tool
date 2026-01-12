// filepath: git_cli_tool/cmd/sync.go
package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"git_cli_tool/config"
	"git_cli_tool/git"
	"git_cli_tool/log"

	"github.com/spf13/cobra"
)

const defaultFallbackBranch = "main"

// syncCmd represents the sync command
var syncCmd = &cobra.Command{
	Use:   "sync <branch>",
	Short: "Sync a branch with its parent branch across all repositories",
	Long: `Switch to the specified branch and merge its parent branch into it.

This is useful when you have branches that depend on other branches.
For example, if 'feature/extension' depends on 'feature/base', running:
  git_cli_tool sync feature/extension

Will:
1. Switch to 'feature/extension' in each repository
2. Merge 'feature/base' into it (bringing in any new commits)

Parent branches are defined in the config file under 'branch_dependencies'.
If a parent branch is not found, it falls back to 'main' (or the configured fallback_branch).

Example config:
  branch_dependencies:
    "feature/extension": "feature/base"
    "feature/part2": "feature/part1"
  fallback_branch: "main"`,
	Args: cobra.ExactArgs(1),
	Run:  runSyncCmd,
}

// initSyncCmd initializes the sync command with its flags
func initSyncCmd() {
	// No specific flags needed for now
}

// SyncResult holds the result of syncing a single repository
type SyncResult struct {
	RepoPath     string
	RepoName     string
	TargetBranch string
	ParentBranch string
	Success      bool
	Message      string
	WasFallback  bool
}

// runSyncCmd is the main function for the sync command
func runSyncCmd(cmd *cobra.Command, args []string) {
	targetBranch := args[0]

	// Read configuration
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

	// Determine parent branch from config (using nested sync config)
	parentBranch := ""
	if configObj.Sync.BranchDependencies != nil {
		parentBranch = configObj.Sync.BranchDependencies[targetBranch]
	}

	// Determine fallback branch
	fallbackBranch := configObj.Sync.FallbackBranch
	if fallbackBranch == "" {
		fallbackBranch = defaultFallbackBranch
	}

	log.PrintOperation(fmt.Sprintf("Syncing branch '%s' across all repositories", targetBranch))
	if parentBranch != "" {
		log.PrintInfo(fmt.Sprintf("Parent branch: %s (fallback: %s)", parentBranch, fallbackBranch))
	} else {
		log.PrintInfo(fmt.Sprintf("No parent defined, will sync with: %s", fallbackBranch))
	}
	log.PrintInfo("")

	var results []SyncResult
	successCount := 0
	failCount := 0

	for _, repo := range repositories {
		result := syncRepository(repo.Path, targetBranch, parentBranch, fallbackBranch)
		results = append(results, result)

		if result.Success {
			successCount++
		} else {
			failCount++
		}
	}

	// Print summary
	log.PrintInfo("")
	log.PrintInfo("=== Sync Summary ===")
	for _, result := range results {
		if result.Success {
			syncInfo := fmt.Sprintf("merged %s", result.ParentBranch)
			if result.WasFallback {
				syncInfo += " (fallback)"
			}
			log.PrintSuccess(fmt.Sprintf("%-30s %s", result.RepoName, syncInfo))
		} else {
			log.PrintErrorNoExit("", fmt.Sprintf("%-30s %s", result.RepoName, result.Message), nil)
		}
	}

	log.PrintInfo("")
	if failCount == 0 {
		log.PrintSuccess(fmt.Sprintf("All %d repositories synced successfully!", successCount))
	} else {
		log.PrintWarning(fmt.Sprintf("%d succeeded, %d failed", successCount, failCount))
	}
}

// syncRepository syncs a single repository
func syncRepository(repoPath, targetBranch, parentBranch, fallbackBranch string) SyncResult {
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return SyncResult{
			RepoPath: repoPath,
			RepoName: filepath.Base(repoPath),
			Success:  false,
			Message:  "failed to resolve path",
		}
	}

	repoName := filepath.Base(absPath)
	result := SyncResult{
		RepoPath:     absPath,
		RepoName:     repoName,
		TargetBranch: targetBranch,
	}

	// Check if it's a git repository
	if _, err := os.Stat(filepath.Join(absPath, ".git")); os.IsNotExist(err) {
		result.Message = "not a git repository"
		return result
	}

	// Fetch from remote first
	log.PrintDebug(fmt.Sprintf("[%s] Fetching from remote...", repoName))
	fetchCmd := exec.Command("git", "-C", absPath, "fetch", "--all")
	fetchCmd.CombinedOutput() // Ignore fetch errors, continue anyway

	// Check if target branch exists (local or remote)
	targetExists, _ := git.CheckBranchExists(absPath, targetBranch)
	if !targetExists {
		// Check remote
		remoteExists, _ := git.CheckRemoteBranchExists(absPath, targetBranch)
		if !remoteExists {
			result.Message = fmt.Sprintf("branch '%s' not found", targetBranch)
			return result
		}
	}

	// Switch to target branch
	log.PrintDebug(fmt.Sprintf("[%s] Switching to %s...", repoName, targetBranch))
	err = git.SwitchBranchWithFallback(absPath, []string{targetBranch})
	if err != nil {
		result.Message = fmt.Sprintf("failed to switch to '%s': %v", targetBranch, err)
		return result
	}

	// Determine which parent branch to use
	branchToMerge := parentBranch
	useFallback := false

	if branchToMerge != "" {
		// Check if parent branch exists
		parentExists, _ := git.CheckBranchExists(absPath, branchToMerge)
		if !parentExists {
			remoteParentExists, _ := git.CheckRemoteBranchExists(absPath, branchToMerge)
			if !remoteParentExists {
				// Parent not found, use fallback
				branchToMerge = fallbackBranch
				useFallback = true
			}
		}
	} else {
		// No parent defined, use fallback
		branchToMerge = fallbackBranch
		useFallback = true
	}

	result.ParentBranch = branchToMerge
	result.WasFallback = useFallback

	// Make sure the branch we're merging from exists
	mergeExists, _ := git.CheckBranchExists(absPath, branchToMerge)
	if !mergeExists {
		remoteMergeExists, _ := git.CheckRemoteBranchExists(absPath, branchToMerge)
		if !remoteMergeExists {
			result.Message = fmt.Sprintf("branch '%s' not found to merge from", branchToMerge)
			return result
		}
		// Use remote version
		branchToMerge = "origin/" + branchToMerge
	}

	// Perform the merge
	log.PrintDebug(fmt.Sprintf("[%s] Merging %s...", repoName, branchToMerge))
	mergeCmd := exec.Command("git", "-C", absPath, "merge", branchToMerge, "--no-edit")
	mergeOutput, err := mergeCmd.CombinedOutput()
	
	if err != nil {
		// Check if it's a merge conflict
		if strings.Contains(string(mergeOutput), "CONFLICT") || strings.Contains(string(mergeOutput), "Automatic merge failed") {
			result.Message = "CONFLICT - resolve manually"
			// Leave conflicts in place for manual resolution
			return result
		}
		result.Message = fmt.Sprintf("merge failed: %s", strings.TrimSpace(string(mergeOutput)))
		return result
	}

	result.Success = true
	
	// Check if there were actually changes merged
	if strings.Contains(string(mergeOutput), "Already up to date") {
		result.Message = "already up to date"
	} else {
		result.Message = "merged successfully"
	}

	return result
}
