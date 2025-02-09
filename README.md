# Terraform State Move

Terraform State Move is a Go-based CLI tool designed to help you organize and manage large Terraform state files. It provides an interactive command-line interface for moving selected resources between Terraform state files.

## Installation

### Go Install (All Platforms)
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

Terraform must be installed and initialized within your working directory.

To use tf-state-move:

```bash
# Extract the state file from your Terraform workspace or remote state storage
terraform state pull > terraform.tfstate

# Run the tool and follow the prompts to select resources to be moved
tf-state-move -i terraform.tfstate -o new-state.tfstate
```

## Usage

```bash
tf-state-move [options]

Options:
  -h, --help                Show help message
  -v, --version            Show version information
  -i, --in-state-path      Input state file path
  -o, --out-state-path     Output state file path
  --debug                  Enable debug output
```

The tool will:
1. Prompt for input/output state file paths if not provided
2. Display all resources from the input state file
3. Allow selection of resources to move using an interactive UI
4. Move selected resources using `terraform state mv` command

## Features

- Interactive UI for selecting resources to move
- Support for module resources
- Handles complex resource addresses including modules, string indices, and numeric indices
- Easy resource selection with arrow keys, space bar, and enter key

## License

MIT License - see LICENSE file for details
