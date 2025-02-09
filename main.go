package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
)

// Version will be set during build
var Version = "dev"

// Add debug flag
var debug bool

var (
	version string
)

func main() {
	var (
		showHelp    bool
		showVersion bool
		debug       bool
	)

	// Define flags
	flag.BoolVar(&showHelp, "h", false, "Show help message")
	flag.BoolVar(&showHelp, "help", false, "Show help message")
	flag.BoolVar(&showVersion, "v", false, "Show version information")
	flag.BoolVar(&showVersion, "version", false, "Show version information")
	flag.BoolVar(&debug, "debug", false, "Enable debug output")

	var inStatePath string
	var outStatePath string

	flag.StringVar(&inStatePath, "i", "", "Input state file path")
	flag.StringVar(&inStatePath, "in-state-path", "", "Input state file path")
	flag.StringVar(&outStatePath, "o", "", "Output state file path")
	flag.StringVar(&outStatePath, "out-state-path", "", "Output state file path")

	flag.Parse()

	if showHelp {
		printHelp()
		return
	}

	if showVersion {
		fmt.Printf("tf-state-move version %s\n", Version)
		return
	}

	var err error
	if inStatePath == "" {
		inStatePath, err = getFilePath("", "Enter input state file path", "terraform.tfstate")
		if err != nil {
			fmt.Printf("Error getting input state file path: %v\n", err)
			os.Exit(1)
		}
	}

	if outStatePath == "" {
		outStatePath, err = getFilePath("", "Enter output state file path", "terraform.tfstate.backup")
		if err != nil {
			fmt.Printf("Error getting output state file path: %v\n", err)
			os.Exit(1)
		}
	}

	// Read and parse state file
	state, err := readStateFile(inStatePath)
	if err != nil {
		fmt.Printf("Error reading state file: %v\n", err)
		os.Exit(1)
	}

	// Get choices with tree structure
	choices := getResourceChoices(state.Resources)

	// Create and run UI
	ui := newUI(choices, inStatePath, outStatePath)
	if err := ui.run(); err != nil {
		fmt.Printf("Error running UI: %v\n", err)
		os.Exit(1)
	}

	// Get selected resources
	var selectedResources []string
	for value, selected := range ui.selectedItems {
		if selected {
			// Skip the module itself but keep its resources
			// Skip entries like "module.module.acm["testnet"]" but keep "module.module.acm["testnet"].aws_acm_certificate.this"
			if strings.HasPrefix(value, "module.") && !strings.Contains(value, ".aws_") {
				continue // Skip module entries that don't contain a resource
			}
			selectedResources = append(selectedResources, value)
		}
	}

	if len(selectedResources) == 0 {
		fmt.Println("No resources selected. Exiting.")
		return
	}

	// Sort resources for consistent display
	sort.Strings(selectedResources)

	// Show selected resources and confirm
	fmt.Println("\nSelected Resources:")
	for _, resource := range selectedResources {
		fmt.Printf("  • %s\n", resource)
	}

	fmt.Print("\nProceed with moving these resources? [y/N] ")
	var confirm string
	fmt.Scanln(&confirm)
	if strings.ToLower(confirm) != "y" {
		fmt.Println("Operation cancelled.")
		return
	}

	// Move resources
	fmt.Println("\nMoving selected resources...")
	if err := moveResources(selectedResources, inStatePath, outStatePath, debug); err != nil {
		fmt.Printf("Error moving resources: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Resources moved successfully!")
}
