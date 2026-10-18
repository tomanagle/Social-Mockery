package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func tiktokContract(t *testing.T, product string) *openapi3.T {
	t.Helper()
	path := filepath.Join("..", "..", "..", "specs", "tiktok", "v2", product+".openapi.yaml")
	doc, err := openapi3.NewLoader().LoadFromFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.Validate(context.Background()); err != nil {
		t.Fatal(err)
	}
	return doc
}

// Fixtures are synthetic and offline. They test the local reference contract,
// not remote behavior, token validity, quotas, or a registered runtime handler.
func TestTikTokOAuthNativeEncodings(t *testing.T) {
	d := tiktokContract(t, "oauth")
	auth := d.Paths.Value("/v2/auth/authorize/").Get
	if auth.Servers == nil || (*auth.Servers)[0].URL != "https://www.tiktok.com" {
		t.Fatal("wrong authorization server")
	}
	token := d.Paths.Value("/v2/oauth/token/").Post
	if len(token.RequestBody.Value.Content) != 1 {
		t.Fatal("token endpoint must remain form-only")
	}
	s := token.RequestBody.Value.Content["application/x-www-form-urlencoded"].Schema.Value
	accepts(t, s, map[string]any{"client_key": "synthetic-client", "client_secret": "synthetic-secret", "grant_type": "authorization_code", "code": "synthetic-code", "redirect_uri": "https://example.invalid/callback"})
	accepts(t, s, map[string]any{"client_key": "synthetic-client", "client_secret": "synthetic-secret", "grant_type": "refresh_token", "refresh_token": "synthetic-refresh"})
	rejects(t, s, map[string]any{"client_key": "synthetic-client", "grant_type": "refresh_token"})
	rejects(t, s, map[string]any{"client_key": "synthetic-client", "client_secret": "synthetic-secret", "grant_type": "client_credentials"})
	if d.Paths.Value("/v2/oauth/refresh_token/") != nil {
		t.Fatal("invented separate refresh path")
	}
	if d.Paths.Value("/v2/oauth/revoke/") == nil || d.Paths.Value("/v2/oauth/get_qrcode/") == nil || d.Paths.Value("/v2/oauth/check_qrcode/") == nil {
		t.Fatal("missing OAuth operations")
	}
	for _, name := range []string{"UserTokenOutcome", "QRCodeResponseOutcome", "QRCodeStatusOutcome"} {
		accepts(t, d.Components.Schemas[name].Value, map[string]any{"error": "invalid_request", "error_description": "synthetic failure", "log_id": "synthetic-log"})
	}
	accepts(t, d.Components.Schemas["QRCodeResponseOutcome"].Value, map[string]any{"scan_qrcode_url": "aweme://authorize?client_ticket=tobefilled", "token": "synthetic-qr-token"})
	accepts(t, d.Components.Schemas["QRCodeStatusOutcome"].Value, map[string]any{"status": "confirmed", "redirect_uri": "https://example.invalid/callback?code=synthetic-code", "client_ticket": "synthetic-ticket"})
	if !d.Components.Schemas["OAuthError"].Value.Properties["error"].Value.Type.Is("string") {
		t.Fatal("OAuth error is a flat string")
	}
}

func TestTikTokDisplayProjectionAndPagination(t *testing.T) {
	d := tiktokContract(t, "display")
	for _, path := range []string{"/v2/user/info/", "/v2/video/list/", "/v2/video/query/"} {
		item := d.Paths.Value(path)
		var op *openapi3.Operation
		if item.Get != nil {
			op = item.Get
		} else {
			op = item.Post
		}
		found := false
		for _, p := range op.Parameters {
			if p.Value.Name == "fields" && p.Value.In == "query" && p.Value.Required {
				found = true
			}
		}
		if !found {
			t.Fatalf("%s missing fields query", path)
		}
	}
	user := d.Components.Schemas["User"].Value
	accepts(t, user, map[string]any{"display_name": "Synthetic account"})
	rejects(t, user, map[string]any{"display_name": nil})
	list := d.Components.Schemas["ListVideosRequest"].Value
	accepts(t, list, map[string]any{"cursor": float64(1700000000123), "max_count": float64(20)})
	rejects(t, list, map[string]any{"max_count": float64(21)})
	if !d.Components.Schemas["Video"].Value.Properties["id"].Value.Type.Is("string") {
		t.Fatal("video ids must be strings")
	}
}

