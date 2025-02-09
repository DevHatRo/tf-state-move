package main

type TerraformState struct {
	Resources []Resource `json:"resources"`
}

type Resource struct {
	Module    string      `json:"module"`
	Mode      string      `json:"mode"`
	Type      string      `json:"type"`
	Name      string      `json:"name"`
	Index     interface{} `json:"index,omitempty"`
	Instances []Instance  `json:"instances,omitempty"`
}

type Instance struct {
	Index    interface{} `json:"index,omitempty"`
	IndexKey interface{} `json:"index_key,omitempty"`
}

type SelectionItem struct {
	Display    string
	Value      string
	IsModule   bool
	IsExpanded bool
	ModuleName string
	Resources  []string
	IsSelected bool
	Level      int
}
