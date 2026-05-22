package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

// Version will be set during build
var Version = "dev"

func main() {
	var (
		showHelp    bool
		showVersion bool
	)

	// Define flags
	flag.BoolVar(&showHelp, "h", false, "Show help message")
	flag.BoolVar(&showHelp, "help", false, "Show help message")
	flag.BoolVar(&showVersion, "v", false, "Show version information")
	flag.BoolVar(&showVersion, "version", false, "Show version information")

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

	// Build the module/resource tree
	tree := buildResourceTree(state.Resources)

	// Create and run UI
	ui := newUI(tree, inStatePath, outStatePath)
	if err := ui.run(); err != nil {
		fmt.Printf("Error running UI: %v\n", err)
		os.Exit(1)
	}

	// Get selected resources
	selectedResources := selectedResourcePaths(ui.selectedItems)

	if len(selectedResources) == 0 {
		fmt.Println("No resources selected. Exiting.")
		return
	}

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
	if err := moveResources(selectedResources, inStatePath, outStatePath, false); err != nil {
		fmt.Printf("Error moving resources: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Resources moved successfully!")
}
