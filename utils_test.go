package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFormatModulePath(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"network", "module.network"},
		{"module.network", "module.network"},
		{"module.a.module.b", "module.a.module.b"},
	}
	for _, c := range cases {
		if got := formatModulePath(c.in); got != c.want {
			t.Errorf("formatModulePath(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestResourceAddress(t *testing.T) {
	cases := []struct {
		name string
		r    Resource
		want string
	}{
		{"managed root", Resource{Mode: "managed", Type: "aws_instance", Name: "web"}, "aws_instance.web"},
		{"data root", Resource{Mode: "data", Type: "aws_ami", Name: "ubuntu"}, "data.aws_ami.ubuntu"},
		{"managed in module", Resource{Module: "module.network", Mode: "managed", Type: "aws_vpc", Name: "main"}, "module.network.aws_vpc.main"},
		{"bare module name", Resource{Module: "network", Mode: "managed", Type: "aws_vpc", Name: "main"}, "module.network.aws_vpc.main"},
		{"data in module", Resource{Module: "module.a", Mode: "data", Type: "aws_ami", Name: "u"}, "module.a.data.aws_ami.u"},
	}
	for _, c := range cases {
		if got := resourceAddress(c.r); got != c.want {
			t.Errorf("%s: resourceAddress = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestResourceLevelAddress(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"aws_instance.web", "aws_instance.web"},
		{"aws_instance.web[0]", "aws_instance.web"},
		{`aws_instance.web["primary"]`, "aws_instance.web"},
		{"module.m.aws_x.y", "module.m.aws_x.y"},
		{"module.m.aws_x.y[2]", "module.m.aws_x.y"},
		{`module.m["k"].aws_x.y`, `module.m["k"].aws_x.y`},
		{`module.m["k"].aws_x.y["i"]`, `module.m["k"].aws_x.y`},
	}
	for _, c := range cases {
		if got := resourceLevelAddress(c.in); got != c.want {
			t.Errorf("resourceLevelAddress(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestReadStateFile(t *testing.T) {
	dir := t.TempDir()

	t.Run("valid", func(t *testing.T) {
		path := writeTempState(t, dir, "valid.tfstate", sampleState)
		state, err := readStateFile(path)
		if err != nil {
			t.Fatalf("readStateFile: %v", err)
		}
		if state.Version != 4 || len(state.Resources) != 4 {
			t.Errorf("got version %d, %d resources", state.Version, len(state.Resources))
		}
	})

	t.Run("missing file", func(t *testing.T) {
		if _, err := readStateFile(filepath.Join(dir, "does-not-exist.tfstate")); err == nil {
			t.Error("expected an error for a missing file")
		}
	})

	t.Run("malformed json", func(t *testing.T) {
		path := writeTempState(t, dir, "bad.tfstate", "{not valid json")
		if _, err := readStateFile(path); err == nil {
			t.Error("expected an error for malformed JSON")
		}
	})

	t.Run("unsupported version", func(t *testing.T) {
		path := writeTempState(t, dir, "v3.tfstate", `{"version": 3, "resources": []}`)
		_, err := readStateFile(path)
		if err == nil || !strings.Contains(err.Error(), "unsupported state format version") {
			t.Errorf("expected an unsupported-version error, got %v", err)
		}
	})
}

func TestWriteStateFile(t *testing.T) {
	dir := t.TempDir()
	src := writeTempState(t, dir, "src.tfstate", sampleState)
	state, err := readStateFile(src)
	if err != nil {
		t.Fatalf("readStateFile: %v", err)
	}

	dst := filepath.Join(dir, "dst.tfstate")
	if err = writeStateFile(dst, state); err != nil {
		t.Fatalf("writeStateFile: %v", err)
	}

	reread, err := readStateFile(dst)
	if err != nil {
		t.Fatalf("re-read written state: %v", err)
	}
	if reread.Version != 4 || len(reread.Resources) != 4 {
		t.Errorf("written state changed: version %d, %d resources", reread.Version, len(reread.Resources))
	}
}

func TestMoveResources(t *testing.T) {
	// move sets up a fresh input file and runs moveResources, returning the
	// resulting input and output paths.
	move := func(t *testing.T, selected []string, dryRun bool) (in, out string) {
		t.Helper()
		dir := t.TempDir()
		in = writeTempState(t, dir, "in.tfstate", sampleState)
		out = filepath.Join(dir, "out.tfstate")
		if err := moveResources(selected, in, out, dryRun); err != nil {
			t.Fatalf("moveResources: %v", err)
		}
		return in, out
	}

	t.Run("moves a root resource keeping every instance attribute", func(t *testing.T) {
		in, out := move(t, []string{"aws_instance.web"}, false)

		outState, err := readStateFile(out)
		if err != nil {
			t.Fatalf("read output: %v", err)
		}
		if len(outState.Resources) != 1 {
			t.Fatalf("want 1 resource moved, got %d", len(outState.Resources))
		}
		if got := resourceAddress(outState.Resources[0]); got != "aws_instance.web" {
			t.Fatalf("moved the wrong resource: %s", got)
		}

		// The core regression: instance attributes must survive the move.
		m := decodeJSONFile(t, out)
		insts, ok := dig(t, m, "resources", 0, "instances").([]any)
		if !ok || len(insts) != 2 {
			t.Fatalf("want 2 instances with data, got %v", insts)
		}
		if id := dig(t, m, "resources", 0, "instances", 0, "attributes", "id"); id != "i-aaa" {
			t.Errorf("instance 0 attributes lost: id = %v", id)
		}
		if id := dig(t, m, "resources", 0, "instances", 1, "attributes", "id"); id != "i-bbb" {
			t.Errorf("instance 1 attributes lost: id = %v", id)
		}

		inState, err := readStateFile(in)
		if err != nil {
			t.Fatalf("read input: %v", err)
		}
		if len(inState.Resources) != 3 {
			t.Fatalf("want 3 resources kept, got %d", len(inState.Resources))
		}

		for _, s := range []*TerraformState{inState, outState} {
			if s.Version != 4 {
				t.Errorf("version = %d, want 4", s.Version)
			}
			if s.TerraformVersion != "1.9.5" {
				t.Errorf("terraform_version = %q, want 1.9.5", s.TerraformVersion)
			}
			if s.Lineage != sampleLineage {
				t.Errorf("lineage = %q, want %q", s.Lineage, sampleLineage)
			}
			if s.Serial != 8 {
				t.Errorf("serial = %d, want 8 (bumped from 7)", s.Serial)
			}
			if len(s.Outputs) == 0 {
				t.Error("outputs not preserved")
			}
		}
	})

	t.Run("moves a resource inside a module", func(t *testing.T) {
		_, out := move(t, []string{"module.network.aws_vpc.main"}, false)
		outState, err := readStateFile(out)
		if err != nil {
			t.Fatalf("read output: %v", err)
		}
		if len(outState.Resources) != 1 || resourceAddress(outState.Resources[0]) != "module.network.aws_vpc.main" {
			t.Errorf("module resource not moved: %+v", outState.Resources)
		}
	})

	t.Run("moves a data source", func(t *testing.T) {
		_, out := move(t, []string{"data.aws_ami.ubuntu"}, false)
		outState, err := readStateFile(out)
		if err != nil {
			t.Fatalf("read output: %v", err)
		}
		if len(outState.Resources) != 1 || resourceAddress(outState.Resources[0]) != "data.aws_ami.ubuntu" {
			t.Errorf("data source not moved: %+v", outState.Resources)
		}
	})

	t.Run("moves a non-AWS resource", func(t *testing.T) {
		_, out := move(t, []string{"google_storage_bucket.assets"}, false)
		outState, err := readStateFile(out)
		if err != nil {
			t.Fatalf("read output: %v", err)
		}
		if len(outState.Resources) != 1 || resourceAddress(outState.Resources[0]) != "google_storage_bucket.assets" {
			t.Errorf("non-AWS resource not moved: %+v", outState.Resources)
		}
	})

	t.Run("an instance-indexed selection moves the whole resource", func(t *testing.T) {
		_, out := move(t, []string{"aws_instance.web[0]"}, false)
		outState, err := readStateFile(out)
		if err != nil {
			t.Fatalf("read output: %v", err)
		}
		if len(outState.Resources) != 1 || len(outState.Resources[0].Instances) != 2 {
			t.Errorf("indexed selection should move the whole resource, got %+v", outState.Resources)
		}
	})

	t.Run("dry run writes nothing", func(t *testing.T) {
		in, out := move(t, []string{"aws_instance.web"}, true)

		if _, err := os.Stat(out); !os.IsNotExist(err) {
			t.Errorf("output file should not exist after a dry run (err = %v)", err)
		}
		inState, err := readStateFile(in)
		if err != nil {
			t.Fatalf("read input: %v", err)
		}
		if len(inState.Resources) != 4 || inState.Serial != 7 {
			t.Errorf("input file changed by a dry run: %d resources, serial %d", len(inState.Resources), inState.Serial)
		}
	})

	t.Run("a selection matching nothing yields an empty output state", func(t *testing.T) {
		in, out := move(t, []string{"aws_instance.absent"}, false)
		outState, err := readStateFile(out)
		if err != nil {
			t.Fatalf("read output: %v", err)
		}
		if len(outState.Resources) != 0 {
			t.Errorf("want 0 resources moved, got %d", len(outState.Resources))
		}
		inState, err := readStateFile(in)
		if err != nil {
			t.Fatalf("read input: %v", err)
		}
		if len(inState.Resources) != 4 {
			t.Errorf("want all 4 resources kept, got %d", len(inState.Resources))
		}
	})
}
