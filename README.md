# GitSwitch - Multi-Repository Git Branch Management Tool

### This is created by AI

GitSwitch is a command-line tool that helps you efficiently manage branches across multiple Git repositories. It allows you to switch branches in multiple repositories simultaneously, with fallback logic if preferred branches don't exist, and also provides functionality to refresh Git tags across repositories.

## Features

- **Branch Switching with Fallback Logic**: Automatically attempts to switch to branches in a priority order, with fallbacks if preferred branches don't exist
- **Tag Management**: Easily delete local tags and fetch the latest tags from remotes across all repositories
- **Hierarchical Configuration**: Manages repositories using a parent-subfolder structure in a YAML configuration file
- **Parallel Processing**: Option to run operations in parallel for faster execution in large repository structures
- **Repository Status Overview**: View the current state of all repositories

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
go build -o gitswitch.exe
```

## Configuration

GitSwitch uses a YAML configuration file (`gitswitch.yml` by default) to define the branches and repositories to manage.

### Example Configuration

```yaml
branches:
  - "feature/new-feature"  # First priority branch
  - "develop"              # Second priority (fallback)
  - "main"                 # Third priority (last resort)

repositories:
  - "H:/code_base/project1/backend":
      - "api-service" 
      - "db-service"
      - "auth-service"

  - "H:/code_base/project1/frontend":
      - "web-client"
      - "mobile-client"
```

In this configuration:
- The tool will try to switch each repository to `feature/new-feature` first
- If that branch doesn't exist, it will try `develop`
- If neither exists, it will try `main`
- Repositories are organized hierarchically with parent paths and subfolders

## Usage

### List Repository Status

View the current branch status of all repositories:

```
gitswitch list
```

### Switch Branches

Switch branches in all repositories according to the priority defined in the configuration:

```
gitswitch switch
```

Use parallel processing for faster execution:

```
gitswitch switch --parallel
```

### Refresh Tags

Delete all local tags and fetch the latest tags from remotes for all repositories:

```
gitswitch tags
```

Use parallel processing for faster tag refresh:

```
gitswitch tags --parallel
```

### Using a Custom Configuration File

You can specify a different configuration file:

```
gitswitch list --config other-config.yml
gitswitch switch --config other-config.yml
gitswitch tags --config other-config.yml
```

## Project Structure

- `main.go`: Entry point of the application
- `cmd/cmd.go`: Command-line interface implementation
- `config/config.go`: Configuration parsing and management
- `git/git.go`: Git operations implementation

## License

MIT

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.