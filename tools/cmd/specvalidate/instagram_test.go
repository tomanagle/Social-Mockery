package main

import (
	"path/filepath"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

// Synthetic fixtures verify documented contracts offline, not live Meta behavior.
func contract(t *testing.T, version, product string) *openapi3.T {
	t.Helper()
	path := filepath.Join("..", "..", "..", "specs", "meta", "instagram", version, product+".openapi.yaml")
	d, err := openapi3.NewLoader().LoadFromFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return d
}
func accepts(t *testing.T, s *openapi3.Schema, v any) {
	t.Helper()
	if err := s.VisitJSON(v); err != nil {
		t.Fatal(err)
	}
}
func rejects(t *testing.T, s *openapi3.Schema, v any) {
	t.Helper()
	if err := s.VisitJSON(v); err == nil {
		t.Fatalf("invalid fixture accepted: %#v", v)
	}
}

func TestInstagramNativeContracts(t *testing.T) {
	for _, version := range []string{"v24.0", "v25.0", "v26.0"} {
		t.Run(version, func(t *testing.T) {
			ig := contract(t, version, "instagram-login")
			fb := contract(t, version, "facebook-login")
			media := ig.Paths.Value("/" + version + "/{objectID}/media").Post
			for _, ct := range []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"} {
				if media.RequestBody.Value.Content[ct] == nil {
					t.Fatalf("missing %s publishing input", ct)
				}
			}
			image := map[string]any{"image_url": "https://example.invalid/image.jpg"}
			accepts(t, media.RequestBody.Value.Content["application/json"].Schema.Value, image)
			accepts(t, media.RequestBody.Value.Content["multipart/form-data"].Schema.Value, image)
			publish := ig.Paths.Value("/" + version + "/{objectID}/media_publish").Post.RequestBody.Value.Content["application/json"].Schema.Value
			accepts(t, publish, map[string]any{"creation_id": "synthetic-container"})
			rejects(t, publish, map[string]any{})
			resumable := map[string]any{"media_type": "REELS", "upload_type": "resumable"}
			accepts(t, fb.Paths.Value("/" + version + "/{objectID}/media").Post.RequestBody.Value.Content["application/json"].Schema.Value, resumable)
			rejects(t, media.RequestBody.Value.Content["application/json"].Schema.Value, resumable)
			form := fb.Paths.Value("/" + version + "/{objectID}/media").Post.RequestBody.Value.Content["application/x-www-form-urlencoded"].Schema.Value
			accepts(t, form, map[string]any{"media_type": "CAROUSEL", "children": "[\"child-a\",\"child-b\"]"})
			profile := ig.Components.Schemas["MessagingUserProfile"].Value
			accepts(t, profile, map[string]any{"id": "synthetic-user", "name": nil, "profile_pic": nil})
			send := ig.Components.Schemas["SendInstagramMessage"].Value
			accepts(t, send, map[string]any{"recipient": map[string]any{"id": "synthetic-recipient"}, "sender_action": "typing_on"})
			accepts(t, send, map[string]any{"recipient": map[string]any{"id": "synthetic-recipient"}, "message": map[string]any{"text": "hello"}})
			flow := ig.Paths.Value("/" + version + "/{objectID}/welcome_message_flows").Post.Responses.Status(200).Value.Content["application/json"].Schema.Value
			accepts(t, flow, map[string]any{"flow_id": "synthetic-flow"})
			accepts(t, flow, map[string]any{"success": true})
			if ig.Paths.Value("/"+version+"/ig_hashtag_search") != nil {
				t.Fatal("Facebook-only hashtag root copied to Instagram Login")
			}
			if fb.Paths.Value("/"+version+"/ig_hashtag_search") == nil {
				t.Fatal("missing hashtag root")
			}
			for _, edge := range []string{"creator_marketplace_creators", "creator_marketplace_brand_info", "pass_thread_control", "moderate_conversations"} {
				if fb.Paths.Value("/"+version+"/{objectID}/"+edge) == nil {
					t.Errorf("missing documented Facebook edge %s", edge)
				}
			}
		})
	}
}

