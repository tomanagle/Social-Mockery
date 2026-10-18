package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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

func TestUploadContract(t *testing.T) {
	for _, tc := range []struct {
		name, route, server string
		valid               bool
	}{
		{"valid", "/ig-api-upload/v26.0/{containerID}", "https://rupload.facebook.com", true},
		{"wrong version", "/ig-api-upload/v25.0/{containerID}", "https://rupload.facebook.com", false},
		{"unversioned", "/ig-api-upload/{containerID}", "https://rupload.facebook.com", false},
		{"graph path", "/v26.0/{containerID}", "https://rupload.facebook.com", false},
		{"extra suffix", "/ig-api-upload/v26.0/{containerID}/anything", "https://rupload.facebook.com", false},
		{"graph host", "/ig-api-upload/v26.0/{containerID}", "https://graph.facebook.com", false},
		{"lookalike host", "/ig-api-upload/v26.0/{containerID}", "https://rupload.facebook.com.evil.test", false},
		{"http", "/ig-api-upload/v26.0/{containerID}", "http://rupload.facebook.com", false},
		{"server path", "/ig-api-upload/v26.0/{containerID}", "https://rupload.facebook.com/other", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := contractFixture()
			op := fixtureOperation(d)
			op["parameters"] = []any{map[string]any{"name": "containerID", "in": "path", "required": true, "schema": map[string]any{"type": "string"}}}
			d["paths"] = map[string]any{tc.route: map[string]any{"post": op}}
			d["x-contract-kind"] = "upload"
			d["servers"] = []any{map[string]any{"url": tc.server}}
			dir := filepath.Join(t.TempDir(), "v26.0")
			if err := os.Mkdir(dir, 0700); err != nil {
				t.Fatal(err)
			}
			data, err := json.Marshal(d)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(dir, "upload.openapi.yaml")
			if err = os.WriteFile(path, data, 0600); err != nil {
				t.Fatal(err)
			}
			if err = validate(path); (err == nil) != tc.valid {
				t.Fatalf("valid=%v error=%v", tc.valid, err)
			}
		})
	}
}

func providerFixture(provider, kind, version, route, server string) map[string]any {
	d := contractFixture()
	op := fixtureOperation(d)
	d["x-provider"] = provider
	d["x-contract-kind"] = kind
	d["info"].(map[string]any)["version"] = version
	d["servers"] = []any{map[string]any{"url": server}}
	if provider == "tiktok" {
		op["x-sources"] = []any{"https://developers.tiktok.com/doc/overview/"}
		if kind == "upload" {
			d["x-dynamic-upload-url"] = true
		}
	}
	parameters := []any{}
	for _, match := range regexp.MustCompile(`\{([^}]+)\}`).FindAllStringSubmatch(route, -1) {
		parameters = append(parameters, map[string]any{
			"name": match[1], "in": "path", "required": true,
			"schema": map[string]any{"type": "string"},
		})
	}
	if len(parameters) > 0 {
		op["parameters"] = parameters
	}
	d["paths"] = map[string]any{route: map[string]any{"post": op}}
	return d
}

func validateFixture(t *testing.T, d map[string]any, directory, version string) error {
	t.Helper()
	dir := filepath.Join(t.TempDir(), directory, version)
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "test.openapi.yaml")
	if err = os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	return validate(path)
}

