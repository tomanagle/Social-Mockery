package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestIndependentContracts(t *testing.T) {
	root := t.TempDir()
	// Different fields and even different operations are legal across versions.
	for i, version := range []string{"v24.0", "v25.0"} {
		dir := filepath.Join(root, version)
		if err := os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(dir, "openapi.yaml")
		data := fmt.Sprintf(`{"openapi":"3.0.3","info":{"title":"test","version":%q},"paths":{%q:{"get":{"operationId":%q,"responses":{"200":{"description":"ok"}}}}}}`, version, "/"+version+"/operation", fmt.Sprintf("Operation%d", i))
		if err := os.WriteFile(path, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
		if err := validate(path); err != nil {
			t.Fatal(err)
		}
	}
}
func TestRejectVersionMismatch(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "v24.0")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	for _, body := range []string{
		`{"openapi":"3.0.3","info":{"title":"test","version":"v25.0"},"paths":{}}`,
		`{"openapi":"3.0.3","info":{"title":"test","version":"v24.0"},"paths":{"/v25.0/object":{"get":{"responses":{"200":{"description":"ok"}}}}}}`,
	} {
		path := filepath.Join(dir, "openapi.yaml")
		if err := os.WriteFile(path, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
		if err := validate(path); err == nil {
			t.Fatal("accepted mismatched version")
		}
	}
}