func TestOAuthUploadWebhookContracts(t *testing.T) {
	for _, version := range []string{"v24.0", "v25.0", "v26.0"} {
		oauth := contract(t, version, "oauth")
		token := oauth.Paths.Value("/oauth/access_token").Post
		if token.Servers == nil || (*token.Servers)[0].URL != "https://api.instagram.com" {
			t.Fatal("wrong Instagram exchange host")
		}
		req := token.RequestBody.Value.Content["multipart/form-data"].Schema.Value
		accepts(t, req, map[string]any{"client_id": "synthetic-client", "client_secret": "synthetic-secret", "grant_type": "authorization_code", "redirect_uri": "https://example.invalid/callback", "code": "synthetic-code"})
		rejects(t, req, map[string]any{"client_id": "synthetic-client"})
		if oauth.Paths.Value("/"+version+"/debug_token") == nil || oauth.Paths.Value("/"+version+"/{objectID}/permissions/{permission_name}") == nil {
			t.Fatal("missing Facebook token/permission management")
		}
		upload := contract(t, version, "upload")
		if upload.Servers[0].URL != "https://rupload.facebook.com" {
			t.Fatal("wrong upload host")
		}
		op := upload.Paths.Value("/ig-api-upload/" + version + "/{containerID}").Post
		if op.RequestBody.Value.Content["application/octet-stream"] == nil {
			t.Fatal("missing raw binary upload")
		}
		webhook := contract(t, version, "webhooks").Paths.Value("/webhooks/instagram")
		if webhook.Get == nil || webhook.Post == nil {
			t.Fatal("missing callback verification/delivery")
		}
		found := false
		for _, p := range webhook.Post.Parameters {
			if p.Value.Name == "X-Hub-Signature-256" {
				found = true
			}
		}
		if !found {
			t.Fatal("missing signed delivery header")
		}
	}
}

func TestInstagramContractsStaySeparateFromRuntime(t *testing.T) {
	for _, v := range []string{"v24.0", "v25.0", "v26.0"} {
		for _, product := range []string{"facebook-login", "instagram-login", "oauth", "upload", "webhooks"} {
			d := contract(t, v, product)
			if d.Extensions["x-contract-only"] != true {
				t.Fatal("contract-only flag missing")
			}
			for path, item := range d.Paths.Map() {
				for method, op := range item.Operations() {
					if op.Extensions["x-simulator-implemented"] != false {
						t.Errorf("%s %s falsely implemented", method, path)
					}
					for _, key := range []string{"x-request-evidence", "x-response-evidence"} {
						if _, ok := op.Extensions[key]; !ok {
							t.Errorf("%s %s missing %s", method, path, key)
						}
					}
				}
			}
		}
	}
}

// These alternatives are independently documented request encodings. A query
// parameter must not accidentally become mandatory for its body alternative.
func TestInstagramRequestEncodingRegressions(t *testing.T) {
	for _, version := range []string{"v24.0", "v25.0", "v26.0"} {
		t.Run(version, func(t *testing.T) {
			ig := contract(t, version, "instagram-login")
			fb := contract(t, version, "facebook-login")
			for _, product := range []*openapi3.T{ig, fb} {
				reply := product.Paths.Value("/" + version + "/{objectID}/replies").Post
				accepts(t, reply.RequestBody.Value.Content["application/json"].Schema.Value, map[string]any{"message": "synthetic reply"})
				rejects(t, reply.RequestBody.Value.Content["application/json"].Schema.Value, map[string]any{})
				for _, p := range reply.Parameters {
					if p.Value.In == "query" && p.Value.Name == "message" && p.Value.Required {
						t.Fatal("JSON reply incorrectly requires query message")
					}
				}
				for _, edge := range []string{"media", "media_publish"} {
					op := product.Paths.Value("/" + version + "/{objectID}/" + edge).Post
					part := op.RequestBody.Value.Content["multipart/form-data"]
					if part == nil || part.Schema == nil {
						t.Fatalf("%s missing multipart schema", edge)
					}
					fixture := map[string]any{"creation_id": "synthetic-container"}
					if edge == "media" {
						fixture = map[string]any{"media_type": "CAROUSEL", "children": "[\"child-a\",\"child-b\"]"}
					}
					accepts(t, part.Schema.Value, fixture)
				}
			}
			flow := ig.Paths.Value("/" + version + "/{objectID}/welcome_message_flows").Post
			accepts(t, flow.RequestBody.Value.Content["application/json"].Schema.Value, map[string]any{
				"name": "updated flow",
				"welcome_message_flow": []any{map[string]any{"message": map[string]any{
					"text": "hello", "quick_replies": []any{},
				}}},
			})
			profile := ig.Paths.Value("/" + version + "/{objectID}/messenger_profile").Delete
			accepts(t, profile.RequestBody.Value.Content["application/json"].Schema.Value,
				map[string]any{"fields": []any{"persistent_menu"}})
			for _, p := range profile.Parameters {
				if p.Value.In == "query" && p.Value.Name == "fields" && p.Value.Required {
					t.Fatal("JSON profile deletion incorrectly requires query fields")
				}
			}
			creator := fb.Paths.Value("/" + version + "/{objectID}/creator_marketplace_creators").Get
			for _, name := range []string{"creator_countries", "creator_interests", "creator_states", "major_audience_countries", "major_audience_states", "major_audience_device_type", "similar_to_creators"} {
				found := false
				for _, p := range creator.Parameters {
					if p.Value.Name != name {
						continue
					}
					found = true
					if p.Value.Schema == nil || !p.Value.Schema.Value.Type.Is("string") {
						t.Fatalf("%s must carry a single JSON-encoded query value", name)
					}
					if _, ok := p.Value.Schema.Value.Extensions["x-json-schema"]; !ok {
						t.Fatalf("%s missing decoded list schema", name)
					}
					accepts(t, p.Value.Schema.Value, "[\"synthetic-value\"]")
				}
				if !found {
					t.Fatalf("missing filter %s", name)
				}
			}
			upload := contract(t, version, "upload")
			op := upload.Paths.Value("/ig-api-upload/" + version + "/{containerID}").Post
			if op.Security == nil || len(*op.Security) != 1 {
				t.Fatal("upload must explicitly require authentication")
			}
			for name := range (*op.Security)[0] {
				scheme := upload.Components.SecuritySchemes[name].Value
				if scheme.Type != "apiKey" || scheme.In != "header" || scheme.Name != "Authorization" {
					t.Fatal("upload must use its native OAuth Authorization header scheme")
				}
			}
			if len((*op.Security)[0]) == 0 {
				t.Fatal("upload permits anonymous authentication")
			}
			for _, p := range op.Parameters {
				if p.Value.In == "header" && p.Value.Name == "Authorization" {
					t.Fatal("OpenAPI ignores Authorization header parameters; use security scheme")
				}
			}
		})
	}
}

