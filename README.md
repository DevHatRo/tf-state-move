# Terraform State Move

Terraform State Move is a Go-based CLI tool designed to help you organize and manage large Terraform state files. It provides an interactive command-line interface for moving selected resources between Terraform state files.

## Features

- Interactive UI for selecting resources to move
- Support for module resources
- Handles complex resource addresses including modules, string indices, and numeric indices
- Easy resource selection with arrow keys, space bar, and enter key

## Installation

```bash
go install github.com/DevHatRo/tf-state-move@latest
```

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

## License

MIT License - see LICENSE file for details
