package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func readStateFile(path string) (*TerraformState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading state file: %w", err)
	}

	var state TerraformState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("error parsing state file: %w", err)
	}

	return &state, nil
}

func moveResources(selectedResources []string, inStatePath, outStatePath string, debug bool) error {
	inDir := filepath.Dir(inStatePath)

	// Convert paths to absolute to avoid any path-related issues
	inStateAbs, err := filepath.Abs(inStatePath)
	if err != nil {
		return fmt.Errorf("error getting absolute path for input state: %w", err)
	}
	outStateAbs, err := filepath.Abs(outStatePath)
	if err != nil {
		return fmt.Errorf("error getting absolute path for output state: %w", err)
	}

	// Move each resource
	for _, resourceName := range selectedResources {
		args := []string{
			"state", "mv",
			fmt.Sprintf("-state=%s", inStateAbs),
			fmt.Sprintf("-state-out=%s", outStateAbs),
			resourceName, resourceName,
		}
		
		if debug {
			fmt.Printf("Debug: Running command: terraform %s\n", strings.Join(args, " "))
			fmt.Printf("Debug: Working directory: %s\n", inDir)
		}

		cmd := exec.Command("terraform", args...)
		cmd.Dir = inDir
		
		output, err := cmd.CombinedOutput()
		if err != nil {
			// If the resource doesn't exist, provide a clear error message
			if strings.Contains(string(output), "does not match anything in the current state") {
				return fmt.Errorf("resource '%s' not found in state file '%s'", 
					resourceName, 
					inStateAbs)
			}
			return fmt.Errorf("error moving resource %s:\n%s", 
				resourceName, 
				string(output))
		}
		
		fmt.Printf("Successfully moved: %s\n", resourceName)
	}

	return nil
} 
