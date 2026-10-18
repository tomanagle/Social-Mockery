package handlers_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/tomanagle/Social-Mockery/internal/handlers"
	"github.com/tomanagle/Social-Mockery/internal/profiles"
	"github.com/tomanagle/Social-Mockery/internal/simulation"
	"github.com/tomanagle/Social-Mockery/internal/store/db"
	"github.com/tomanagle/Social-Mockery/internal/store/dbstore"
)

// These tests exercise the exported HTTP router and real SQLite. No repository
// implementation is substituted, and provider requests never leave the process.
func TestSQLiteAPILifecycleMatrix(t *testing.T) {
	tests := []struct{ name, host, version, kind, encoding string }{}
	for _, host := range []string{"graph.instagram.com", "graph.facebook.com"} {
		for _, version := range []string{"v24.0", "v25.0"} {
			for _, kind := range []string{"IMAGE", "REELS"} {
				for _, encoding := range []string{"query", "form", "json"} {
					tests = append(tests, struct{ name, host, version, kind, encoding string }{host + "/" + version + "/" + kind + "/" + encoding, host, version, kind, encoding})
				}
			}
		}
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := setup(t, profiles.Default(), 100, "")
			base := "/https://" + tc.host + "/" + tc.version + "/"
			values := url.Values{"media_type": {tc.kind}, "caption": {"雪 + & café"}, "access_token": {"mock-token"}}
			mediaURL := "https://fixture.invalid/media?x=1&y=2"
			field, wantType, wantProduct := "image_url", "IMAGE", "FEED"
			if tc.kind == "REELS" {
				field, wantType, wantProduct = "video_url", "VIDEO", "REELS"
			}
			values.Set(field, mediaURL)
			encode := func(path string, values url.Values) (string, string, string) {
				switch tc.encoding {
				case "query":
					return path + "?" + values.Encode(), "", ""
				case "form":
					return path, "application/x-www-form-urlencoded", values.Encode()
				default:
					v := map[string]string{}
					for k := range values {
						v[k] = values.Get(k)
					}
					b, err := json.Marshal(v)
					if err != nil {
						t.Fatal(err)
					}
					return path, "application/json", string(b)
				}
			}
			path, ct, body := encode(base+account+"/media", values)
			id := object(t, request(h, "POST", path, ct, body, ""), 200)["id"].(string)
			for _, step := range []struct {
				advance int
				status  string
			}{{0, "IN_PROGRESS"}, {4, "IN_PROGRESS"}, {1, "FINISHED"}} {
				advance(t, h, step.advance)
				status := object(t, request(h, "GET", base+id+"?fields=status_code", "", "", "mock-token"), 200)
				if status["status_code"] != step.status || len(status) != 2 {
					t.Fatalf("container=%v want %s", status, step.status)
				}
			}
			path, ct, body = encode(base+account+"/media_publish", url.Values{"creation_id": {id}, "access_token": {"mock-token"}})
			postID := object(t, request(h, "POST", path, ct, body, ""), 200)["id"].(string)
			object(t, request(h, "POST", path, ct, body, ""), 400)
			post := object(t, request(h, "GET", base+postID, "", "", "mock-token"), 200)
			for key, want := range map[string]any{"id": postID, "caption": "雪 + & café", "media_type": wantType, "media_product_type": wantProduct, "media_url": mediaURL, "timestamp": "2026-01-01T00:00:05+0000"} {
				if post[key] != want {
					t.Errorf("%s=%v want %v", key, post[key], want)
				}
			}
			page := object(t, request(h, "GET", base+account+"/media?fields=caption,media_type", "", "", "mock-token"), 200)
			data := page["data"].([]any)
			if len(data) != 1 || !reflect.DeepEqual(data[0], map[string]any{"id": postID, "caption": "雪 + & café", "media_type": wantType}) {
				t.Fatalf("list=%v", page)
			}
			advance(t, h, 86400)
			status := object(t, request(h, "GET", base+id, "", "", "mock-token"), 200)
			if status["status_code"] != "PUBLISHED" {
				t.Fatal(status)
			}
		})
	}
}

