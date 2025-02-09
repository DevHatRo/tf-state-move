package main

import (
	"fmt"
	"sort"
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
						instanceStr = fmt.Sprintf("%s[%d]", resourceStr, int(idx))
					case string:
						instanceStr = fmt.Sprintf("%s[\"%s\"]", resourceStr, idx)
					}
				} else if inst.Index != nil {
					switch idx := inst.Index.(type) {
					case float64:
						instanceStr = fmt.Sprintf("%s[%d]", resourceStr, int(idx))
					case string:
						instanceStr = fmt.Sprintf("%s[\"%s\"]", resourceStr, idx)
					case map[string]interface{}:
						if v, ok := idx["value"].(string); ok {
							instanceStr = fmt.Sprintf("%s[\"%s\"]", resourceStr, v)
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

	// Add module entries with their resources
	for _, module := range modules {
		resources := moduleMap[module]
		sort.Strings(resources)

		// Remove "module." prefix for ModuleName
		moduleName := strings.TrimPrefix(module, "module.")

		choices = append(choices, SelectionItem{
			Display:    module,
			Value:      module,
			IsModule:   true,
			ModuleName: moduleName,
			Resources:  resources,
			Level:      0,
		})
	}

	// Sort non-module entries
	sort.Slice(choices, func(i, j int) bool {
		return choices[i].Display < choices[j].Display
	})

	return choices
}
