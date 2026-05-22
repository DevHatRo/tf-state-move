# Terraform State Move

Terraform State Move is a Go-based CLI tool designed to help you organize and manage large Terraform state files. It provides an interactive command-line interface for moving selected resources between Terraform state files.

## Building from Source

### Prerequisites

- Go 1.26 or later
- Make (optional, for using Makefile targets)

### Build Scripts

The project includes several build scripts for cross-compilation:

#### Build for All Supported Platforms
```bash
./build.sh
```
This builds for:
- Linux (amd64, 386, arm64, arm)
- macOS (amd64, arm64)
- Windows (amd64, 386, arm64)

#### Build for CI Platforms Only
```bash
./build-ci.sh
```
This builds for the same platforms as the CI workflow:
- Linux (amd64)
- Windows (amd64)
- macOS (amd64)

#### Using Makefile
```bash
# Show all available targets
make help

# Build for all platforms
make build

# Build for CI platforms only
make build-ci

# Build for a single platform
make single GOOS=linux GOARCH=amd64

# Clean build artifacts
make clean

# Create release zip
make release
```

All build artifacts are placed in the `build/` directory.

### Installation

#### Go Install (All Platforms)
```bash
go install github.com/DevHatRo/tf-state-move@latest
```

### Homebrew (macOS and Linux)
```bash
# Tap the repository
brew tap DevHatRo/homebrew-tap

# Install the tool
brew install tf-state-move
```

### Manual Installation

Download the appropriate binary for your platform from the [releases page](https://github.com/DevHatRo/tf-state-move/releases).

#### Linux and macOS
```bash
# Download (replace VERSION and ARCH with appropriate values)
curl -L https://github.com/DevHatRo/tf-state-move/releases/download/vVERSION/tf-state-move_VERSION_ARCH.tar.gz | tar xz

# Move to a directory in your PATH
sudo mv tf-state-move /usr/local/bin/
```

#### Windows
Download the ZIP file from the releases page and extract it to a directory in your PATH.

## Getting Started

`tf-state-move` works directly on Terraform state files in JSON format. You
need Terraform installed only to pull the state out of (and later push it back
into) your workspace or remote backend — the tool itself does not invoke
Terraform.

```bash
# Export the state file from your Terraform workspace or remote backend
terraform state pull > terraform.tfstate

# Run the tool and follow the prompts to select resources to move
tf-state-move -i terraform.tfstate -o new-state.tfstate
```

The selected resources are **removed from the input file** and written to the
output file. Both files are written in place, so back them up first. Review
the result, then push each file to its workspace with `terraform state push`.

## Usage

```bash
tf-state-move [options]

Options:
  -h, --help               Show help message
  -v, --version            Show version information
  -i, --in-state-path      Input state file path
  -o, --out-state-path     Output state file path
```

The tool will:
1. Prompt for input/output state file paths if not provided
2. Display all resources from the input state file
3. Allow selection of resources to move using an interactive UI
4. Write the selected resources to the output state file and remove them from the input state file

## Features

- Interactive UI for selecting resources to move
- Multi-level tree view: nested modules are shown as expandable groups, to any depth
- Selecting a module selects every resource beneath it
- Handles complex resource addresses including modules, string indices, and numeric indices
- Easy resource selection with arrow keys, space bar, and enter key

## License

MIT License - see LICENSE file for details