func TestSQLiteAPIValidationTable(t *testing.T) {
	tests := []struct {
		name, method, path, ct, body string
		status, code                 int
	}{
		{"missing image", "POST", account + "/media", "application/json", `{}`, 400, 100},
		{"unsupported carousel", "POST", account + "/media", "application/json", `{"media_type":"CAROUSEL"}`, 400, 100},
		{"reel without video", "POST", account + "/media", "application/json", `{"media_type":"REELS"}`, 400, 100},
		{"image with video", "POST", account + "/media", "application/json", `{"image_url":"https://a.test/i","video_url":"https://a.test/v"}`, 400, 100},
		{"reel with image", "POST", account + "/media", "application/json", `{"media_type":"REELS","video_url":"https://a.test/v","image_url":"https://a.test/i"}`, 400, 100},
		{"relative URL", "POST", account + "/media", "application/json", `{"image_url":"/i.jpg"}`, 400, 100},
		{"URL credentials", "POST", account + "/media", "application/json", `{"image_url":"https://user:pass@a.test/i"}`, 400, 100},
		{"file URL", "POST", account + "/media", "application/json", `{"image_url":"file:///tmp/image"}`, 400, 100},
		{"caption overflow", "POST", account + "/media", "application/json", `{"image_url":"https://a.test/i","caption":"` + strings.Repeat("雪", 2201) + `"}`, 400, 100},
		{"URL overflow", "POST", account + "/media", "application/json", `{"image_url":"https://a.test/` + strings.Repeat("a", 8192) + `"}`, 400, 100},
		{"array JSON", "POST", account + "/media", "application/json", `[]`, 400, 100},
		{"null JSON", "POST", account + "/media", "application/json", `null`, 400, 100},
		{"nested JSON", "POST", account + "/media", "application/json", `{"image_url":{}}`, 400, 100},
		{"unsupported content type", "POST", account + "/media", "text/plain", `image_url=https://a.test/i`, 400, 100},
		{"missing content type", "POST", account + "/media", "", `image_url=https://a.test/i`, 400, 100},
		{"bad multipart boundary", "POST", account + "/media", "multipart/form-data; boundary=abc", `garbage`, 400, 100},
		{"missing creation ID", "POST", account + "/media_publish", "application/json", `{}`, 400, 100},
		{"unknown creation ID", "POST", account + "/media_publish", "application/json", `{"creation_id":"unknown"}`, 400, 100},
		{"unknown object", "GET", "unknown", "", "", 400, 100},
		{"unknown account field", "GET", account + "?fields=password", "", "", 400, 100},
		{"invalid empty-list field", "GET", account + "/media?fields=unknown", "", "", 400, 100},
		{"zero page limit", "GET", account + "/media?limit=0", "", "", 400, 100},
		{"negative page limit", "GET", account + "/media?limit=-1", "", "", 400, 100},
		{"large page limit", "GET", account + "/media?limit=101", "", "", 400, 100},
		{"noninteger page limit", "GET", account + "/media?limit=1.5", "", "", 400, 100},
		{"negative cursor", "GET", account + "/media?after=-1", "", "", 400, 100},
		{"large cursor", "GET", account + "/media?after=101", "", "", 400, 100},
		{"nonnumeric cursor", "GET", account + "/media?after=hello", "", "", 400, 100},
		{"unknown parameter", "GET", account + "?extra=1", "", "", 400, 100},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := setup(t, profiles.Default(), 100, "")
			w := request(h, tc.method, graph+tc.path, tc.ct, tc.body, "mock-token")
			v := object(t, w, tc.status)
			e, ok := v["error"].(map[string]any)
			message, messageOK := e["message"].(string)
			if !ok || e["code"] != float64(tc.code) || e["type"] != "IGApiException" || !messageOK || message == "" {
				t.Fatalf("error envelope=%v", v)
			}
			if w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("Content-Type") != "application/json" {
				t.Fatal(w.Header())
			}
			// Rejected operations must not consume IDs or poison the transaction.
			if id := create(t, h); id != "18000000000000001" {
				t.Fatalf("failed request mutated sequence: %s", id)
			}
		})
	}
}