func TestProviderRoutes(t *testing.T) {
	for _, tc := range []struct {
		name, provider, kind, version, route, server string
		valid                                        bool
	}{
		{"facebook graph", "facebook", "graph", "v26.0", "/v26.0/me", "https://graph.facebook.com", true},
		{"facebook video", "facebook", "graph", "v26.0", "/v26.0/me/videos", "https://graph-video.facebook.com", true},
		{"facebook wrong release", "facebook", "graph", "v26.0", "/v25.0/me", "https://graph.facebook.com", false},
		{"facebook oauth", "facebook", "oauth", "v26.0", "/v26.0/dialog/oauth", "https://www.facebook.com", true},
		{"facebook token", "facebook", "oauth", "v26.0", "/v26.0/oauth/access_token", "https://graph.facebook.com", true},
		{"facebook instagram oauth", "facebook", "oauth", "v26.0", "/oauth/access_token", "https://graph.facebook.com", false},
		{"facebook webhook", "facebook", "webhook", "v26.0", "/webhooks/facebook", "https://example.invalid", true},
		{"facebook instagram webhook", "facebook", "webhook", "v26.0", "/webhooks/instagram", "https://example.invalid", false},
		{"facebook tiktok webhook", "facebook", "webhook", "v26.0", "/webhooks/tiktok", "https://example.invalid", false},
		{"facebook versioned webhook", "facebook", "webhook", "v26.0", "/v26.0/me", "https://example.invalid", false},
		{"facebook upload", "facebook", "upload", "v26.0", "/video-upload/v26.0/{videoID}", "https://rupload.facebook.com", true},
		{"facebook old upload", "facebook", "upload", "v24.0", "/video-upload/v24.0/{videoID}", "https://rupload.facebook.com", true},
		{"facebook wrong upload release", "facebook", "upload", "v26.0", "/video-upload/v25.0/{videoID}", "https://rupload.facebook.com", false},
		{"facebook instagram upload", "facebook", "upload", "v26.0", "/ig-api-upload/v26.0/{containerID}", "https://rupload.facebook.com", false},
		{"facebook dynamic upload", "facebook", "upload", "v26.0", "/{uploadPath}", "https://rupload.facebook.com", false},
		{"instagram facebook upload", "instagram", "upload", "v26.0", "/video-upload/v26.0/{videoID}", "https://rupload.facebook.com", false},
		{"instagram dynamic upload", "instagram", "upload", "v26.0", "/{uploadPath}", "https://rupload.facebook.com", false},
		{"instagram facebook webhook", "instagram", "webhook", "v26.0", "/webhooks/facebook", "https://example.invalid", false},
		{"instagram tiktok webhook", "instagram", "webhook", "v26.0", "/webhooks/tiktok", "https://example.invalid", false},
		{"tiktok graph", "tiktok", "graph", "v2", "/v2/user/info/", "https://open.tiktokapis.com", true},
		{"tiktok wrong release", "tiktok", "graph", "v2", "/v1/user/info/", "https://open.tiktokapis.com", false},
		{"tiktok authorize", "tiktok", "oauth", "v2", "/v2/auth/authorize/", "https://www.tiktok.com", true},
		{"tiktok token", "tiktok", "oauth", "v2", "/v2/oauth/token/", "https://open.tiktokapis.com", true},
		{"tiktok revoke", "tiktok", "oauth", "v2", "/v2/oauth/revoke/", "https://open.tiktokapis.com", true},
		{"tiktok instagram oauth", "tiktok", "oauth", "v2", "/oauth/access_token", "https://open.tiktokapis.com", false},
		{"tiktok webhook", "tiktok", "webhook", "v2", "/webhooks/tiktok", "https://example.invalid", true},
		{"tiktok facebook webhook", "tiktok", "webhook", "v2", "/webhooks/facebook", "https://example.invalid", false},
		{"tiktok instagram webhook", "tiktok", "webhook", "v2", "/webhooks/instagram", "https://example.invalid", false},
		{"tiktok versioned webhook", "tiktok", "webhook", "v2", "/v2/webhooks/tiktok", "https://example.invalid", false},
		{"tiktok dynamic upload", "tiktok", "upload", "v2", "/{uploadPath}", "https://open-upload.tiktokapis.com", true},
		{"tiktok regional upload", "tiktok", "upload", "v2", "/{uploadPath}", "https://upload.us.tiktokapis.com", true},
		{"tiktok future upload", "tiktok", "upload", "v3", "/{uploadPath}", "https://open-upload.tiktokapis.com", false},
		{"tiktok extra upload suffix", "tiktok", "upload", "v2", "/{uploadPath}/anything", "https://open-upload.tiktokapis.com", false},
		{"tiktok versioned upload", "tiktok", "upload", "v2", "/v2/{uploadPath}", "https://open-upload.tiktokapis.com", false},
		{"tiktok meta upload", "tiktok", "upload", "v2", "/video-upload/v2/{videoID}", "https://open-upload.tiktokapis.com", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := providerFixture(tc.provider, tc.kind, tc.version, tc.route, tc.server)
			if err := validateFixture(t, d, tc.provider, tc.version); (err == nil) != tc.valid {
				t.Fatalf("valid=%v, error=%v", tc.valid, err)
			}
		})
	}
}

