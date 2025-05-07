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
		fmt.Printf("No tags to delete in %s\n", repoPath)
		return nil
	}

	fmt.Printf("Found %d tags to delete in %s\n", len(tags), repoPath)

	// Delete each tag locally
	for _, tag := range tags {
		if tag == "" {
			continue
		}

		deleteCmd := exec.Command("git", "-C", absPath, "tag", "-d", tag)
		deleteOutput, err := deleteCmd.CombinedOutput()
		if err != nil {
			fmt.Printf("Error deleting tag %s: %v\n%s\n", tag, err, deleteOutput)
		} else {
			fmt.Printf("Deleted tag %s in %s\n", tag, repoPath)
		}
	}

	fmt.Printf("Completed tag deletion for %s\n", repoPath)
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

	fmt.Printf("Successfully fetched tags in %s\n", repoPath)
	return nil
}

// ProcessTagsSequential deletes local tags and fetches remote tags for all repositories sequentially
func ProcessTagsSequential(repositories []config.Repository) {
	for _, repo := range repositories {
		fmt.Printf("Processing tags for %s\n", repo.Path)
		
		err := DeleteTags(repo.Path)
		if err != nil {
			fmt.Printf("Error deleting tags in %s: %v\n", repo.Path, err)
			continue
		}
		
		err = FetchTags(repo.Path)
		if err != nil {
			fmt.Printf("Error fetching tags in %s: %v\n", repo.Path, err)
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
			
			fmt.Printf("Processing tags for %s\n", r.Path)
			
			err := DeleteTags(r.Path)
			if err != nil {
				fmt.Printf("Error deleting tags in %s: %v\n", r.Path, err)
				return
			}
			
			err = FetchTags(r.Path)
			if err != nil {
				fmt.Printf("Error fetching tags in %s: %v\n", r.Path, err)
			}
		}(repo)
	}

	wg.Wait()
}