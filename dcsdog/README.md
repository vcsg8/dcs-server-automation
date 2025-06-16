# DCSDOG

DCSDOG is a Windows automation tool for DCS World Server that helps manage server security and automation tasks.

## Features

- Automatically finds DCS World installation path from Windows registry
- Manages MissionScripting.lua file to ensure proper security settings
- Monitors and restarts DCS server process when needed

## Requirements

- Windows operating system
- DCS World installed
- Go 1.24 or later
- Task (task runner)

## Development Setup

1. Clone this repository
2. Navigate to the dcsdog directory
3. Run the setup command:
   ```bash
   task setup
   ```
   This will:
   - Download Go dependencies
   - Install golangci-lint for code quality checks

## Development Tasks

The project uses Task for development tasks:

- `task setup` - Set up development environment
- `task lint` - Run golangci-lint
- `task vet` - Run go vet
- `task build` - Build the executable
- `task clean` - Remove build artifacts
- `task test` - Run tests
- `task` - Run all checks (lint, vet, test)

## Building

To build the executable:

```bash
task build
```

This will create `dcsdog.exe` in the current directory.

## Usage

Run the compiled executable:

```bash
./dcsdog
```

The tool will:
1. Find your DCS World installation
2. Check and update MissionScripting.lua if needed
3. Restart the DCS server if it's running

## Security

The tool creates backups of modified files and includes safety checks to prevent accidental modifications. It also marks its modifications with a comment to prevent duplicate modifications.

## CI/CD

The project uses GitHub Actions for continuous integration:
- Runs on every push to main and pull requests
- Performs linting, vetting, and build checks
- Ensures code quality and buildability

## Releases

Releases are automatically created when a tag is pushed that matches the pattern `dcsdog/v*` (e.g., `dcsdog/v1.0.0`).

To create a new release:
1. Update version in code if needed
2. Create and push a new tag:
   ```bash
   git tag dcsdog/v1.0.0
   git push origin dcsdog/v1.0.0
   ```
3. The release workflow will automatically:
   - Build the executable
   - Create a GitHub release
   - Upload the executable as a release asset

## Testing

Run the test suite:
```bash
task test
```

The test suite includes:
- Unit tests for file modification logic
- Process detection tests
- Registry interaction tests (mocked) 