package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func facebookContract(t *testing.T, version, product string) *openapi3.T {
	t.Helper()
	p := filepath.Join("..", "..", "..", "specs", "meta", "facebook", version, product+".openapi.yaml")
	if err := validate(p); err != nil {
		t.Fatal(err)
	}
	d, err := openapi3.NewLoader().LoadFromFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// Synthetic fixtures test the authored contract, not live provider behavior.
func TestFacebookNativeContracts(t *testing.T) {
	for _, v := range []string{"v24.0", "v25.0", "v26.0"} {
		t.Run(v, func(t *testing.T) {
			d := facebookContract(t, v, "pages")
			p := "/" + v
			if d.Paths.Value(p+"/{pageID}/videos").Get != nil {
				t.Fatal("SDK-only video read should not override current endpoint reference")
			}
			for _, forbidden := range []string{"/me/feed", "/{groupID}/feed", "/{pageID}/messages", "/{adAccountID}/ads"} {
				if d.Paths.Value(p+forbidden) != nil {
					t.Fatalf("excluded API route included: %s", forbidden)
				}
			}
			accounts := d.Paths.Value(p + "/{userID}/accounts").Get
			if accounts.Extensions["x-token-type"] != "Facebook User" {
				t.Fatal("Page discovery must preserve User token semantics")
			}
			accepts(t, d.Components.Schemas["PageList"].Value, map[string]any{"data": []any{map[string]any{"id": "synthetic-page", "access_token": "synthetic-token", "tasks": []any{"CREATE_CONTENT"}, "instagram_business_account": map[string]any{"id": "synthetic-ig"}}}})
			rejects(t, d.Components.Schemas["PageList"].Value, map[string]any{"data": "wrong-envelope"})
			photo := d.Paths.Value(p + "/{pageID}/photos").Post.RequestBody.Value
			accepts(t, photo.Content["application/x-www-form-urlencoded"].Schema.Value, map[string]any{"url": "https://example.invalid/photo.jpg", "published": false})
			rejects(t, photo.Content["application/x-www-form-urlencoded"].Schema.Value, map[string]any{"caption": "missing URL"})
			if photo.Content["multipart/form-data"] == nil {
				t.Fatal("missing binary photo transfer")
			}
			accepts(t, photo.Content["application/json"].Schema.Value, map[string]any{"url": "https://example.invalid/photo.jpg"})
			video := d.Paths.Value(p + "/{pageID}/videos").Post
			if video.Servers == nil || (*video.Servers)[0].URL != "https://graph-video.facebook.com" {
				t.Fatal("lost native video host")
			}
			vf := video.RequestBody.Value.Content["application/x-www-form-urlencoded"].Schema.Value
			accepts(t, vf, map[string]any{"upload_phase": "start", "file_size": float64(4096)})
			accepts(t, vf, map[string]any{"upload_phase": "finish", "upload_session_id": "synthetic-session"})
			rejects(t, vf, map[string]any{"upload_phase": "start"})
			rejects(t, vf, map[string]any{"file_url": "https://example.invalid/video.mp4", "upload_phase": "start"})
			rejects(t, vf, map[string]any{"file_url": "https://example.invalid/video.mp4", "upload_phase": "start", "file_size": float64(100)})
			rejects(t, vf, map[string]any{"file_url": "https://example.invalid/video.mp4", "upload_phase": "nonsense"})
			reel := d.Paths.Value(p + "/{pageID}/video_reels").Post.RequestBody.Value.Content["application/json"].Schema.Value
			accepts(t, reel, map[string]any{"upload_phase": "start"})
			accepts(t, reel, map[string]any{"upload_phase": "finish", "video_id": "synthetic-video", "video_state": "PUBLISHED"})
			rejects(t, reel, map[string]any{"upload_phase": "complete", "video_id": "synthetic-video"})
			rejects(t, reel, map[string]any{"upload_phase": "finish", "video_state": "PUBLISHED"})
			sub := d.Paths.Value(p + "/{pageID}/subscribed_apps")
			if !strings.Contains(sub.Delete.Extensions["x-token-type"].(string), "App access token") {
				t.Fatal("own subscription deletion lost App-token alternative")
			}
			if !strings.Contains(sub.Post.Description, "cannot configure Instagram") {
				t.Fatal("Page subscription must not imply Instagram field setup")
			}
		})
	}
}

func TestFacebookOAuthUploadAndWebhooks(t *testing.T) {
	for _, v := range []string{"v24.0", "v25.0", "v26.0"} {
		o := facebookContract(t, v, "oauth")
		auth := o.Paths.Value("/" + v + "/dialog/oauth").Get
		if auth.Servers == nil || (*auth.Servers)[0].URL != "https://www.facebook.com" || auth.Responses.Status(302) == nil {
			t.Fatal("authorization host/redirect missing")
		}
		exchange := o.Paths.Value("/" + v + "/oauth/access_token").Get
		if exchange.Extensions["x-parameter-variants"] == nil {
			t.Fatal("code and long-lived exchange requirements missing")
		}
		accepts(t, o.Components.Schemas["Token"].Value, map[string]any{"access_token": "synthetic", "token_type": "bearer", "expires_in": float64(3600)})
		debug := o.Components.Schemas["DebugToken"].Value
		accepts(t, debug, map[string]any{"data": map[string]any{"granular_scopes": []any{map[string]any{"scope": "pages_show_list", "target_ids": []any{float64(123)}}}}})
		accepts(t, debug, map[string]any{"data": map[string]any{"granular_scopes": []any{map[string]any{"target_ids": nil}}}})
		revoke := o.Paths.Value("/" + v + "/{userID}/permissions/{permission}").Delete
		accepts(t, revoke.Responses.Status(200).Value.Content["application/json"].Schema.Value, true)
		rejects(t, revoke.Responses.Status(200).Value.Content["application/json"].Schema.Value, map[string]any{"success": true})
		u := facebookContract(t, v, "upload")
		up := u.Paths.Value("/video-upload/" + v + "/{videoID}").Post
		if u.Servers[0].URL != "https://rupload.facebook.com" || up.RequestBody.Value.Content["application/octet-stream"] == nil || up.Extensions["x-transfer-variants"] == nil {
			t.Fatal("native raw/hosted upload distinctions missing")
		}
		w := facebookContract(t, v, "webhooks")
		wh := w.Paths.Value("/webhooks/facebook")
		if wh.Get.Responses.Status(200).Value.Content["text/plain"] == nil || wh.Post.Extensions["x-signature"] == nil {
			t.Fatal("challenge or raw signature metadata missing")
		}
		s := w.Components.Schemas["PageWebhook"].Value
		accepts(t, s, map[string]any{"object": "page", "entry": []any{map[string]any{"id": "synthetic-page", "changes": []any{map[string]any{"field": "feed", "value": map[string]any{"item": "comment", "verb": "add", "comment_id": "synthetic-comment"}}}}}})
		rejects(t, s, map[string]any{"object": "instagram", "entry": []any{}})
		rejects(t, s, []any{})
	}
}

func TestFacebookCoverageInventory(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	b, err := os.ReadFile(filepath.Join(root, "specs", "meta", "facebook", "coverage.json"))
	if err != nil {
		t.Fatal(err)
	}
	var inv struct {
		Versions map[string]struct {
			Products map[string]struct {
				File           string `json:"file"`
				SHA256         string `json:"sha256"`
				OperationCount int    `json:"operation_count"`
			} `json:"products"`
		} `json:"versions"`
	}
	if err := json.Unmarshal(b, &inv); err != nil {
		t.Fatal(err)
	}
	if len(inv.Versions) != 3 {
		t.Fatal("missing independently inventoried versions")
	}
	for version, entry := range inv.Versions {
		if len(entry.Products) != 4 {
			t.Fatal("missing Facebook products")
		}
		for product, p := range entry.Products {
			b, err := os.ReadFile(filepath.Join(root, p.File))
			if err != nil {
				t.Fatal(err)
			}
			h := sha256.Sum256(b)
			if hex.EncodeToString(h[:]) != p.SHA256 {
				t.Fatalf("stale hash %s %s", version, product)
			}
			d := facebookContract(t, version, product)
			count := 0
			for _, path := range d.Paths.Map() {
				count += len(path.Operations())
			}
			if count != p.OperationCount {
				t.Fatalf("stale count %s %s", version, product)
			}
		}
	}
}
