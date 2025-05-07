package log

import (
	"fmt"
	"strings"
)

// Error codes for all application errors
const (
	// Configuration errors (1xx)
	ErrConfigReadFailed  = "E101" // Error reading configuration file
	ErrConfigParseFailed = "E102" // Error parsing configuration file
	ErrNoConfigBranches  = "E103" // No branches specified in configuration
	ErrNoConfigRepos     = "E104" // No repositories found in configuration

	// Git operation errors (2xx)
	ErrGitBranchNotFound     = "E201" // Branch not found locally or remotely
	ErrGitCheckoutFailed     = "E202" // Failed to checkout branch
	ErrGitStashFailed        = "E203" // Failed to stash changes
	ErrGitApplyStashFailed   = "E204" // Failed to apply stashed changes
	ErrGitFetchFailed        = "E205" // Failed to fetch from remote
	ErrGitPullFailed         = "E206" // Failed to pull from remote
	ErrGitTagOperationFailed = "E207" // Failed to perform tag operation

	// Repository errors (3xx)
	ErrRepoNotFound    = "E301" // Repository not found
	ErrRepoInvalidPath = "E302" // Invalid repository path
	ErrRepoNotGit      = "E303" // Not a git repository

	// History operation errors (4xx)
	ErrHistoryReadFailed   = "E401" // Failed to read history file
	ErrHistoryWriteFailed  = "E402" // Failed to write history file
	ErrHistoryStateFailed  = "E403" // Failed to save state to history
	ErrHistoryIndexInvalid = "E404" // Invalid history index

	// General errors (9xx)
	ErrInvalidArgument = "E901" // Invalid argument passed
	ErrOperationFailed = "E999" // Generic operation failed
)

// FormatError formats an error with a consistent structure including the error code
func FormatError(code string, description string, err error) string {
	if err != nil {
		return fmt.Sprintf("[%s] %s: %v", code, description, err)
	}
	return fmt.Sprintf("[%s] %s", code, description)
}

// GetErrorCode extracts the error code from a formatted error message
func GetErrorCode(errorMsg string) string {
	if strings.HasPrefix(errorMsg, "[E") && len(errorMsg) >= 6 {
		return errorMsg[1:5]
	}
	return ""
}
