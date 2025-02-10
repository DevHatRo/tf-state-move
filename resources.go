package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

func getResourceChoices(resources []Resource) []SelectionItem {
	moduleMap := make(map[string][]string)
	var choices []SelectionItem

	// First, collect modules and their resources
	for _, r := range resources {
		// Build base resource string
		var resourceStr string

		// Handle data sources
		if r.Mode == "data" {
			resourceStr = "data." + r.Type + "." + r.Name
		} else {
			resourceStr = r.Type + "." + r.Name
		}

		// Handle instances array first
		if len(r.Instances) > 0 {
			for _, inst := range r.Instances {
				instanceStr := resourceStr

				// Check for index_key first
				if inst.IndexKey != nil {
					switch idx := inst.IndexKey.(type) {
					case float64:
						instanceStr = resourceStr + "[" + strconv.Itoa(int(idx)) + "]"
					case string:
						instanceStr = resourceStr + "[\"" + idx + "\"]"
					}
				} else if inst.Index != nil {
					switch idx := inst.Index.(type) {
					case float64:
						instanceStr = resourceStr + "[" + strconv.Itoa(int(idx)) + "]"
					case string:
						instanceStr = resourceStr + "[\"" + idx + "\"]"
					case map[string]interface{}:
						if v, ok := idx["value"].(string); ok {
							instanceStr = resourceStr + "[\"" + v + "\"]"
						}
					}
				}

				if r.Module != "" {
					modulePath := formatModulePath(r.Module)
					moduleMap[modulePath] = append(moduleMap[modulePath], instanceStr)
				} else {
					choices = append(choices, SelectionItem{
						Display:  instanceStr,
						Value:    instanceStr,
						IsModule: false,
					})
				}
			}
		} else {
			if r.Module != "" {
				modulePath := formatModulePath(r.Module)
				moduleMap[modulePath] = append(moduleMap[modulePath], resourceStr)
			} else {
				choices = append(choices, SelectionItem{
					Display:  resourceStr,
					Value:    resourceStr,
					IsModule: false,
				})
			}
		}
	}

	// Then create module entries
	var modules []string
	for module := range moduleMap {
		modules = append(modules, module)
	}
	sort.Strings(modules)

	// Create a map to store module hierarchies
	moduleHierarchy := make(map[string][]SelectionItem)
	processedModules := make(map[string]bool)    // Track processed modules
	moduleResources := make(map[string][]string) // Track resources for each module

	// First pass: collect all resources, including nested module resources
	for _, module := range modules {
		resources := moduleMap[module]
		sort.Strings(resources)

		if strings.Contains(module, ".module.") {
			// This is a nested module, add its resources to the parent module
			parts := strings.SplitN(module, ".module.", 2)
			parentModule := parts[0]
			nestedPart := parts[1]

			// Add resources with full nested module path
			for _, resource := range resources {
				fullResource := "module." + nestedPart + "." + resource
				if _, exists := moduleResources[parentModule]; !exists {
					moduleResources[parentModule] = []string{}
				}
				moduleResources[parentModule] = append(moduleResources[parentModule], fullResource)
			}
		} else {
			// This is a root module
			moduleResources[module] = resources
		}
	}

	// Add module entries
	for _, module := range modules {
		if processedModules[module] {
			continue
		}

		// Only process root-level modules
		if strings.Contains(module, ".module.") {
			continue
		}

		moduleName := strings.TrimPrefix(module, "module.")
		baseModule := moduleName
		if idx := strings.Index(baseModule, "."); idx != -1 {
			baseModule = baseModule[:idx]
		}

		// Create the module entry with resource count
		moduleEntry := SelectionItem{
			Display:    fmt.Sprintf("%s (%d)", module, len(moduleResources[module])),
			Value:      module,
			IsModule:   true,
			ModuleName: baseModule,
			Level:      0,
			IsExpanded: false,
			Resources:  moduleResources[module],
		}

		moduleHierarchy["module"] = append(moduleHierarchy["module"], moduleEntry)
		processedModules[module] = true
	}

	// Add module entries in hierarchical order
	var addModules func(parentPath string, level int, parentExpanded bool)
	addModules = func(parentPath string, level int, parentExpanded bool) {
		if modules, ok := moduleHierarchy[parentPath]; ok {
			// Sort modules at the current level
			sort.Slice(modules, func(i, j int) bool {
				return modules[i].ModuleName < modules[j].ModuleName
			})

			for _, module := range modules {
				if level == 0 || parentExpanded {
					// Add module with IsGrouping flag set to true
					moduleChoice := module
					moduleChoice.IsGrouping = true // Add this field to SelectionItem struct
					choices = append(choices, moduleChoice)

					// Add resources if module is expanded
					if module.IsExpanded && len(module.Resources) > 0 {
						// Sort resources for consistent display
						resources := module.Resources
						sort.Strings(resources)

						for _, resource := range resources {
							choices = append(choices, SelectionItem{
								Display:  resource,
								Value:    module.Value + "." + resource,
								Level:    level + 1,
								IsModule: false,
							})
						}
					}
				}
			}
		}
	}

	// Start adding modules from the root
	addModules("module", 0, true)

	return choices
}