func TestProviderMarkers(t *testing.T) {
	for _, tc := range []struct {
		name, directory string
		marker          any
		valid           bool
	}{
		{"legacy instagram", "instagram", nil, true},
		{"explicit instagram", "instagram", "instagram", true},
		{"explicit facebook", "facebook", "facebook", true},
		{"explicit tiktok", "tiktok", "tiktok", true},
		{"facebook missing marker", "facebook", nil, false},
		{"tiktok missing marker", "tiktok", nil, false},
		{"facebook mislabeled instagram", "facebook", "instagram", false},
		{"instagram mislabeled facebook", "instagram", "facebook", false},
		{"tiktok mislabeled facebook", "tiktok", "facebook", false},
		{"facebook mislabeled tiktok", "facebook", "tiktok", false},
		{"unknown provider", "instagram", "other", false},
		{"empty provider", "instagram", "", false},
		{"nonstring provider", "instagram", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			version, server := "v26.0", "https://graph.facebook.com"
			if tc.directory == "tiktok" {
				version, server = "v2", "https://open.tiktokapis.com"
			}
			d := providerFixture(tc.directory, "graph", version, "/"+version+"/me", server)
			if tc.marker == nil {
				delete(d, "x-provider")
			} else {
				d["x-provider"] = tc.marker
			}
			if err := validateFixture(t, d, tc.directory, version); (err == nil) != tc.valid {
				t.Fatalf("valid=%v, error=%v", tc.valid, err)
			}
		})
	}
}

func TestProviderServerBoundaries(t *testing.T) {
	for _, tc := range []struct {
		provider, kind, server string
		valid                  bool
	}{
		{"instagram", "graph", "https://graph.facebook.com", true},
		{"instagram", "graph", "https://graph-video.facebook.com", false},
		{"instagram", "graph", "https://open.tiktokapis.com", false},
		{"instagram", "oauth", "https://www.tiktok.com", false},
		{"instagram", "upload", "https://open-upload.tiktokapis.com", false},
		{"instagram", "upload", "https://upload.us.tiktokapis.com", false},
		{"facebook", "graph", "https://graph.instagram.com", false},
		{"facebook", "graph", "https://open.tiktokapis.com", false},
		{"facebook", "graph", "https://www.facebook.com", false},
		{"facebook", "oauth", "https://www.instagram.com", false},
		{"facebook", "oauth", "https://www.tiktok.com", false},
		{"facebook", "webhook", "http://localhost:3000", false},
		{"facebook", "webhook", "https://graph.facebook.com", false},
		{"facebook", "upload", "https://open-upload.tiktokapis.com", false},
		{"facebook", "upload", "https://upload.us.tiktokapis.com", false},
		{"facebook", "upload", "https://rupload.facebook.com/other", false},
		{"tiktok", "graph", "https://graph.facebook.com", false},
		{"tiktok", "graph", "https://www.tiktok.com", false},
		{"tiktok", "graph", "https://example.invalid", false},
		{"tiktok", "oauth", "https://api.instagram.com", false},
		{"tiktok", "webhook", "http://localhost:3000", false},
		{"tiktok", "webhook", "https://open.tiktokapis.com", false},
		{"tiktok", "upload", "https://rupload.facebook.com", false},
		{"tiktok", "upload", "https://open.tiktokapis.com", false},
		{"tiktok", "upload", "https://open-upload.tiktokapis.com.evil.test", false},
		{"tiktok", "upload", "https://open-upload.tiktokapis.com/other", false},
		{"tiktok", "upload", "https://upload.eu.tiktokapis.com", false},
		{"tiktok", "upload", "http://open-upload.tiktokapis.com", false},
		{"facebook", "graph", "https://graph.facebook.com/v25.0", false},
		{"tiktok", "graph", "https://open.tiktokapis.com/v1", false},
		{"tiktok", "graph", "http://open.tiktokapis.com", false},
		{"tiktok", "graph", "https://open.tiktokapis.com.evil.test", false},
		{"tiktok", "graph", "https://user@open.tiktokapis.com", false},
		{"tiktok", "graph", "https://open.tiktokapis.com:443", false},
		{"tiktok", "graph", "https://open.tiktokapis.com?query=1", false},
		{"tiktok", "graph", "https://open.tiktokapis.com#fragment", false},
	} {
		t.Run(tc.provider+"/"+tc.kind+"/"+tc.server, func(t *testing.T) {
			for _, level := range []string{"document", "path", "operation"} {
				t.Run(level, func(t *testing.T) {
					version, route, server := "v26.0", "/v26.0/me", "https://graph.facebook.com"
					if tc.provider == "tiktok" {
						version, route, server = "v2", "/v2/user/info/", "https://open.tiktokapis.com"
					}
					if tc.kind == "webhook" {
						route, server = "/webhooks/"+tc.provider, "https://example.invalid"
					}
					if tc.kind == "upload" {
						server = "https://rupload.facebook.com"
						switch tc.provider {
						case "instagram":
							route = "/ig-api-upload/" + version + "/{containerID}"
						case "facebook":
							route = "/video-upload/" + version + "/{videoID}"
						case "tiktok":
							route, server = "/{uploadPath}", "https://open-upload.tiktokapis.com"
						}
					}
					d := providerFixture(tc.provider, tc.kind, version, route, server)
					target := d
					if level != "document" {
						target = d["paths"].(map[string]any)[route].(map[string]any)
						if level == "operation" {
							target = target["post"].(map[string]any)
						}
					}
					target["servers"] = []any{map[string]any{"url": tc.server}}
					if err := validateFixture(t, d, tc.provider, version); (err == nil) != tc.valid {
						t.Fatalf("valid=%v, error=%v", tc.valid, err)
					}
				})
			}
		})
	}
}