func TestSQLiteAPIClockBoundaries(t *testing.T) {
	tests := []struct {
		name          string
		seconds       int
		want          string
		publishStatus int
	}{{"initial", 0, "IN_PROGRESS", 400}, {"before ready", 4, "IN_PROGRESS", 400}, {"at ready", 5, "FINISHED", 200}, {"before expiry", 86399, "FINISHED", 200}, {"at expiry", 86400, "EXPIRED", 400}, {"after expiry", 86401, "EXPIRED", 400}}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := setup(t, profiles.Default(), 100, "")
			id := create(t, h)
			advance(t, h, tc.seconds)
			v := object(t, request(h, "GET", graph+id, "", "", "mock-token"), 200)
			if v["status_code"] != tc.want {
				t.Fatal(v)
			}
			object(t, publish(h, id), tc.publishStatus)
		})
	}
}

func TestSQLiteAPIPersistenceAndIsolation(t *testing.T) {
	for _, tc := range []struct{ name, kind string }{{"image", "IMAGE"}, {"reel", "REELS"}} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "state.sqlite")
			open := func(initialize bool) (http.Handler, func()) {
				conn, err := db.Open(path)
				if err != nil {
					t.Fatal(err)
				}
				close := func() {
					if err := conn.Close(); err != nil {
						t.Error(err)
					}
				}
				t.Cleanup(func() { _ = conn.Close() })
				engine := simulation.New(dbstore.New(conn), profiles.Default(), 100)
				if initialize {
					if err := engine.Reset(context.Background(), profiles.Default()); err != nil {
						t.Fatal(err)
					}
				}
				return handlers.New(engine, ""), close
			}
			h, close := open(true)
			field := "image_url"
			if tc.kind == "REELS" {
				field = "video_url"
			}
			body := fmt.Sprintf(`{"media_type":%q,%q:"https://a.test/media","caption":"persisted"}`, tc.kind, field)
			id := object(t, request(h, "POST", graph+account+"/media", "application/json", body, "mock-token"), 200)["id"].(string)
			advance(t, h, 4)
			close()
			h, close = open(false)
			v := object(t, request(h, "GET", graph+id, "", "", "mock-token"), 200)
			if v["status_code"] != "IN_PROGRESS" {
				t.Fatal(v)
			}
			advance(t, h, 1)
			postID := object(t, publish(h, id), 200)["id"].(string)
			close()
			h, _ = open(false)
			post := object(t, request(h, "GET", graph+postID, "", "", "mock-token"), 200)
			if post["caption"] != "persisted" {
				t.Fatal(post)
			}
			object(t, publish(h, id), 400)
			list := object(t, request(h, "GET", graph+account+"/media", "", "", "mock-token"), 200)
			if len(list["data"].([]any)) != 1 {
				t.Fatal(list)
			}
			isolated := setup(t, profiles.Default(), 100, "")
			object(t, request(isolated, "GET", graph+postID, "", "", "mock-token"), 400)
			if create(t, isolated) != id {
				t.Fatal("fresh database did not start independent sequence")
			}
			object(t, request(h, "POST", "/admin/reset", "", "", ""), 200)
			object(t, request(h, "GET", graph+postID, "", "", "mock-token"), 400)
			if create(t, h) != id {
				t.Fatal("reset did not restore sequence")
			}
		})
	}
}

