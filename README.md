# GitSwitch - Multi-Repository Git Branch Management Tool

A command-line tool that helps you efficiently manage branches across multiple Git repositories. It allows you to switch branches in multiple repositories simultaneously, with fallback logic if preferred branches don't exist, and also provides functionality for tag management, branch history tracking, and parallel operations.

## Features

- **Branch Switching with Fallback Logic**: Automatically attempts to switch to branches in a priority order, with fallbacks if preferred branches don't exist
- **Tag Management**: Easily delete local tags and fetch the latest tags from remotes across all repositories
- **Hierarchical Configuration**: Manages repositories using a parent-subfolder structure in a YAML configuration file
- **Parallel Processing**: Option to run operations in parallel for faster execution in large repository structures
- **Repository Status Overview**: View the current state of all repositories
- **Stash Management**: Stash your changes before switching branches with automatic tracking
- **Branch History**: Save and restore previous branch states across all repositories
- **Pull Operations**: Pull the latest changes from remote repositories

## Installation

### Prerequisites

- Go 1.16 or higher
- Git installed and accessible from the command line

### Building from source

1. Clone the repository

```
git clone https://github.com/yourusername/gitswitch.git
cd gitswitch
```

2. Build the application

```
go build -o git_cli_tool.exe
```

## Configuration

GitSwitch uses a YAML configuration file (`git_cli_tool.yml` by default) to define the branches and repositories to manage.

### Example Configuration

```yaml
branches:
  - "feature/new-feature" # First priority branch
  - "develop" # Second priority (fallback)
  - "main" # Third priority (last resort)

repositories:
  - "H:/code_base/project1/backend":
      - "api-service"
      - "db-service"
      - "auth-service"

  - "H:/code_base/project1/frontend":
      - "web-client"
      - "mobile-client"

# Optional: Define branch dependencies for the sync command
branch_dependencies:
  "feature/extension": "feature/base"
  "feature/part2": "feature/part1"

# Optional: Fallback branch when parent is not found (default: main)
fallback_branch: "main"
```

In this configuration:

- The tool will try to switch each repository to `feature/new-feature` first
- If that branch doesn't exist, it will try `develop`
- If neither exists, it will try `main`
- Repositories are organized hierarchically with parent paths and subfolders

## Usage

### Quick Status Check

Get a quick overview of repositories that have uncommitted changes or are out of sync:

```
git_cli_tool status
```

Show all repositories (not just those with issues):

```
git_cli_tool status --all
```

### List Repository Status

View the current branch status of all repositories:

```
git_cli_tool list
```

### Pull Latest Changes

Pull the latest changes from remote repositories for all configured repositories:

```
git_cli_tool pull
```

Use parallel processing for faster pulling:

```
git_cli_tool pull --parallel
```

### Switch Branches

Switch branches in all repositories according to the priority defined in the configuration:

```
git_cli_tool switch
```

Use parallel processing for faster execution:

```
git_cli_tool switch --parallel
```

Stash your changes before switching:

```
git_cli_tool switch --autostash "my-stash-name"
```

or using the shorter form:

```
git_cli_tool switch -a "my-stash-name"
```

Control whether to store branch state history:

```
git_cli_tool switch --store-history=false
```

Add a description to the history entry:

```
git_cli_tool switch --description "Switching to feature branch for sprint 10"
```

Preview what branches would be switched to without making changes (includes remote branch check):

```
git_cli_tool switch --dry-run
```

### Sync Branch with Parent

Sync a branch with its parent branch across all repositories. This is useful when you have dependent branches that need to stay up-to-date with their parent:

```
git_cli_tool sync feature/extension
```

This will:

1. Switch to `feature/extension` in each repository
2. Look up the parent branch from `branch_dependencies` in your config
3. Merge the parent branch into `feature/extension`
4. If the parent branch doesn't exist, fall back to `main` (or your configured `fallback_branch`)

The sync command handles merge conflicts gracefully - it will report which repositories had conflicts and abort the merge, leaving those repos in a clean state for manual resolution.

### Refresh Tags

Delete all local tags and fetch the latest tags from remotes for all repositories:

```
git_cli_tool tags
```

Use parallel processing for faster tag refresh:

```
git_cli_tool tags --parallel
```

### View Branch History

View the history of previous branch states:

```
git_cli_tool history
```

### Revert to Previous State

Revert to the most recent saved branch state:

```
git_cli_tool revert
```

Revert to a specific state by index:

```
git_cli_tool revert <index>
```

Apply stashes when reverting (on by default):

```
git_cli_tool revert --apply-stashes
```

Disable stash application:

```
git_cli_tool revert --apply-stashes=false
```

### Using a Custom Configuration File

You can specify a different configuration file with any command:

```
git_cli_tool list --config other-config.yml
git_cli_tool switch --config other-config.yml
git_cli_tool tags --config other-config.yml
```

## Project Structure

The project has a modular structure for better organization:

### Main packages

- `main.go`: Entry point of the application
- `cmd/`: Command-line interface implementation using Cobra
  - `cmd.go`: Package documentation
  - `root.go`: Core command structure and global flags
  - `switch.go`: Branch switching functionality
  - `list.go`: Repository listing operations
  - `tags.go`: Tag management commands
  - `history.go`: Branch history tracking
  - `revert.go`: State restoration functionality
  - `pull.go`: Repository pull operations
  - `status.go`: Quick repository status overview
  - `sync.go`: Branch dependency synchronization
- `config/`: Configuration parsing and management
  - `config.go`: Core configuration loading
  - `history.go`: History tracking and state management
- `git/`: Git operations implementation
  - `git.go`: Package documentation
  - `branch.go`: Branch-related operations
  - `stash.go`: Stash management functions
  - `tags.go`: Tag operations
  - `util.go`: Common utility functions

## License

MIT

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