func TestTikTokPostingVariants(t *testing.T) {
	d := tiktokContract(t, "posting")
	source := d.Components.Schemas["VideoSource"].Value
	accepts(t, source, map[string]any{"source": "PULL_FROM_URL", "video_url": "https://example.invalid/synthetic.mp4"})
	accepts(t, source, map[string]any{"source": "FILE_UPLOAD", "video_size": float64(100), "chunk_size": float64(100), "total_chunk_count": float64(1)})
	rejects(t, source, map[string]any{"source": "FILE_UPLOAD", "video_url": "https://example.invalid/synthetic.mp4"})
	rejects(t, source, map[string]any{"source": "PULL_FROM_URL"})
	photo := d.Components.Schemas["PhotoRequest"].Value
	photoSource := map[string]any{"source": "PULL_FROM_URL", "photo_images": []any{"https://example.invalid/a.jpg"}, "photo_cover_index": float64(0)}
	direct := map[string]any{"media_type": "PHOTO", "post_mode": "DIRECT_POST", "post_info": map[string]any{"privacy_level": "SELF_ONLY"}, "source_info": photoSource, "is_aigc": true}
	accepts(t, photo, direct)
	direct["post_info"] = map[string]any{}
	rejects(t, photo, direct)
	direct["post_mode"] = "MEDIA_UPLOAD"
	accepts(t, photo, direct)
	if d.Paths.Value("/v2/post/publish/creator_info/query/").Post.RequestBody != nil {
		t.Fatal("creator info must not invent a request body")
	}
	if d.Paths.Value("/v2/post/publish/cancel/") == nil {
		t.Fatal("missing documented download cancellation")
	}
	status := d.Components.Schemas["PublishStatusResponse"].Value.Properties["data"].Value
	if status.Properties["publicaly_available_post_id"] == nil {
		t.Fatal("documented status wire spelling changed")
	}
	if status.Properties["publicly_available_post_id"] != nil {
		t.Fatal("invented corrected field spelling")
	}
}

func TestTikTokUploadAndWebhookWireShapes(t *testing.T) {
	d := tiktokContract(t, "upload")
	if d.Extensions["x-dynamic-upload-url"] != true {
		t.Fatal("opaque upload URL metadata missing")
	}
	up := d.Paths.Value("/{uploadPath}").Put
	for _, ct := range []string{"video/mp4", "video/quicktime", "video/webm"} {
		if up.RequestBody.Value.Content[ct] == nil {
			t.Fatal("missing upload media encoding")
		}
	}
	for _, code := range []int{201, 206, 400, 403, 404, 416} {
		if up.Responses.Status(code) == nil {
			t.Fatalf("missing upload response %d", code)
		}
	}
	wh := tiktokContract(t, "webhooks")
	if wh.Paths.Value("/webhooks/tiktok").Get != nil {
		t.Fatal("invented Meta-style verification challenge")
	}
	s := wh.Components.Schemas["WebhookEnvelope"].Value
	accepts(t, s, map[string]any{"event": "authorization.removed", "content": "{\"reason\":1}"})
	rejects(t, s, map[string]any{"event": "authorization.removed", "content": map[string]any{"reason": float64(1)}})
}

func TestTikTokCoverageInventory(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	raw, err := os.ReadFile(filepath.Join(root, "specs", "tiktok", "coverage.json"))
	if err != nil {
		t.Fatal(err)
	}
	var c struct {
		Versions map[string]struct {
			Products map[string]struct {
				File           string `json:"file"`
				SHA256         string `json:"sha256"`
				OperationCount int    `json:"operation_count"`
			} `json:"products"`
		} `json:"versions"`
	}
	if err := json.Unmarshal(raw, &c); err != nil {
		t.Fatal(err)
	}
	total := 0
	for product, item := range c.Versions["v2"].Products {
		data, err := os.ReadFile(filepath.Join(root, item.File))
		if err != nil {
			t.Fatal(err)
		}
		h := sha256.Sum256(data)
		if hex.EncodeToString(h[:]) != item.SHA256 {
			t.Fatalf("stale coverage hash: %s", product)
		}
		d := tiktokContract(t, product)
		count := 0
		for _, p := range d.Paths.Map() {
			count += len(p.Operations())
		}
		if count != item.OperationCount {
			t.Fatalf("stale operation count: %s", product)
		}
		total += count
	}
	if total != 16 {
		t.Fatalf("expected 16 bounded operations, got %d", total)
	}
}
