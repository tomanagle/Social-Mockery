package main

import (
	"encoding/json"
	"github.com/getkin/kin-openapi/openapi3"
	"os"
	"path/filepath"
	"testing"
)

// These checks protect evidenced contract content, not just OpenAPI syntax.
// They do not assert live compatibility with Meta.
func TestInstagramPublishingContractInputs(t *testing.T) {
	for _, version := range []string{"v24.0", "v25.0", "v26.0"} {
		t.Run(version, func(t *testing.T) {
			doc := readContract(t, version, "instagram-login")
			paths := doc["paths"].(map[string]any)
			for edge, fields := range map[string][]string{"media": {"image_url", "video_url", "media_type", "children", "caption"}, "media_publish": {"creation_id"}} {
				op := paths["/"+version+"/{objectID}/"+edge].(map[string]any)["post"].(map[string]any)
				body, ok := op["requestBody"].(map[string]any)
				if !ok {
					t.Fatalf("%s missing requestBody", edge)
				}
				content := body["content"].(map[string]any)
				multipart, ok := content["multipart/form-data"].(map[string]any)
				if !ok {
					t.Fatalf("%s missing multipart form", edge)
				}
				encoded, _ := json.Marshal(multipart)
				var schema map[string]any
				_ = json.Unmarshal(encoded, &schema)
				for _, field := range fields {
					if !hasProperty(schema, field) {
						t.Errorf("%s missing %s", edge, field)
					}
				}
			}
		})
	}
}

func hasProperty(value any, name string) bool {
	switch v := value.(type) {
	case map[string]any:
		if props, ok := v["properties"].(map[string]any); ok {
			if _, ok := props[name]; ok {
				return true
			}
		}
		for _, child := range v {
			if hasProperty(child, name) {
				return true
			}
		}
	case []any:
		for _, child := range v {
			if hasProperty(child, name) {
				return true
			}
		}
	}
	return false
}

func readContract(t *testing.T, version, product string) map[string]any {
	t.Helper()
	path := filepath.Join("..", "..", "..", "specs", "meta", "instagram", version, product+".openapi.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err = json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	return doc
}

func TestInstagramContractsRemainResearchOnly(t *testing.T) {
	for _, v := range []string{"v24.0", "v25.0", "v26.0"} {
		for _, product := range []string{"facebook-login", "instagram-login", "webhooks"} {
			doc := readContract(t, v, product)
			if doc["x-contract-only"] != true {
				t.Errorf("%s/%s not contract-only", v, product)
			}
			evidence := doc["x-version-evidence"].(map[string]any)
			if evidence["latest-release-verified"] != false || evidence["collection-release-specific"] != false {
				t.Errorf("%s/%s overclaims collection version evidence", v, product)
			}
			for path, item := range doc["paths"].(map[string]any) {
				for method, value := range item.(map[string]any) {
					if method != "get" && method != "post" && method != "delete" {
						continue
					}
					op := value.(map[string]any)
					if op["x-simulator-implemented"] != false {
						t.Errorf("%s %s falsely implemented", method, path)
					}
					for _, key := range []string{"x-request-evidence", "x-response-evidence"} {
						if _, ok := op[key]; !ok {
							t.Errorf("%s %s missing %s", method, path, key)
						}
					}
				}
			}
		}
		fb := readContract(t, v, "facebook-login")["paths"].(map[string]any)
		ig := readContract(t, v, "instagram-login")["paths"].(map[string]any)
		if _, ok := fb["/"+v+"/{objectID}/authorized_adaccounts"]; !ok {
			t.Error("lost SDK adjacent edge inventory")
		}
		if _, ok := ig["/"+v+"/{objectID}/authorized_adaccounts"]; ok {
			t.Error("Facebook-only edge copied to Instagram Login")
		}
	}
}

func TestWelcomeFlowResponsesAllowDocumentedAlternatives(t *testing.T) {
	for _, version := range []string{"v24.0", "v25.0", "v26.0"} {
		path := filepath.Join("..", "..", "..", "specs", "meta", "instagram", version, "instagram-login.openapi.yaml")
		doc, err := openapi3.NewLoader().LoadFromFile(path)
		if err != nil {
			t.Fatal(err)
		}
		response := doc.Paths.Value("/" + version + "/{objectID}/welcome_message_flows").Post.Responses.Status(200)
		schema := response.Value.Content["application/json"].Schema.Value
		for _, body := range []any{map[string]any{"flow_id": "synthetic-flow"}, map[string]any{"success": true}} {
			if err := schema.VisitJSON(body); err != nil {
				t.Errorf("%s rejects documented response shape: %v", version, err)
			}
		}
	}
}