func TestProviderSources(t *testing.T) {
	for _, tc := range []struct {
		provider, source string
		valid            bool
	}{
		{"instagram", "https://developers.facebook.com/docs/instagram-platform/", true},
		{"facebook", "https://developers.meta.com/docs/graph-api/", true},
		{"facebook", "https://www.postman.com/meta/facebook/documentation/", true},
		{"facebook", "https://github.com/facebook/facebook-nodejs-business-sdk/blob/revision/src/api.js", true},
		{"facebook", "https://raw.githubusercontent.com/facebook/facebook-nodejs-business-sdk/revision/src/api.js", true},
		{"tiktok", "https://developers.tiktok.com/doc/content-posting-api/", true},
		{"tiktok", "https://github.com/tiktok/tiktok-opensdk-ios/blob/revision/README.md", true},
		{"instagram", "https://developers.tiktok.com/doc/overview/", false},
		{"facebook", "https://developers.tiktok.com/doc/overview/", false},
		{"facebook", "https://github.com/tiktok/tiktok-opensdk-ios/", false},
		{"tiktok", "https://developers.facebook.com/docs/graph-api/", false},
		{"tiktok", "https://github.com/facebook/facebook-nodejs-business-sdk/", false},
		{"tiktok", "https://www.postman.com/meta/instagram/", false},
		{"tiktok", "https://developers.tiktok.com.evil.test/doc/overview/", false},
		{"tiktok", "https://github.com/tiktok-evil/project/", false},
		{"tiktok", "https://github.com/other/tiktok/", false},
		{"tiktok", "http://developers.tiktok.com/doc/overview/", false},
		{"tiktok", "https://user@developers.tiktok.com/doc/overview/", false},
		{"tiktok", "https://developers.tiktok.com:443/doc/overview/", false},
	} {
		t.Run(tc.provider+"/"+tc.source, func(t *testing.T) {
			version, route, server := "v26.0", "/v26.0/me", "https://graph.facebook.com"
			if tc.provider == "tiktok" {
				version, route, server = "v2", "/v2/user/info/", "https://open.tiktokapis.com"
			}
			d := providerFixture(tc.provider, "graph", version, route, server)
			d["paths"].(map[string]any)[route].(map[string]any)["post"].(map[string]any)["x-sources"] = []any{tc.source}
			if err := validateFixture(t, d, tc.provider, version); (err == nil) != tc.valid {
				t.Fatalf("valid=%v, error=%v", tc.valid, err)
			}
		})
	}
}

func TestDynamicUploadMarker(t *testing.T) {
	for _, marker := range []any{nil, false, "true", 1} {
		t.Run(fmt.Sprintf("invalid marker %v", marker), func(t *testing.T) {
			d := providerFixture("tiktok", "upload", "v2", "/{uploadPath}", "https://open-upload.tiktokapis.com")
			if marker == nil {
				delete(d, "x-dynamic-upload-url")
			} else {
				d["x-dynamic-upload-url"] = marker
			}
			if err := validateFixture(t, d, "tiktok", "v2"); err == nil {
				t.Fatal("accepted dynamic upload without the explicit true marker")
			}
		})
	}
	for _, provider := range []string{"instagram", "facebook", "tiktok"} {
		t.Run("graph "+provider, func(t *testing.T) {
			version, server := "v26.0", "https://graph.facebook.com"
			if provider == "tiktok" {
				version, server = "v2", "https://open.tiktokapis.com"
			}
			d := providerFixture(provider, "graph", version, "/"+version+"/me", server)
			d["x-dynamic-upload-url"] = true
			if err := validateFixture(t, d, provider, version); err == nil {
				t.Fatal("accepted a dynamic upload exception outside a TikTok upload contract")
			}
		})
	}
}