func TestSQLiteAPIAuthMatrix(t *testing.T) {
	tests := []struct {
		name, host   string
		scopes       []string
		expires      int64
		seconds      int
		status, code int
	}{
		{"instagram basic read only", "graph.instagram.com", []string{"instagram_business_basic"}, 0, 0, 403, 200},
		{"facebook basic read only", "graph.facebook.com", []string{"instagram_basic"}, 0, 0, 403, 200},
		{"instagram wrong scope family", "graph.instagram.com", []string{"instagram_basic", "instagram_content_publish"}, 0, 0, 403, 200},
		{"facebook wrong scope family", "graph.facebook.com", []string{"instagram_business_basic", "instagram_business_content_publish"}, 0, 0, 403, 200},
		{"missing basic scope", "graph.instagram.com", []string{"instagram_business_content_publish"}, 0, 0, 403, 200},
		{"before token expiry", "graph.instagram.com", []string{"instagram_business_basic", "instagram_business_content_publish"}, 1767225605, 4, 200, 0},
		{"at token expiry", "graph.instagram.com", []string{"instagram_business_basic", "instagram_business_content_publish"}, 1767225605, 5, 400, 190},
		{"after token expiry", "graph.instagram.com", []string{"instagram_business_basic", "instagram_business_content_publish"}, 1767225605, 6, 400, 190},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			seed := profiles.Default()
			seed.Tokens[0].Scopes = tc.scopes
			seed.Tokens[0].ExpiresAt = tc.expires
			h := setup(t, seed, 100, "")
			advance(t, h, tc.seconds)
			v := object(t, request(h, "POST", "/https://"+tc.host+"/v25.0/"+account+"/media", "application/json", `{"image_url":"https://a.test/i"}`, "mock-token"), tc.status)
			if tc.code != 0 && v["error"].(map[string]any)["code"] != float64(tc.code) {
				t.Fatal(v)
			}
		})
	}
}

func TestSQLiteAPIRouteRejections(t *testing.T) {
	tests := []struct {
		name, method, path string
		status             int
	}{
		{"http host", "GET", "/http://graph.instagram.com/v25.0/" + account, 400},
		{"unapproved host", "GET", "/https://evil.test/v25.0/" + account, 400},
		{"explicit port", "GET", "/https://graph.instagram.com:443/v25.0/" + account, 400},
		{"userinfo", "GET", "/https://user@graph.instagram.com/v25.0/" + account, 400},
		{"host suffix", "GET", "/https://graph.instagram.com.evil.test/v25.0/" + account, 400},
		{"unsupported version", "GET", "/https://graph.instagram.com/v99.0/" + account, 404},
		{"encoded object", "GET", graph + "%31", 404},
		{"empty segment", "GET", graph + account + "/", 404},
		{"nested path", "GET", graph + account + "/media/extra", 404},
		{"unsupported edge", "GET", graph + account + "/comments", 404},
		{"post object", "POST", graph + account, 404},
		{"get publish", "GET", graph + account + "/media_publish", 404},
		{"delete object", "DELETE", graph + account, 405},
		{"head object", "HEAD", graph + account, 405},
		{"URI bound", "GET", graph + account + "?fields=" + strings.Repeat("x", 16384), 414},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := setup(t, profiles.Default(), 100, "")
			object(t, request(h, tc.method, tc.path, "", "", "mock-token"), tc.status)
		})
	}
}

func TestSQLiteAPIAdminValidationPreservesState(t *testing.T) {
	tests := []struct{ name, path, body string }{
		{"negative seconds", "/admin/clock/advance", `{"seconds":-1}`},
		{"seconds overflow", "/admin/clock/advance", `{"seconds":31536001}`},
		{"integer overflow", "/admin/clock/advance", `{"seconds":9223372036854775808}`},
		{"null seconds", "/admin/clock/advance", `{"seconds":null}`},
		{"fractional seconds", "/admin/clock/advance", `{"seconds":0.5}`},
		{"unknown property", "/admin/clock/advance", `{"seconds":1,"other":1}`},
		{"trailing JSON", "/admin/clock/advance", `{"seconds":1} {}`},
		{"oversize clock body", "/admin/clock/advance", `{"seconds":1,"other":"` + strings.Repeat("x", 1024) + `"}`},
		{"invalid reseed", "/admin/reset", `{"clock":"invalid"}`},
		{"malformed reseed", "/admin/reset", `{`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := setup(t, profiles.Default(), 100, "")
			id := create(t, h)
			before := object(t, request(h, "GET", "/admin/clock", "", "", ""), 200)
			object(t, request(h, "POST", tc.path, "application/json", tc.body, ""), 400)
			after := object(t, request(h, "GET", "/admin/clock", "", "", ""), 200)
			if !reflect.DeepEqual(before, after) {
				t.Fatalf("failed admin request changed clock: %v -> %v", before, after)
			}
			object(t, request(h, "GET", graph+id, "", "", "mock-token"), 200)
			advance(t, h, 5)
			object(t, publish(h, id), 200)
		})
	}
}

