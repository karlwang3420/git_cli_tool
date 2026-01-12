// filepath: c:\Users\Karl\CodeBase\personal\git_cli_tool\git\tags.go
package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"git_cli_tool/config"
	"git_cli_tool/log"
)

// SyncTags synchronizes local tags with remote in a single optimized operation.
// This command:
// - Updates tags that point to different commits locally vs remote (--force)
// - Removes local tags that no longer exist on remote (--prune --prune-tags)
// - Fetches new tags from remote (--tags)
func SyncTags(repoPath string) error {
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %v", err)
	}

	// Check if repository exists
	if _, err := os.Stat(filepath.Join(absPath, ".git")); os.IsNotExist(err) {
		return fmt.Errorf("not a git repository or directory does not exist")
	}

	// Single command to sync all tags:
	// --tags: fetch all tags
	// --force: overwrite local tags that differ from remote
	// --prune: remove remote-tracking refs that no longer exist
	// --prune-tags: remove local tags that no longer exist on remote
	fetchCmd := exec.Command("git", "-C", absPath, "fetch", "--tags", "--force", "--prune", "--prune-tags")
	fetchOutput, err := fetchCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to sync tags: %v\n%s", err, fetchOutput)
	}

	log.PrintSuccess(fmt.Sprintf("Successfully synced tags in %s", repoPath))
	return nil
}

// ProcessTagsSequential syncs tags for all repositories sequentially
func ProcessTagsSequential(repositories []config.Repository) {
	for _, repo := range repositories {
		log.PrintOperation(fmt.Sprintf("Syncing tags for %s", repo.Path))

		err := SyncTags(repo.Path)
		if err != nil {
			log.PrintErrorNoExit(log.ErrGitTagOperationFailed, fmt.Sprintf("Error syncing tags in %s", repo.Path), err)
		}
	}
}

// ProcessTagsParallel syncs tags for all repositories in parallel
func ProcessTagsParallel(repositories []config.Repository) {
	var wg sync.WaitGroup
	wg.Add(len(repositories))

	for _, repo := range repositories {
		go func(r config.Repository) {
			defer wg.Done()

			log.PrintOperation(fmt.Sprintf("Syncing tags for %s", r.Path))

			err := SyncTags(r.Path)
			if err != nil {
				log.PrintErrorNoExit(log.ErrGitTagOperationFailed, fmt.Sprintf("Error syncing tags in %s", r.Path), err)
			}
		}(repo)
	}

	wg.Wait()
}
