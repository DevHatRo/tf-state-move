package main

import (
	"sort"
	"strconv"
	"strings"
)

// buildResourceTree turns the flat list of state resources into a tree of
// module groups and resource leaves. Modules nest to any depth, so a resource
// at "module.app.module.db" becomes a leaf under a "module.db" node that is
// itself a child of "module.app".
func buildResourceTree(resources []Resource) []*treeNode {
	root := &treeNode{isModule: true, expanded: true}

	for _, r := range resources {
		parent := root
		cumPath := ""
		for _, seg := range moduleSegments(r.Module) {
			if cumPath == "" {
				cumPath = "module." + seg
			} else {
				cumPath += ".module." + seg
			}
			parent = childModule(parent, "module."+seg, cumPath)
		}

		for _, leaf := range resourceLeafStrings(r) {
			value := leaf
			if cumPath != "" {
				value = cumPath + "." + leaf
			}
			parent.children = append(parent.children, &treeNode{label: leaf, value: value})
		}
	}

	sortTree(root.children)
	setLevels(root.children, 0)
	return root.children
}

// childModule returns the module child of parent with the given path,
// creating it if it does not exist yet.
func childModule(parent *treeNode, label, path string) *treeNode {
	for _, c := range parent.children {
		if c.isModule && c.value == path {
			return c
		}
	}
	child := &treeNode{label: label, value: path, isModule: true}
	parent.children = append(parent.children, child)
	return child
}

// moduleSegments splits a state module path into its successive module names,
// e.g. "module.app.module.db" -> ["app", "db"]. A root resource yields nil.
func moduleSegments(module string) []string {
	module = formatModulePath(module)
	if module == "" {
		return nil
	}
	return strings.Split(strings.TrimPrefix(module, "module."), ".module.")
}

// resourceLeafStrings returns the resource-address suffix for each instance of
// a resource: "aws_instance.web", "aws_instance.web[0]" for a counted
// resource, or "data.aws_ami.ubuntu" for a data source.
func resourceLeafStrings(r Resource) []string {
	base := r.Type + "." + r.Name
	if r.Mode == "data" {
		base = "data." + base
	}
	if len(r.Instances) == 0 {
		return []string{base}
	}

	leaves := make([]string, 0, len(r.Instances))
	for _, inst := range r.Instances {
		// A count index is a number; a for_each key is a string.
		switch idx := inst.IndexKey.(type) {
		case float64:
			leaves = append(leaves, base+"["+strconv.Itoa(int(idx))+"]")
		case string:
			leaves = append(leaves, base+"[\""+idx+"\"]")
		default:
			leaves = append(leaves, base)
		}
	}
	return leaves
}

// sortTree orders every level of the tree: module groups first, then resource
// leaves, each group sorted by label.
func sortTree(nodes []*treeNode) {
	sort.SliceStable(nodes, func(i, j int) bool {
		if nodes[i].isModule != nodes[j].isModule {
			return nodes[i].isModule
		}
		return nodes[i].label < nodes[j].label
	})
	for _, n := range nodes {
		sortTree(n.children)
	}
}

// setLevels records each node's depth so its row can be indented correctly.
func setLevels(nodes []*treeNode, level int) {
	for _, n := range nodes {
		n.level = level
		setLevels(n.children, level+1)
	}
}

// flattenVisible returns the rows currently visible in the tree, top to
// bottom. A node is visible when every ancestor module is expanded.
func flattenVisible(nodes []*treeNode) []*treeNode {
	var rows []*treeNode
	var walk func([]*treeNode)
	walk = func(ns []*treeNode) {
		for _, n := range ns {
			rows = append(rows, n)
			if n.isModule && n.expanded {
				walk(n.children)
			}
		}
	}
	walk(nodes)
	return rows
}

// leafValues returns the resource addresses of every resource leaf at or
// below n (n itself when it is a leaf).
func leafValues(n *treeNode) []string {
	if !n.isModule {
		return []string{n.value}
	}
	var values []string
	for _, c := range n.children {
		values = append(values, leafValues(c)...)
	}
	return values
}

// resourceCount returns the number of resource leaves under a node.
func resourceCount(n *treeNode) int {
	return len(leafValues(n))
}

// nodeSelected reports whether a node should render as selected: a leaf when
// it is in the selection, a module when every one of its leaves is.
func nodeSelected(n *treeNode, selected map[string]bool) bool {
	leaves := leafValues(n)
	if len(leaves) == 0 {
		return false
	}
	for _, v := range leaves {
		if !selected[v] {
			return false
		}
	}
	return true
}

// selectedResourcePaths returns the chosen resource addresses, sorted. The
// selection map holds only resource leaves — module rows are never stored.
func selectedResourcePaths(selected map[string]bool) []string {
	paths := make([]string, 0, len(selected))
	for value, isSelected := range selected {
		if isSelected {
			paths = append(paths, value)
		}
	}
	sort.Strings(paths)
	return paths
}
