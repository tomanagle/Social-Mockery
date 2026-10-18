package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

// These tests preserve source-backed authorization distinctions. They are
// offline contract checks, not evidence that Meta accepted a live request.
func facebookAccessExtension(t *testing.T, extensions map[string]any, name string) map[string]any {
	t.Helper()
	raw, ok := extensions[name]
	if !ok {
		t.Fatalf("missing %s", name)
	}
	data, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	return decoded
}

func facebookAccessStrings(t *testing.T, value any, want ...string) {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestInstagramFacebookLoginAccessBoundaries(t *testing.T) {
	for _, version := range []string{"v24.0", "v25.0", "v26.0"} {
		t.Run(version, func(t *testing.T) {
			d := contract(t, version, "facebook-login")
			access := facebookAccessExtension(t, d.Extensions, "x-login-access-requirements")
			if access["linked-facebook-page-required"] != true || access["professional-account-required"] != true {
				t.Fatal("Facebook Login must retain professional account and linked Page prerequisites")
			}
			if access["basic-permission"] != "instagram_basic" || access["other-login-basic-permission"] != "instagram_business_basic" {
				t.Fatal("login-specific scope families were conflated")
			}
			facebookAccessStrings(t, access["access-token-types"], "facebook-user", "facebook-page")
			tasks := access["page-task-permissions"].(map[string]any)
			facebookAccessStrings(t, tasks["PROFILE_PLUS_MESSAGING"], "instagram_basic", "instagram_manage_messages")
			facebookAccessStrings(t, tasks["PROFILE_PLUS_ANALYZE"], "instagram_basic", "instagram_manage_insights")
			dependencies := facebookAccessExtension(t, d.Extensions, "x-permission-dependency-evidence")
			if dependencies["source-conflict"] == "" || dependencies["applicability"] == "" {
				t.Fatal("permission-reference dependency differences must remain explicit")
			}
			pre := "/" + version
			pages := d.Paths.Value(pre + "/me/accounts").Get
			permissions := facebookAccessExtension(t, pages.Extensions, "x-permissions")
			facebookAccessStrings(t, permissions["required"], "pages_show_list")
			facebookAccessStrings(t, permissions["workflow"], "instagram_basic")
			for _, op := range []*openapi3.Operation{
				pages,
				d.Paths.Value(pre + "/{objectID}/media").Get,
				d.Paths.Value(pre + "/{objectID}/insights").Get,
			} {
				tokens := facebookAccessExtension(t, op.Extensions, "x-access-token-requirements")
				facebookAccessStrings(t, tokens["types"], "facebook-user")
			}
		})
	}
}

func TestInstagramFacebookPublishingAccessSourceConflicts(t *testing.T) {
	for _, version := range []string{"v24.0", "v25.0", "v26.0"} {
		t.Run(version, func(t *testing.T) {
			d := contract(t, version, "facebook-login")
			for _, edge := range []string{"media", "media_publish"} {
				op := d.Paths.Value("/" + version + "/{objectID}/" + edge).Post
				conflict := facebookAccessExtension(t, op.Extensions, "x-publishing-access-source-conflict")
				endpoint := conflict["endpoint-reference"].(map[string]any)
				guide := conflict["publishing-guide"].(map[string]any)
				facebookAccessStrings(t, endpoint["access-token-types"], "facebook-user")
				facebookAccessStrings(t, guide["access-token-types"], "facebook-page")
				endpointScopes := endpoint["business-manager-permissions"].(map[string]any)
				guideScopes := guide["business-manager-permissions"].(map[string]any)
				facebookAccessStrings(t, endpointScopes["anyOf"], "ads_management", "ads_read")
				facebookAccessStrings(t, guideScopes["required"], "ads_management", "ads_read")
				tasks := facebookAccessExtension(t, op.Extensions, "x-page-task-requirements")
				facebookAccessStrings(t, tasks["anyOf"], "MANAGE", "CREATE_CONTENT")
				if edge == "media_publish" {
					facebookAccessStrings(t, endpoint["permissions"], "instagram_basic", "instagram_content_publish")
				}
			}
			limit := d.Paths.Value("/" + version + "/{objectID}/content_publishing_limit").Get
			permissions := facebookAccessExtension(t, limit.Extensions, "x-permissions")
			facebookAccessStrings(t, permissions["required"], "instagram_basic", "instagram_content_publish", "pages_read_engagement")
		})
	}
}

func TestInstagramFacebookMessagingAccessBoundaries(t *testing.T) {
	for _, version := range []string{"v24.0", "v25.0", "v26.0"} {
		t.Run(version, func(t *testing.T) {
			d := contract(t, version, "facebook-login")
			conversations := d.Paths.Value("/" + version + "/{objectID}/conversations").Get
			found := false
			for _, p := range conversations.Parameters {
				if p.Value.Name == "platform" {
					found = true
					accepts(t, p.Value.Schema.Value, "instagram")
					accepts(t, p.Value.Schema.Value, "INSTAGRAM")
					if len(p.Value.Schema.Value.Enum) != 0 {
						t.Fatal("SDK enum must not become exhaustive server validation")
					}
				}
			}
			if !found {
				t.Fatal("missing platform selector")
			}
			path := d.Paths.Value("/" + version + "/{objectID}/subscribed_apps")
			for _, op := range []*openapi3.Operation{path.Get, path.Post} {
				tokens := facebookAccessExtension(t, op.Extensions, "x-access-token-requirements")
				facebookAccessStrings(t, tokens["types"], "facebook-page")
				permissions := facebookAccessExtension(t, op.Extensions, "x-permissions")
				facebookAccessStrings(t, permissions["required"], "pages_manage_metadata", "pages_show_list")
			}
			boundary := facebookAccessExtension(t, path.Post.Extensions, "x-instagram-webhook-boundary")
			if boundary["subscribed-fields-configures-instagram-webhooks"] != false {
				t.Fatal("Page fields must not imply Instagram topic subscription support")
			}
			tokens := facebookAccessExtension(t, path.Delete.Extensions, "x-access-token-requirements")
			facebookAccessStrings(t, tokens["types"], "facebook-app")
		})
	}
}

func TestInstagramFacebookOAuthConsentProducts(t *testing.T) {
	for _, version := range []string{"v24.0", "v25.0", "v26.0"} {
		t.Run(version, func(t *testing.T) {
			d := contract(t, version, "oauth")
			op := d.Paths.Value("/" + version + "/dialog/oauth").Get
			found := false
			for _, p := range op.Parameters {
				if p.Value.Name == "config_id" {
					found = true
					if p.Value.Required || p.Value.In != "query" {
						t.Fatal("business config_id must be an optional dialog query parameter")
					}
				}
			}
			if !found {
				t.Fatal("missing Facebook Login for Business config_id")
			}
			consent := facebookAccessExtension(t, op.Extensions, "x-consent-products")
			ordinary := consent["facebook-login"].(map[string]any)
			business := consent["facebook-login-for-business"].(map[string]any)
			if ordinary["partial-permission-decline"] != true || business["all-requested-permissions-required"] != true {
				t.Fatal("ordinary and business consent behavior must remain distinct")
			}
			if business["additional-supported-business-permission-required"] != true {
				t.Fatal("business login needs a supported business permission")
			}
		})
	}
}

func TestInstagramAccessAuditCoverageHashes(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	raw, err := os.ReadFile(filepath.Join(root, "specs", "meta", "instagram", "coverage.json"))
	if err != nil {
		t.Fatal(err)
	}
	var coverage struct {
		Versions map[string]struct {
			Products map[string]struct {
				File   string `json:"file"`
				SHA256 string `json:"sha256"`
			} `json:"products"`
		} `json:"versions"`
	}
	if err := json.Unmarshal(raw, &coverage); err != nil {
		t.Fatal(err)
	}
	for version, data := range coverage.Versions {
		for _, product := range []string{"facebook-login", "oauth"} {
			entry := data.Products[product]
			raw, err := os.ReadFile(filepath.Join(root, entry.File))
			if err != nil {
				t.Fatal(err)
			}
			hash := sha256.Sum256(raw)
			if hex.EncodeToString(hash[:]) != entry.SHA256 {
				t.Errorf("stale coverage hash for %s/%s", version, product)
			}
		}
	}
}
