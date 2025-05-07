// filepath: h:\code_base\git_cli_tool\git\tags.go
package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"git_cli_tool/config"
	"git_cli_tool/log"
)

// DeleteTags deletes all tags in a repository
func DeleteTags(repoPath string) error {
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %v", err)
	}

	// Get all tags
	cmd := exec.Command("git", "-C", absPath, "tag")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to list tags: %v", err)
	}

	tags := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(tags) == 0 || (len(tags) == 1 && tags[0] == "") {
		log.PrintInfo(fmt.Sprintf("No tags to delete in %s", repoPath))
		return nil
	}

	log.PrintInfo(fmt.Sprintf("Found %d tags to delete in %s", len(tags), repoPath))

	// Delete each tag locally
	for _, tag := range tags {
		if tag == "" {
			continue
		}

		deleteCmd := exec.Command("git", "-C", absPath, "tag", "-d", tag)
		deleteOutput, err := deleteCmd.CombinedOutput()
		if err != nil {
			log.PrintWarning(fmt.Sprintf("Error deleting tag %s: %v\n%s", tag, err, deleteOutput))
		} else {
			log.PrintInfo(fmt.Sprintf("Deleted tag %s in %s", tag, repoPath))
		}
	}

	log.PrintInfo(fmt.Sprintf("Completed tag deletion for %s", repoPath))
	return nil
}

// FetchTags fetches all tags from the remote
func FetchTags(repoPath string) error {
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %v", err)
	}

	// Check if repository exists
	if _, err := os.Stat(filepath.Join(absPath, ".git")); os.IsNotExist(err) {
		return fmt.Errorf("not a git repository or directory does not exist")
	}

	// Fetch tags from remote
	fetchCmd := exec.Command("git", "-C", absPath, "fetch", "--tags")
	fetchOutput, err := fetchCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to fetch tags: %v\n%s", err, fetchOutput)
	}

	log.PrintSuccess(fmt.Sprintf("Successfully fetched tags in %s", repoPath))
	return nil
}

// ProcessTagsSequential deletes local tags and fetches remote tags for all repositories sequentially
func ProcessTagsSequential(repositories []config.Repository) {
	for _, repo := range repositories {
		log.PrintOperation(fmt.Sprintf("Processing tags for %s", repo.Path))

		err := DeleteTags(repo.Path)
		if err != nil {
			log.PrintErrorNoExit(log.ErrGitTagOperationFailed, fmt.Sprintf("Error deleting tags in %s", repo.Path), err)
			continue
		}

		err = FetchTags(repo.Path)
		if err != nil {
			log.PrintErrorNoExit(log.ErrGitTagOperationFailed, fmt.Sprintf("Error fetching tags in %s", repo.Path), err)
		}
	}
}

// ProcessTagsParallel deletes local tags and fetches remote tags for all repositories in parallel
func ProcessTagsParallel(repositories []config.Repository) {
	var wg sync.WaitGroup
	wg.Add(len(repositories))

	for _, repo := range repositories {
		go func(r config.Repository) {
			defer wg.Done()

			log.PrintOperation(fmt.Sprintf("Processing tags for %s", r.Path))

			err := DeleteTags(r.Path)
			if err != nil {
				log.PrintErrorNoExit(log.ErrGitTagOperationFailed, fmt.Sprintf("Error deleting tags in %s", r.Path), err)
				return
			}

			err = FetchTags(r.Path)
			if err != nil {
				log.PrintErrorNoExit(log.ErrGitTagOperationFailed, fmt.Sprintf("Error fetching tags in %s", r.Path), err)
			}
		}(repo)
	}

	wg.Wait()
}
