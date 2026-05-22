package main

import (
	"encoding/json"
	"testing"
)

// nestedResources is a state resource list with a module nested inside
// another module: module.app contains aws_lb.main and module.db, and
// module.db contains two resources.
const nestedResources = `[
  {"mode":"managed","type":"aws_lb","name":"main","module":"module.app","instances":[{"attributes":{}}]},
  {"mode":"managed","type":"aws_db_instance","name":"primary","module":"module.app.module.db","instances":[{"attributes":{}}]},
  {"mode":"managed","type":"aws_db_subnet_group","name":"main","module":"module.app.module.db","instances":[{"attributes":{}}]}
]`

// findNode returns the first node in the tree whose value matches, or nil.
func findNode(nodes []*treeNode, value string) *treeNode {
	for _, n := range nodes {
		if n.value == value {
			return n
		}
		if found := findNode(n.children, value); found != nil {
			return found
		}
	}
	return nil
}

func parseResources(t *testing.T, arrayJSON string) []Resource {
	t.Helper()
	var rs []Resource
	if err := json.Unmarshal([]byte(arrayJSON), &rs); err != nil {
		t.Fatalf("parse resources: %v", err)
	}
	return rs
}

func TestBuildResourceTree(t *testing.T) {
	var state TerraformState
	if err := json.Unmarshal([]byte(sampleState), &state); err != nil {
		t.Fatalf("unmarshal sample: %v", err)
	}

	tree := buildResourceTree(state.Resources)

	// Top level: the module sorts before the four root resources.
	if len(tree) != 5 {
		t.Fatalf("want 5 top-level rows, got %d", len(tree))
	}
	if !tree[0].isModule || tree[0].value != "module.network" || tree[0].level != 0 {
		t.Errorf("first row should be the module.network group, got %+v", tree[0])
	}

	// Count-indexed instances become a leaf each.
	for _, want := range []string{"aws_instance.web[0]", "aws_instance.web[1]"} {
		n := findNode(tree, want)
		if n == nil || n.isModule {
			t.Errorf("missing resource leaf %q", want)
		}
	}

	// The module's resource is a child leaf with the full address as value.
	vpc := findNode(tree, "module.network.aws_vpc.main")
	if vpc == nil || vpc.isModule || vpc.level != 1 {
		t.Fatalf("module child not nested correctly: %+v", vpc)
	}
	if rc := resourceCount(tree[0]); rc != 1 {
		t.Errorf("module.network resource count = %d, want 1", rc)
	}
}

func TestBuildResourceTreeNested(t *testing.T) {
	tree := buildResourceTree(parseResources(t, nestedResources))

	if len(tree) != 1 || tree[0].value != "module.app" {
		t.Fatalf("want a single top-level module.app, got %+v", tree)
	}
	app := tree[0]
	if app.level != 0 || resourceCount(app) != 3 {
		t.Errorf("module.app: level %d, count %d (want 0, 3)", app.level, resourceCount(app))
	}

	// module.app contains the nested module.db (sorted first) and aws_lb.main.
	if len(app.children) != 2 || !app.children[0].isModule || app.children[1].isModule {
		t.Fatalf("module.app children wrong: %+v", app.children)
	}

	db := findNode(tree, "module.app.module.db")
	if db == nil || !db.isModule || db.level != 1 || resourceCount(db) != 2 {
		t.Fatalf("nested module.db wrong: %+v", db)
	}

	primary := findNode(tree, "module.app.module.db.aws_db_instance.primary")
	if primary == nil || primary.isModule || primary.level != 2 {
		t.Errorf("deeply nested resource wrong: %+v", primary)
	}
}

func TestFlattenVisible(t *testing.T) {
	tree := buildResourceTree(parseResources(t, nestedResources))

	// Modules start collapsed: only the top-level row is visible.
	rows := flattenVisible(tree)
	if len(rows) != 1 || rows[0].value != "module.app" {
		t.Fatalf("collapsed tree should show only module.app, got %d rows", len(rows))
	}

	// Expanding module.app reveals module.db and aws_lb.main.
	tree[0].expanded = true
	rows = flattenVisible(tree)
	got := rowValues(rows)
	want := []string{"module.app", "module.app.module.db", "module.app.aws_lb.main"}
	if !equalStrings(got, want) {
		t.Errorf("after expanding module.app: got %v, want %v", got, want)
	}

	// Expanding the nested module.db reveals its two resources.
	findNode(tree, "module.app.module.db").expanded = true
	rows = flattenVisible(tree)
	got = rowValues(rows)
	want = []string{
		"module.app",
		"module.app.module.db",
		"module.app.module.db.aws_db_instance.primary",
		"module.app.module.db.aws_db_subnet_group.main",
		"module.app.aws_lb.main",
	}
	if !equalStrings(got, want) {
		t.Errorf("after expanding module.db: got %v, want %v", got, want)
	}
}

func TestNodeSelected(t *testing.T) {
	tree := buildResourceTree(parseResources(t, nestedResources))
	db := findNode(tree, "module.app.module.db")

	selected := map[string]bool{}
	if nodeSelected(db, selected) {
		t.Error("empty selection should not mark the module selected")
	}

	for _, leaf := range leafValues(db) {
		selected[leaf] = true
	}
	if !nodeSelected(db, selected) {
		t.Error("module should be selected once all its leaves are")
	}
	// A module is selected only when every leaf is.
	selected[leafValues(db)[0]] = false
	if nodeSelected(db, selected) {
		t.Error("module should not be selected with a leaf deselected")
	}
}

func TestSelectedResourcePaths(t *testing.T) {
	selected := map[string]bool{
		"module.network.aws_vpc.main": true,
		"aws_instance.web[0]":         true,
		"data.aws_ami.ubuntu":         false,
	}

	got := selectedResourcePaths(selected)
	want := []string{"aws_instance.web[0]", "module.network.aws_vpc.main"}
	if !equalStrings(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func rowValues(rows []*treeNode) []string {
	values := make([]string, len(rows))
	for i, r := range rows {
		values[i] = r.value
	}
	return values
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
