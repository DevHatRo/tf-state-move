package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// formatModulePath ensures a module path carries the "module." prefix.
func formatModulePath(module string) string {
	if module != "" && !strings.HasPrefix(module, "module.") {
		return "module." + module
	}
	return module
}

func getFilePath(flagValue, promptText, defaultPath string) (string, error) {
	if flagValue != "" {
		return flagValue, nil
	}
	fmt.Printf("%s [%s]: ", promptText, defaultPath)
	var input string
	_, err := fmt.Scanln(&input)
	if err != nil {
		if err == io.EOF {
			return defaultPath, nil
		}
		return "", fmt.Errorf("error reading input: %w", err)
	}
	if input == "" {
		return defaultPath, nil
	}
	return input, nil
}

func printHelp() {
	fmt.Println(`Usage: tf-state-move [options]

Options:
  -h, --help               Show help message
  -v, --version            Show version information
  -i, --in-state-path      Input state file path
  -o, --out-state-path     Output state file path`)
}

// readStateFile reads and parses a Terraform state file, rejecting any state
// format version this tool does not understand.
func readStateFile(filePath string) (*TerraformState, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state TerraformState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}

	if state.Version != stateVersion {
		return nil, fmt.Errorf("unsupported state format version %d (expected %d)", state.Version, stateVersion)
	}

	return &state, nil
}

// writeStateFile marshals a Terraform state and writes it to path.
func writeStateFile(path string, state *TerraformState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}
	data = append(data, '\n')
	if err = os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}
	return nil
}

// resourceAddress returns the canonical Terraform address of a resource,
// e.g. "module.network.aws_vpc.main" or "data.aws_ami.ubuntu".
func resourceAddress(r Resource) string {
	var b strings.Builder
	if r.Module != "" {
		b.WriteString(formatModulePath(r.Module))
		b.WriteString(".")
	}
	if r.Mode == "data" {
		b.WriteString("data.")
	}
	b.WriteString(r.Type)
	b.WriteString(".")
	b.WriteString(r.Name)
	return b.String()
}

// resourceLevelAddress strips a trailing instance index ("[0]" or `["key"]`)
// from a selection path so it can be matched against a resourceAddress. The
// whole resource (every instance) moves together, so the index is dropped.
func resourceLevelAddress(path string) string {
	if strings.HasSuffix(path, "]") {
		if i := strings.LastIndex(path, "["); i > 0 {
			return path[:i]
		}
	}
	return path
}

// moveResources splits resources between two state files: the selected
// resources are written to outStatePath and removed from inStatePath, while
// every other field of the state is preserved. The serial of each written
// file is bumped so `terraform state push` accepts it. When dryRun is true,
// nothing is written to disk.
func moveResources(selectedResources []string, inStatePath, outStatePath string, dryRun bool) error {
	inState, err := readStateFile(inStatePath)
	if err != nil {
		return fmt.Errorf("failed to read input state: %w", err)
	}

	selected := make(map[string]bool, len(selectedResources))
	for _, s := range selectedResources {
		selected[resourceLevelAddress(s)] = true
	}

	// outState keeps every top-level field of the input (version, lineage,
	// outputs, ...) and differs only in which resources it carries.
	outState := *inState

	toMove := make([]Resource, 0, len(selectedResources))
	toKeep := make([]Resource, 0, len(inState.Resources))
	for _, r := range inState.Resources {
		if selected[resourceAddress(r)] {
			toMove = append(toMove, r)
		} else {
			toKeep = append(toKeep, r)
		}
	}

	inState.Resources = toKeep
	outState.Resources = toMove
	inState.Serial++
	outState.Serial++

	if !dryRun {
		if writeErr := writeStateFile(outStatePath, &outState); writeErr != nil {
			return fmt.Errorf("failed to write output state: %w", writeErr)
		}
		if writeErr := writeStateFile(inStatePath, inState); writeErr != nil {
			return fmt.Errorf("failed to write input state: %w", writeErr)
		}
	}

	fmt.Printf("Moved %d resource(s) to %s\n", len(toMove), outStatePath)
	fmt.Printf("Kept %d resource(s) in %s\n", len(toKeep), inStatePath)
	return nil
}
