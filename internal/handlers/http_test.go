package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/tomanagle/Social-Mockery/internal/handlers"
	"github.com/tomanagle/Social-Mockery/internal/profiles"
	"github.com/tomanagle/Social-Mockery/internal/simulation"
	"github.com/tomanagle/Social-Mockery/internal/store"
	"github.com/tomanagle/Social-Mockery/internal/store/db"
	"github.com/tomanagle/Social-Mockery/internal/store/dbstore"
)

const graph = "/https://graph.instagram.com/v25.0/"
const account = "17841400000000001"

func setup(t testing.TB, seed store.Seed, max int, admin string) http.Handler {
	t.Helper()
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	engine := simulation.New(dbstore.New(conn), seed, max)
	if err = engine.Reset(context.Background(), seed); err != nil {
		t.Fatal(err)
	}
	return handlers.New(engine, admin)
}
func request(h http.Handler, method, path, contentType, body, token string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	if contentType != "" {
		r.Header.Set("Content-Type", contentType)
	}
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}
func object(t testing.TB, w *httptest.ResponseRecorder, status int) map[string]any {
	t.Helper()
	if w.Code != status {
		t.Fatalf("status %d want %d: %s", w.Code, status, w.Body.String())
	}
	var v map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &v); err != nil {
		t.Fatalf("JSON: %v (%s)", err, w.Body.String())
	}
	return v
}
func create(t testing.TB, h http.Handler) string {
	t.Helper()
	return object(t, request(h, "POST", graph+account+"/media", "application/json", `{"image_url":"https://example.test/image.jpg","caption":"hello"}`, "mock-token"), 200)["id"].(string)
}
func advance(t testing.TB, h http.Handler, seconds int) {
	t.Helper()
	object(t, request(h, "POST", "/admin/clock/advance", "application/json", fmt.Sprintf(`{"seconds":%d}`, seconds), ""), 200)
}
func publish(h http.Handler, id string) *httptest.ResponseRecorder {
	return request(h, "POST", graph+account+"/media_publish", "application/x-www-form-urlencoded", url.Values{"creation_id": {id}}.Encode(), "mock-token")
}
func TestPublishingLifecycle(t *testing.T) {
	h := setup(t, profiles.Default(), 10000, "")
	id := create(t, h)
	status := func(want string) {
		t.Helper()
		v := object(t, request(h, "GET", graph+id+"?fields=status_code", "", "", "mock-token"), 200)
		if v["status_code"] != want {
			t.Fatalf("status=%v want %s", v, want)
		}
	}
	status("IN_PROGRESS")
	object(t, publish(h, id), 400)
	advance(t, h, 4)
	status("IN_PROGRESS")
	advance(t, h, 1)
	status("FINISHED")
	postID := object(t, publish(h, id), 200)["id"].(string)
	status("PUBLISHED")
	object(t, publish(h, id), 400)
	post := object(t, request(h, "GET", graph+postID+"?fields=permalink,media_url,caption,media_type", "", "", "mock-token"), 200)
	if post["media_url"] != "https://example.test/image.jpg" || post["caption"] != "hello" || post["media_type"] != "IMAGE" || !strings.Contains(post["permalink"].(string), postID) {
		t.Fatalf("post=%v", post)
	}
	advance(t, h, 86400)
	status("PUBLISHED")
}
func TestCreateEncodingsAndTokenLocations(t *testing.T) {
	for _, encoding := range []string{"json", "form", "multipart"} {
		for _, kind := range []string{"IMAGE", "REELS"} {
			t.Run(encoding+"/"+kind, func(t *testing.T) {
				h := setup(t, profiles.Default(), 100, "")
				values := url.Values{"caption": {"encoded + & caption"}}
				if kind == "IMAGE" {
					values.Set("image_url", "https://example.test/i.jpg")
				} else {
					values.Set("media_type", "REELS")
					values.Set("video_url", "https://example.test/v.mp4")
				}
				var body, ct string
				path := graph + account + "/media"
				token := ""
				switch encoding {
				case "json":
					obj := map[string]string{}
					for k, v := range values {
						obj[k] = v[0]
					}
					b, _ := json.Marshal(obj)
					body = string(b)
					ct = "application/json"
					path += "?access_token=mock-token"
				case "form":
					values.Set("access_token", "mock-token")
					body = values.Encode()
					ct = "application/x-www-form-urlencoded"
				case "multipart":
					var b bytes.Buffer
					w := multipart.NewWriter(&b)
					for k, v := range values {
						if err := w.WriteField(k, v[0]); err != nil {
							t.Fatal(err)
						}
					}
					w.Close()
					body = b.String()
					ct = w.FormDataContentType()
					token = "mock-token"
				}
				id := object(t, request(h, "POST", path, ct, body, token), 200)["id"].(string)
				advance(t, h, 5)
				postID := object(t, publish(h, id), 200)["id"].(string)
				post := object(t, request(h, "GET", graph+postID, "", "", "mock-token"), 200)
				if post["caption"] != "encoded + & caption" {
					t.Fatalf("caption=%v", post)
				}
				want := "IMAGE"
				if kind == "REELS" {
					want = "VIDEO"
					if post["media_product_type"] != "REELS" {
						t.Fatalf("post=%v", post)
					}
				}
				if post["media_type"] != want {
					t.Fatalf("post=%v", post)
				}
			})
		}
	}
}
func TestExpiryPermissionsOwnership(t *testing.T) {
	t.Run("container expiry", func(t *testing.T) {
		h := setup(t, profiles.Default(), 100, "")
		id := create(t, h)
		advance(t, h, 86400)
		v := object(t, request(h, "GET", graph+id, "", "", "mock-token"), 200)
		if v["status_code"] != "EXPIRED" {
			t.Fatal(v)
		}
		object(t, publish(h, id), 400)
	})
	for _, name := range []string{"expired token", "missing permission", "personal account"} {
		t.Run(name, func(t *testing.T) {
			s := profiles.Default()
			want := 403
			switch name {
			case "expired token":
				s.Tokens[0].ExpiresAt = 1767225600
				want = 400
			case "missing permission":
				s.Tokens[0].Scopes = []string{"instagram_business_basic"}
			case "personal account":
				s.Accounts[0].Type = "PERSONAL"
			}
			h := setup(t, s, 100, "")
			object(t, request(h, "POST", graph+account+"/media", "application/json", `{"image_url":"https://example.test/i.jpg"}`, "mock-token"), want)
		})
	}
	t.Run("other account", func(t *testing.T) {
		s := profiles.Default()
		s.Accounts = append(s.Accounts, store.Account{ID: "222", Username: "other", Type: "CREATOR"})
		s.Tokens = append(s.Tokens, store.Token{Value: "other-token", AccountID: "222", Scopes: s.Tokens[0].Scopes})
		h := setup(t, s, 100, "")
		id := create(t, h)
		object(t, request(h, "GET", graph+id, "", "", "other-token"), 400)
		advance(t, h, 5)
		object(t, request(h, "POST", graph+"222/media_publish", "application/json", fmt.Sprintf(`{"creation_id":%q}`, id), "other-token"), 400)
		p := object(t, publish(h, id), 200)["id"].(string)
		object(t, request(h, "GET", graph+p, "", "", "other-token"), 400)
	})
}
func TestRejectMalformedRequests(t *testing.T) {
	h := setup(t, profiles.Default(), 100, "")
	cases := []struct{ name, path, ct, body string }{{"invalid JSON", graph + account + "/media", "application/json", "{"}, {"non-string parameter", graph + account + "/media", "application/json", `{"image_url":123}`}, {"duplicate JSON", graph + account + "/media", "application/json", `{"image_url":"https://a.test/a","image_url":"https://b.test/b"}`}, {"duplicate query", graph + account + "/media?image_url=a&image_url=b", "application/json", "{}"}, {"duplicate form", graph + account + "/media", "application/x-www-form-urlencoded", "image_url=a&image_url=b"}, {"query/body collision", graph + account + "/media?image_url=a", "application/json", `{"image_url":"https://a.test/a"}`}, {"trailing JSON", graph + account + "/media", "application/json", `{} {}`}, {"unknown field", graph + account + "/media", "application/json", `{"image_url":"https://a.test/a","surprise":"yes"}`}, {"oversize body", graph + account + "/media", "application/json", `{"image_url":"https://a.test/a","caption":"` + strings.Repeat("x", 1<<20) + `"}`}}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := request(h, "POST", tc.path, tc.ct, tc.body, "mock-token")
			if w.Code < 400 || w.Code >= 500 {
				t.Fatalf("expected client error, got %d %s", w.Code, w.Body.String())
			}
		})
	}
}
func TestRoutesNeverForward(t *testing.T) {
	var calls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); io.WriteString(w, "unexpected") }))
	defer upstream.Close()
	h := setup(t, profiles.Default(), 100, "")
	for _, path := range []string{"/" + upstream.URL + "/v25.0/" + account, "/https://evil.test/v25.0/" + account, "/https://graph.instagram.com:443/v25.0/" + account, "/https://graph.instagram.com/v99.0/" + account, graph + account + "/unsupported", graph + "%31/media"} {
		w := request(h, "GET", path, "", "", "mock-token")
		if w.Code < 400 {
			t.Errorf("route accepted %s: %d", path, w.Code)
		}
	}
	object(t, request(h, "DELETE", graph+account, "", "", "mock-token"), 405)
	// Even an accepted media URL is fixture data; the simulator must not fetch it.
	object(t, request(h, "POST", graph+account+"/media", "application/json", fmt.Sprintf(`{"image_url":%q}`, upstream.URL+"/image.jpg"), "mock-token"), 200)
	if calls.Load() != 0 {
		t.Fatalf("outbound requests=%d", calls.Load())
	}
}
func TestAdminAndDeterministicReset(t *testing.T) {
	h := setup(t, profiles.Default(), 100, "admin-secret")
	object(t, request(h, "GET", "/admin/clock", "", "", ""), 401)
	clock := object(t, request(h, "GET", "/admin/clock", "", "", "admin-secret"), 200)
	if clock["now"] != "2026-01-01T00:00:00Z" {
		t.Fatal(clock)
	}
	id := create(t, h)
	object(t, request(h, "POST", "/admin/clock/advance", "application/json", `{"seconds":5}`, "admin-secret"), 200)
	object(t, request(h, "POST", "/admin/reset", "", "", "admin-secret"), 200)
	if got := create(t, h); got != id {
		t.Fatalf("reset id %s != %s", got, id)
	}
	for _, body := range []string{`{}`, `{"seconds":-1}`, `{"seconds":1.5}`, `{"seconds":1,"typo":1}`, `{"seconds":1} {}`} {
		object(t, request(h, "POST", "/admin/clock/advance", "application/json", body, "admin-secret"), 400)
	}
	r := httptest.NewRequest("POST", "/admin/reset", nil)
	r.Header.Set("Authorization", "Bearer admin-secret")
	r.Header.Set("Origin", "https://evil.test")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	object(t, w, 403)
}
func TestPagingAndConcurrentPublish(t *testing.T) {
	h := setup(t, profiles.Default(), 100, "")
	id := create(t, h)
	advance(t, h, 5)
	var wins atomic.Int32
	var wg sync.WaitGroup
	for range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := publish(h, id)
			if w.Code == 200 {
				wins.Add(1)
			} else if w.Code != 400 {
				t.Errorf("status=%d", w.Code)
			}
		}()
	}
	wg.Wait()
	if wins.Load() != 1 {
		t.Fatalf("winners=%d", wins.Load())
	}
	id = create(t, h)
	advance(t, h, 5)
	object(t, publish(h, id), 200)
	page := object(t, request(h, "GET", graph+account+"/media?limit=1", "", "", "mock-token"), 200)
	if len(page["data"].([]any)) != 1 {
		t.Fatal(page)
	}
	next := page["paging"].(map[string]any)["next"].(string)
	page = object(t, request(h, "GET", next, "", "", "mock-token"), 200)
	if len(page["data"].([]any)) != 1 || page["paging"] != nil {
		t.Fatal(page)
	}
}
func TestRecordLimit(t *testing.T) {
	h := setup(t, profiles.Default(), 1, "")
	create(t, h)
	object(t, request(h, "POST", graph+account+"/media", "application/json", `{"image_url":"https://example.test/i.jpg"}`, "mock-token"), 507)
}
func BenchmarkAccountRead(b *testing.B) {
	h := setup(b, profiles.Default(), 10000, "")
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		w := request(h, "GET", graph+account, "", "", "mock-token")
		if w.Code != 200 {
			b.Fatal(w.Body.String())
		}
	}
}
func BenchmarkCreateAndReset(b *testing.B) {
	h := setup(b, profiles.Default(), 10000, "")
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		create(b, h)
		object(b, request(h, "POST", "/admin/reset", "", "", ""), 200)
	}
}

