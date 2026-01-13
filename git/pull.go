package git

import (
	"fmt"
	"os/exec"
	"sync"

	"git_cli_tool/config"
	"git_cli_tool/log"
)

// PullRepositories pulls the latest changes from remote in all repositories in parallel
func PullRepositories(repositories []config.Repository) {
	var wg sync.WaitGroup
	wg.Add(len(repositories))

	// Use a mutex to prevent output from different goroutines from interleaving
	var outputMutex sync.Mutex

	for _, repo := range repositories {
		go func(r config.Repository) {
			defer wg.Done()

			// Sync tags before pulling
			if err := SyncTags(r.Path); err != nil {
				outputMutex.Lock()
				log.PrintErrorNoExit(log.ErrGitTagOperationFailed, fmt.Sprintf("Error syncing tags in %s", r.Path), err)
				outputMutex.Unlock()
			}

			cmd := exec.Command("git", "-C", r.Path, "pull")
			output, err := cmd.CombinedOutput()

			outputMutex.Lock()
			defer outputMutex.Unlock()

			if err != nil {
				log.PrintErrorNoExit(log.ErrGitPullFailed, fmt.Sprintf("Error pulling in %s", r.Path), err)
				log.PrintInfo(string(output))
			} else {
				log.PrintSuccess(fmt.Sprintf("Successfully pulled in %s", r.Path))
				log.PrintInfo(string(output))
			}
		}(repo)
	}

	wg.Wait()
}
