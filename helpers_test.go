package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// sampleLineage is the lineage embedded in sampleState. Keep the two in sync.
const sampleLineage = "4f9b1d2a-0000-1111-2222-333344445555"

// sampleState is a representative Terraform state file (format version 4). It
// deliberately covers every shape this tool has to handle: a resource inside
// a module, a count-indexed resource, a data source, and a non-AWS provider.
const sampleState = `{
  "version": 4,
  "terraform_version": "1.9.5",
  "serial": 7,
  "lineage": "4f9b1d2a-0000-1111-2222-333344445555",
  "outputs": {
    "vpc_id": {
      "value": "vpc-0abc",
      "type": "string"
    }
  },
  "resources": [
    {
      "module": "module.network",
      "mode": "managed",
      "type": "aws_vpc",
      "name": "main",
      "provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
      "instances": [
        {
          "schema_version": 1,
          "attributes": {
            "id": "vpc-0abc",
            "cidr_block": "10.0.0.0/16"
          },
          "sensitive_attributes": [],
          "dependencies": []
        }
      ]
    },
    {
      "mode": "managed",
      "type": "aws_instance",
      "name": "web",
      "provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
      "instances": [
        {
          "index_key": 0,
          "schema_version": 1,
          "attributes": {
            "id": "i-aaa",
            "instance_type": "t3.micro"
          }
        },
        {
          "index_key": 1,
          "schema_version": 1,
          "attributes": {
            "id": "i-bbb",
            "instance_type": "t3.micro"
          }
        }
      ]
    },
    {
      "mode": "data",
      "type": "aws_ami",
      "name": "ubuntu",
      "provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
      "instances": [
        {
          "schema_version": 0,
          "attributes": {
            "id": "ami-0xyz"
          }
        }
      ]
    },
    {
      "mode": "managed",
      "type": "google_storage_bucket",
      "name": "assets",
      "provider": "provider[\"registry.terraform.io/hashicorp/google\"]",
      "instances": [
        {
          "schema_version": 0,
          "attributes": {
            "name": "my-assets"
          }
        }
      ]
    }
  ],
  "check_results": null
}`

// writeTempState writes content to a file named name inside dir and returns
// the full path.
func writeTempState(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return path
}

// decodeJSONFile reads path and unmarshals it into a generic map.
func decodeJSONFile(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	return m
}

// dig walks a decoded JSON value following path elements: a string indexes an
// object, an int indexes an array. It fails the test on a type or bounds
// mismatch.
func dig(t *testing.T, v any, path ...any) any {
	t.Helper()
	for _, p := range path {
		switch key := p.(type) {
		case string:
			obj, ok := v.(map[string]any)
			if !ok {
				t.Fatalf("dig: expected object before key %q, got %T", key, v)
			}
			v = obj[key]
		case int:
			arr, ok := v.([]any)
			if !ok {
				t.Fatalf("dig: expected array before index %d, got %T", key, v)
			}
			if key < 0 || key >= len(arr) {
				t.Fatalf("dig: index %d out of range (len %d)", key, len(arr))
			}
			v = arr[key]
		default:
			t.Fatalf("dig: unsupported path element %T", p)
		}
	}
	return v
}

// jsonEqual reports whether two JSON documents are semantically equal,
// ignoring key order and insignificant whitespace.
func jsonEqual(t *testing.T, a, b []byte) bool {
	t.Helper()
	var av, bv any
	if err := json.Unmarshal(a, &av); err != nil {
		t.Fatalf("decode left: %v", err)
	}
	if err := json.Unmarshal(b, &bv); err != nil {
		t.Fatalf("decode right: %v", err)
	}
	return reflect.DeepEqual(av, bv)
}
