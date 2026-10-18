package main

import (
	"encoding/json"
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

// A minimal independent full contract, distinct from the runnable subset.
func contractFixture() map[string]any {
	return map[string]any{
		"openapi": "3.0.3", "info": map[string]any{"title": "test", "version": "v26.0"},
		"x-contract-only": true,
		"servers":         []any{map[string]any{"url": "https://graph.instagram.com"}},
		"paths": map[string]any{"/v26.0/me": map[string]any{"get": map[string]any{
			"operationId": "GetMe", "x-behavior-provenance": "documentation-based",
			"x-simulator-implemented": false,
			"x-sources":               []any{"https://developers.facebook.com/docs/instagram-platform/"},
			"responses":               map[string]any{"200": map[string]any{"description": "ok"}},
		}}},
	}
}

func TestFullContractValidation(t *testing.T) {
	cases := []struct {
		name   string
		change func(map[string]any)
		valid  bool
	}{
		{"valid", func(map[string]any) {}, true},
		{"missing contract marker", func(d map[string]any) { delete(d, "x-contract-only") }, false},
		{"missing provenance", func(d map[string]any) { delete(fixtureOperation(d), "x-behavior-provenance") }, false},
		{"unsupported provenance", func(d map[string]any) { fixtureOperation(d)["x-behavior-provenance"] = "live-verified" }, false},
		{"missing simulator marker", func(d map[string]any) { delete(fixtureOperation(d), "x-simulator-implemented") }, false},
		{"implemented marker", func(d map[string]any) { fixtureOperation(d)["x-simulator-implemented"] = true }, false},
		{"missing sources", func(d map[string]any) { delete(fixtureOperation(d), "x-sources") }, false},
		{"empty sources", func(d map[string]any) { fixtureOperation(d)["x-sources"] = []any{} }, false},
		{"nonstring source", func(d map[string]any) { fixtureOperation(d)["x-sources"] = []any{123} }, false},
		{"unsafe source", func(d map[string]any) {
			fixtureOperation(d)["x-sources"] = []any{"https://developers.facebook.com.evil.test/docs"}
		}, false},
		{"missing operation id", func(d map[string]any) { delete(fixtureOperation(d), "operationId") }, false},
		{"duplicate operation id", func(d map[string]any) {
			d["paths"].(map[string]any)["/v26.0/other"] = map[string]any{"get": fixtureOperation(d)}
		}, false},
		{"external ref", func(d map[string]any) {
			fixtureOperation(d)["responses"] = map[string]any{"200": map[string]any{"$ref": "https://example.invalid/response.yaml"}}
		}, false},
		{"unsafe server", func(d map[string]any) {
			d["servers"] = []any{map[string]any{"url": "https://graph.instagram.com.evil.test"}}
		}, false},
		{"unsafe operation server", func(d map[string]any) {
			fixtureOperation(d)["servers"] = []any{map[string]any{"url": "https://evil.test"}}
		}, false},
		{"unsafe path server", func(d map[string]any) {
			d["paths"].(map[string]any)["/v26.0/me"].(map[string]any)["servers"] = []any{map[string]any{"url": "https://evil.test"}}
		}, false},
		{"wrong version", func(d map[string]any) { d["info"].(map[string]any)["version"] = "v25.0" }, false},
		{"wrong route version", func(d map[string]any) {
			paths := d["paths"].(map[string]any)
			paths["/v25.0/me"] = paths["/v26.0/me"]
			delete(paths, "/v26.0/me")
		}, false},
		{"oauth exception", func(d map[string]any) { replaceFixtureRoute(d, "oauth", "/oauth/access_token") }, true},
		{"webhook exception", func(d map[string]any) {
			replaceFixtureRoute(d, "webhook", "/webhooks/instagram")
			d["servers"] = []any{map[string]any{"url": "http://localhost:3000"}}
		}, true},
		{"wrong kind exception", func(d map[string]any) { replaceFixtureRoute(d, "oauth", "/webhooks/instagram") }, false},
		{"no kind exception", func(d map[string]any) { replaceFixtureRoute(d, "", "/oauth/access_token") }, false},
		{"arbitrary unversioned route", func(d map[string]any) { replaceFixtureRoute(d, "oauth", "/admin") }, false},
		{"unknown kind", func(d map[string]any) { d["x-contract-kind"] = "anything" }, false},
		{"local server outside webhook", func(d map[string]any) { d["servers"] = []any{map[string]any{"url": "http://localhost:3000"}} }, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := contractFixture()
			tc.change(d)
			data, err := json.Marshal(d)
			if err != nil {
				t.Fatal(err)
			}
			dir := filepath.Join(t.TempDir(), "v26.0")
			if err = os.Mkdir(dir, 0700); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(dir, "instagram-login.openapi.yaml")
			if err = os.WriteFile(path, data, 0600); err != nil {
				t.Fatal(err)
			}
			err = validate(path)
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v, error=%v", tc.valid, err)
			}
		})
	}
}

func fixtureOperation(d map[string]any) map[string]any {
	return d["paths"].(map[string]any)["/v26.0/me"].(map[string]any)["get"].(map[string]any)
}
func replaceFixtureRoute(d map[string]any, kind, route string) {
	d["x-contract-kind"] = kind
	paths := d["paths"].(map[string]any)
	paths[route] = paths["/v26.0/me"]
	delete(paths, "/v26.0/me")
}
