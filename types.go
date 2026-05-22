package main

import "encoding/json"

// stateVersion is the Terraform state format version this tool supports.
// Format 4 has been current since Terraform 0.13 and is still the format
// written by the latest Terraform 1.x releases.
const stateVersion = 4

// TerraformState is a parsed Terraform state file.
//
// Every top-level field is preserved across a read/write round-trip — this
// tool only ever changes which resources live in the "resources" array, so
// version, lineage, serial, and outputs survive untouched.
type TerraformState struct {
	Version          int             `json:"version"`
	TerraformVersion string          `json:"terraform_version,omitempty"`
	Serial           int             `json:"serial"`
	Lineage          string          `json:"lineage,omitempty"`
	Outputs          json.RawMessage `json:"outputs,omitempty"`
	Resources        []Resource      `json:"resources"`
	CheckResults     json.RawMessage `json:"check_results,omitempty"`
}

// Resource is a single entry in the state's "resources" array.
//
// The complete original JSON of the resource is kept in raw, so a resource
// can be relocated to another state file without losing instance attributes,
// dependencies, schema versions, or any field this tool does not model.
type Resource struct {
	Module    string
	Mode      string
	Type      string
	Name      string
	Instances []Instance

	// raw is the verbatim JSON the resource was parsed from. It is what gets
	// written back out, which is correct because resources are relocated
	// between files unchanged.
	raw json.RawMessage
}

// resourceMeta is the subset of resource fields this tool reads. Keeping it a
// distinct type lets Resource.UnmarshalJSON decode identity fields without
// recursing into itself.
type resourceMeta struct {
	Module    string     `json:"module,omitempty"`
	Mode      string     `json:"mode"`
	Type      string     `json:"type"`
	Name      string     `json:"name"`
	Instances []Instance `json:"instances,omitempty"`
}

// UnmarshalJSON records the resource's raw JSON and decodes its identity
// fields.
func (r *Resource) UnmarshalJSON(data []byte) error {
	r.raw = append(json.RawMessage(nil), data...)

	var meta resourceMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return err
	}
	r.Module = meta.Module
	r.Mode = meta.Mode
	r.Type = meta.Type
	r.Name = meta.Name
	r.Instances = meta.Instances
	return nil
}

// MarshalJSON writes the resource's preserved raw JSON. A resource built in
// code rather than parsed (no raw bytes) falls back to encoding the modelled
// fields.
func (r Resource) MarshalJSON() ([]byte, error) {
	if len(r.raw) > 0 {
		return r.raw, nil
	}
	return json.Marshal(resourceMeta{
		Module:    r.Module,
		Mode:      r.Mode,
		Type:      r.Type,
		Name:      r.Name,
		Instances: r.Instances,
	})
}

// Instance is the slice of a resource instance this tool needs to render the
// selection UI. Instances are never modified on their own; the parent
// Resource keeps the full JSON.
type Instance struct {
	// IndexKey is the count index (a number) or for_each key (a string) of
	// the instance, or nil for a single-instance resource.
	IndexKey interface{} `json:"index_key,omitempty"`
}

// treeNode is a node in the interactive resource-selection tree. A node is
// either a module group (isModule, with children) or a resource leaf. Modules
// nest to any depth, so the tree mirrors the module hierarchy of the state.
type treeNode struct {
	label    string // text shown for the row
	value    string // module path, or the full resource address for a leaf
	isModule bool
	level    int  // depth in the tree; 0 is the top level
	expanded bool // module rows only: whether the children are shown
	children []*treeNode
}
