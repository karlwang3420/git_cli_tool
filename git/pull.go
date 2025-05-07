package git

import (
	"fmt"
	"os/exec"
	"sync"
	
	"git_cli_tool/config"
)

// PullRepositoriesSequential pulls the latest changes from remote in each repository sequentially
func PullRepositoriesSequential(repositories []config.Repository) {
	for _, repo := range repositories {
		fmt.Printf("Pulling in %s...\n", repo.Path)
		cmd := exec.Command("git", "-C", repo.Path, "pull")
		output, err := cmd.CombinedOutput()
		
		if err != nil {
			fmt.Printf("Error pulling in %s: %v\n%s\n", repo.Path, err, string(output))
		} else {
			fmt.Printf("Successfully pulled in %s\n%s\n", repo.Path, string(output))
		}
	}
}

// PullRepositoriesParallel pulls the latest changes from remote in all repositories in parallel
func PullRepositoriesParallel(repositories []config.Repository) {
	var wg sync.WaitGroup
	wg.Add(len(repositories))
	
	// Use a mutex to prevent output from different goroutines from interleaving
	var outputMutex sync.Mutex
	
	for _, repo := range repositories {
		go func(r config.Repository) {
			defer wg.Done()
			
			cmd := exec.Command("git", "-C", r.Path, "pull")
			output, err := cmd.CombinedOutput()
			
			outputMutex.Lock()
			defer outputMutex.Unlock()
			
			if err != nil {
				fmt.Printf("Error pulling in %s: %v\n%s\n", r.Path, err, string(output))
			} else {
				fmt.Printf("Successfully pulled in %s\n%s\n", r.Path, string(output))
			}
		}(repo)
	}
	
	wg.Wait()
}