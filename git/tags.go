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

// DeleteLocalTags deletes all local tags in the repository
func DeleteLocalTags(repoPath string) error {
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %v", err)
	}

	// Check if repository exists
	if _, err := os.Stat(filepath.Join(absPath, ".git")); os.IsNotExist(err) {
		return fmt.Errorf("not a git repository or directory does not exist")
	}

	// Get all local tags
	listCmd := exec.Command("git", "-C", absPath, "tag")
	tagList, err := listCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to list tags: %v", err)
	}

	// If there are no tags, we're done
	tagListStr := strings.TrimSpace(string(tagList))
	if len(tagListStr) == 0 {
		fmt.Printf("No tags to delete in %s\n", repoPath)
		return nil
	}

	// Split tags by newline
	tags := strings.Split(tagListStr, "\n")
	fmt.Printf("Found %d tags to delete in %s\n", len(tags), repoPath)

	// Delete tags one by one
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		
		deleteCmd := exec.Command("git", "-C", absPath, "tag", "-d", tag)
		deleteOutput, err := deleteCmd.CombinedOutput()
		if err != nil {
			fmt.Printf("Warning: failed to delete tag %s: %v\n%s\n", tag, err, deleteOutput)
			// Continue with other tags even if one fails
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
		
		err := DeleteLocalTags(repo.Path)
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
			
			err := DeleteLocalTags(r.Path)
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