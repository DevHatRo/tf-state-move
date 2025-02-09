package main

import (
	"fmt"
	"strings"
)

func formatModulePath(module string) string {
	// If the module path doesn't start with "module.", add it
	if module != "" && !strings.HasPrefix(module, "module.") {
		return fmt.Sprintf("module.%s", module)
	}
	return module
}

func getFilePath(flagValue, promptText, defaultPath string) (string, error) {
	if flagValue != "" {
		return flagValue, nil
	}
	fmt.Printf("%s [%s]: ", promptText, defaultPath)
	var input string
	fmt.Scanln(&input)
	if input == "" {
		return defaultPath, nil
	}
	return input, nil
}

func printHelp() {
	fmt.Println(`Usage: tf-state-move [options]

Options:
  -h, --help                Show help message
  -v, --version            Show version information
  -i, --in-state-path      Input state file path
  -o, --out-state-path     Output state file path
  --debug                  Enable debug output`)
} 