func TestInstagramEventsCollaborationAndLikes(t *testing.T) {
	for _, version := range []string{"v24.0", "v25.0", "v26.0"} {
		t.Run(version, func(t *testing.T) {
			fb := contract(t, version, "facebook-login")
			ig := contract(t, version, "instagram-login")
			for _, edge := range []string{"upcoming_events", "collaboration_invites", "collaborative_media", "collaborators", "likes"} {
				if fb.Paths.Value("/"+version+"/{objectID}/"+edge) == nil {
					t.Fatalf("missing Facebook edge %s", edge)
				}
				if ig.Paths.Value("/"+version+"/{objectID}/"+edge) != nil {
					t.Fatalf("Facebook-only edge %s copied to Instagram Login", edge)
				}
			}
			for _, method := range []string{"POST", "DELETE"} {
				op := fb.Paths.Value("/" + version + "/{objectID}/likes").GetOperation(method)
				s := op.RequestBody.Value.Content["application/json"].Schema.Value
				accepts(t, s, map[string]any{"media_id": "synthetic-media"})
				accepts(t, s, map[string]any{"comment_id": "synthetic-comment"})
				rejects(t, s, map[string]any{})
				rejects(t, s, map[string]any{"media_id": "synthetic-media", "comment_id": "synthetic-comment"})
			}
			update := fb.Paths.Value("/" + version + "/{objectID}").Post
			if update.RequestBody.Value.Required {
				t.Fatal("event update body must not be required for query-only comment moderation on shared node path")
			}
			create := fb.Paths.Value("/" + version + "/{objectID}/upcoming_events").Post
			for _, op := range []*openapi3.Operation{create, update} {
				s := op.RequestBody.Value.Content["multipart/form-data"].Schema.Value
				accepts(t, s, map[string]any{
					"title": "synthetic event", "start_time": "2026-11-01T19:00:00+0000",
					"access_token":          "synthetic-token",
					"notification_subtypes": "[\"BEFORE_EVENT_1DAY\",\"BEFORE_EVENT_15MIN\",\"EVENT_START\"]",
				})
			}
			// Verify the decoded conditional constraints survived 3.1-to-3.0 conversion.
			decoded := fb.Components.Schemas["InstagramCreateUpcomingEventRequest"].Value
			event := map[string]any{"title": "event", "start_time": "2026-11-01T19:00:00+0000", "access_token": "synthetic-token"}
			accepts(t, decoded, event)
			event["notification_subtypes"] = []any{"BEFORE_EVENT_1DAY", "BEFORE_EVENT_15MIN", "EVENT_START"}
			accepts(t, decoded, event)
			event["notification_target_time"] = "EVENT_END"
			rejects(t, decoded, event)
			event["notification_subtypes"] = []any{"BEFORE_EVENT_2DAY", "BEFORE_EVENT_1DAY", "BEFORE_EVENT_1HOUR"}
			accepts(t, decoded, event)
			event["end_time"] = "2026-11-01T20:00:00+0000"
			rejects(t, decoded, event)
		})
	}
}
