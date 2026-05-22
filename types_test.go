package main

import (
	"encoding/json"
	"testing"
)

// TestStateRoundTrip is the regression guard for the data-loss bug: parsing a
// state and writing it back must not drop a single field.
func TestStateRoundTrip(t *testing.T) {
	var state TerraformState
	if err := json.Unmarshal([]byte(sampleState), &state); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	out, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	if !jsonEqual(t, []byte(sampleState), out) {
		t.Errorf("round trip changed the state\n--- want ---\n%s\n--- got ---\n%s", sampleState, out)
	}
}

func TestResourceUnmarshalIdentity(t *testing.T) {
	var state TerraformState
	if err := json.Unmarshal([]byte(sampleState), &state); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(state.Resources) != 4 {
		t.Fatalf("want 4 resources, got %d", len(state.Resources))
	}

	vpc := state.Resources[0]
	if vpc.Module != "module.network" || vpc.Mode != "managed" ||
		vpc.Type != "aws_vpc" || vpc.Name != "main" {
		t.Errorf("vpc identity wrong: %+v", vpc)
	}

	web := state.Resources[1]
	if len(web.Instances) != 2 {
		t.Fatalf("want 2 web instances, got %d", len(web.Instances))
	}
	// JSON numbers decode to float64.
	if web.Instances[0].IndexKey != float64(0) || web.Instances[1].IndexKey != float64(1) {
		t.Errorf("index keys wrong: %v, %v", web.Instances[0].IndexKey, web.Instances[1].IndexKey)
	}
}

// TestResourceKeepsUnmodelledFields proves a resource carries fields this tool
// does not model (provider, instance attributes) through a round trip.
func TestResourceKeepsUnmodelledFields(t *testing.T) {
	var state TerraformState
	if err := json.Unmarshal([]byte(sampleState), &state); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	raw, err := json.Marshal(state.Resources[1]) // aws_instance.web
	if err != nil {
		t.Fatalf("marshal resource: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("decode resource: %v", err)
	}
	if _, ok := got["provider"]; !ok {
		t.Error("provider field dropped from resource")
	}
	insts, ok := got["instances"].([]any)
	if !ok || len(insts) != 2 {
		t.Fatalf("instances not preserved: %v", got["instances"])
	}
	inst0, ok := insts[0].(map[string]any)
	if !ok {
		t.Fatalf("instance 0 not an object: %T", insts[0])
	}
	if _, ok := inst0["attributes"]; !ok {
		t.Error("instance attributes dropped")
	}
}

// TestResourceMarshalFallback covers a Resource built in code rather than
// parsed from JSON: it has no raw bytes and must still encode its fields.
func TestResourceMarshalFallback(t *testing.T) {
	r := Resource{Mode: "managed", Type: "aws_s3_bucket", Name: "logs"}

	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got Resource
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Mode != "managed" || got.Type != "aws_s3_bucket" || got.Name != "logs" {
		t.Errorf("fallback marshal lost data: %+v", got)
	}
}