func TestCustomResetAndFacebookScopeSelection(t *testing.T) {
	h := setup(t, profiles.Default(), 100, "")
	seed := profiles.Default()
	seed.Clock = "2027-02-03T04:05:06Z"
	seed.ProcessingSeconds = 0
	seed.Accounts[0].ID = "333"
	seed.Tokens[0].AccountID = "333"
	seed.Tokens[0].Value = "custom-token"
	seed.Tokens[0].Scopes = []string{"instagram_basic", "instagram_content_publish"}
	body, err := json.Marshal(seed)
	if err != nil {
		t.Fatal(err)
	}
	object(t, request(h, "POST", "/admin/reset", "application/json", string(body), ""), 200)
	clock := object(t, request(h, "GET", "/admin/clock", "", "", ""), 200)
	if clock["now"] != seed.Clock {
		t.Fatal(clock)
	}
	object(t, request(h, "POST", graph+"333/media", "application/json", `{"image_url":"https://example.test/image.jpg"}`, "custom-token"), 403)
	id := object(t, request(h, "POST", "/https://graph.facebook.com/v24.0/333/media", "application/json", `{"image_url":"https://example.test/image.jpg"}`, "custom-token"), 200)["id"].(string)
	object(t, request(h, "POST", "/https://graph.facebook.com/v24.0/333/media_publish", "application/json", fmt.Sprintf(`{"creation_id":%q}`, id), "custom-token"), 200)
	object(t, request(h, "GET", graph+account, "", "", "mock-token"), 400)
}
func TestAuthFailures(t *testing.T) {
	h := setup(t, profiles.Default(), 100, "")
	for _, token := range []string{"", "unknown"} {
		v := object(t, request(h, "GET", graph+account, "", "", token), 400)
		if v["error"].(map[string]any)["code"] != float64(190) {
			t.Fatal(v)
		}
	}
	object(t, request(h, "GET", graph+account+"?access_token=other", "", "", "mock-token"), 400)
}
func BenchmarkStartup(b *testing.B) {
	b.ReportAllocs()
	for range b.N {
		conn, err := db.Open(":memory:")
		if err != nil {
			b.Fatal(err)
		}
		e := simulation.New(dbstore.New(conn), profiles.Default(), 10000)
		if err = e.Reset(context.Background(), profiles.Default()); err != nil {
			b.Fatal(err)
		}
		_ = handlers.New(e, "")
		if err = conn.Close(); err != nil {
			b.Fatal(err)
		}
	}
}

func TestRejectMalformedQueryEncoding(t *testing.T) {
	h := setup(t, profiles.Default(), 100, "")
	for _, query := range []string{"caption=%zz", "caption=hello;extra=value"} {
		w := request(h, "POST", graph+account+"/media?"+query, "application/json", `{"image_url":"https://example.test/i.jpg"}`, "mock-token")
		if w.Code != 400 {
			t.Errorf("malformed query %q accepted: status %d %s", query, w.Code, w.Body.String())
		}
	}
}

func TestJSONNullIsNotAString(t *testing.T) {
	h := setup(t, profiles.Default(), 100, "")
	for _, field := range []string{"caption", "media_type"} {
		w := request(h, http.MethodPost, graph+account+"/media", "application/json",
			fmt.Sprintf(`{"image_url":"https://example.test/photo.jpg","%s":null}`, field), "mock-token")
		object(t, w, http.StatusBadRequest)
	}
}