func TestSQLiteAPICapacityAtomicity(t *testing.T) {
	for _, tc := range []struct {
		name        string
		max, status int
	}{{"publish capacity exhausted", 1, 507}, {"exact capacity", 2, 200}} {
		t.Run(tc.name, func(t *testing.T) {
			h := setup(t, profiles.Default(), tc.max, "")
			id := create(t, h)
			advance(t, h, 5)
			object(t, publish(h, id), tc.status)
			v := object(t, request(h, "GET", graph+id, "", "", "mock-token"), 200)
			want, count := "FINISHED", 0
			if tc.status == 200 {
				want, count = "PUBLISHED", 1
			}
			if v["status_code"] != want {
				t.Fatal(v)
			}
			list := object(t, request(h, "GET", graph+account+"/media", "", "", "mock-token"), 200)
			if len(list["data"].([]any)) != count {
				t.Fatal(list)
			}
			object(t, request(h, "POST", graph+account+"/media", "application/json", `{"image_url":"https://a.test/i"}`, "mock-token"), 507)
		})
	}
}

func TestSQLiteAPIPaginationTable(t *testing.T) {
	tests := []struct {
		name, query  string
		want, offset int
		more         bool
	}{
		{"default", "", 3, 0, false},
		{"first page", "?limit=1&fields=caption", 1, 0, true},
		{"middle page", "?limit=1&after=1&fields=caption", 1, 1, true},
		{"last page", "?limit=1&after=2&fields=caption", 1, 2, false},
		{"exact full page", "?limit=3", 3, 0, false},
		{"maximum limit", "?limit=100", 3, 0, false},
		{"end cursor", "?after=3", 0, 3, false},
		{"maximum cursor", "?after=100", 0, 100, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := setup(t, profiles.Default(), 100, "")
			ids := []string{}
			for range 3 {
				id := create(t, h)
				advance(t, h, 5)
				ids = append(ids, object(t, publish(h, id), 200)["id"].(string))
			}
			page := object(t, request(h, "GET", graph+account+"/media"+tc.query, "", "", "mock-token"), 200)
			data := page["data"].([]any)
			if len(data) != tc.want {
				t.Fatalf("page=%v", page)
			}
			for i, item := range data {
				if item.(map[string]any)["id"] != ids[tc.offset+i] {
					t.Fatalf("order=%v want %v", data, ids)
				}
			}
			if (page["paging"] != nil) != tc.more {
				t.Fatalf("paging=%v", page)
			}
			if tc.more {
				next := page["paging"].(map[string]any)["next"].(string)
				if !strings.Contains(next, "fields=caption") || strings.Contains(next, "mock-token") {
					t.Fatalf("next=%s", next)
				}
				nextPage := object(t, request(h, "GET", next, "", "", "mock-token"), 200)
				got := nextPage["data"].([]any)[0].(map[string]any)
				if got["id"] != ids[tc.offset+tc.want] || len(got) != 2 {
					t.Fatalf("next page=%v", nextPage)
				}
			}
		})
	}
}

func TestSQLiteAPICaptionBoundary(t *testing.T) {
	for _, tc := range []struct{ name, caption string }{{"empty", ""}, {"ASCII maximum", strings.Repeat("a", 2200)}, {"Unicode maximum", strings.Repeat("雪", 2200)}} {
		t.Run(tc.name, func(t *testing.T) {
			h := setup(t, profiles.Default(), 100, "")
			body, err := json.Marshal(map[string]string{"image_url": "https://a.test/i", "caption": tc.caption})
			if err != nil {
				t.Fatal(err)
			}
			id := object(t, request(h, "POST", graph+account+"/media", "application/json", string(body), "mock-token"), 200)["id"].(string)
			advance(t, h, 5)
			postID := object(t, publish(h, id), 200)["id"].(string)
			post := object(t, request(h, "GET", graph+postID+"?fields=caption", "", "", "mock-token"), 200)
			if post["caption"] != tc.caption {
				t.Fatalf("caption lost data")
			}
		})
	}
}
