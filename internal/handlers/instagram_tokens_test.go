package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/url"
	"regexp"
	"testing"

	"github.com/tomanagle/Social-Mockery/internal/store"
)

const igGraph = "/https://graph.instagram.com/"

func instagramLoginSeed(version string) store.Seed {
	s := oauthSeed("instagram")
	s.Apps[0].GraphAPIVersion = version
	return s
}

func TestInstagramLongLivedLifecycleSQLite(t *testing.T) {
	s := instagramLoginSeed("v25.0")
	h := setup(t, s, 100, "")
	short := issueOAuthToken(t, h, s)

	q := url.Values{"grant_type": {"ig_exchange_token"}, "client_secret": {"mock-client-secret"}, "access_token": {short}}
	long := object(t, request(h, "GET", igGraph+"access_token?"+q.Encode(), "", "", ""), 200)
	if long["token_type"] != "bearer" || long["expires_in"] != float64(5184000) || long["access_token"] == short {
		t.Fatalf("long-lived response: %v", long)
	}
	value := long["access_token"].(string)
	again := object(t, request(h, "GET", igGraph+"access_token?"+q.Encode(), "", "", ""), 200)
	if again["access_token"] != value {
		t.Fatalf("repeated exchange created a different token: %v", again)
	}

	// The short-lived token expires after an hour; the long-lived token keeps working.
	advance(t, h, 3601)
	object(t, request(h, "GET", graph+account+"?fields=id", "", "", short), 400)
	id := object(t, request(h, "POST", graph+account+"/media", "application/json", `{"image_url":"https://example.test/a.jpg"}`, value), 200)["id"].(string)
	advance(t, h, 5)
	object(t, request(h, "POST", graph+account+"/media_publish", "application/x-www-form-urlencoded", url.Values{"creation_id": {id}}.Encode(), value), 200)

	refresh := url.Values{"grant_type": {"ig_refresh_token"}, "access_token": {value}}
	early := object(t, request(h, "GET", igGraph+"refresh_access_token?"+refresh.Encode(), "", "", ""), 400)
	if early["error"].(map[string]any)["code"] != float64(100) {
		t.Fatalf("refresh before 24 hours: %v", early)
	}
	advance(t, h, 86400)
	refreshed := object(t, request(h, "GET", igGraph+"refresh_access_token?"+refresh.Encode(), "", "", ""), 200)
	if refreshed["access_token"] != value || refreshed["expires_in"] != float64(5184000) || refreshed["token_type"] != "bearer" {
		t.Fatalf("refresh response: %v", refreshed)
	}
	// Without the refresh the token would expire 60 days after the exchange.
	advance(t, h, 5184000-86400-3606+10)
	object(t, request(h, "GET", graph+account+"?fields=id", "", "", value), 200)
	advance(t, h, 5184000)
	object(t, request(h, "GET", graph+account+"?fields=id", "", "", value), 400)
	expired := object(t, request(h, "GET", igGraph+"refresh_access_token?"+refresh.Encode(), "", "", ""), 400)
	if expired["error"].(map[string]any)["code"] != float64(190) {
		t.Fatalf("expired long-lived token refreshed: %v", expired)
	}
}

func TestInstagramLongLivedRejectionsSQLite(t *testing.T) {
	cases := []struct {
		name   string
		path   func(short, long string) string
		method string
		setup  func(t *testing.T, h http.Handler)
		status int
		code   float64
	}{
		{"wrong client secret", func(short, _ string) string {
			return "access_token?" + url.Values{"grant_type": {"ig_exchange_token"}, "client_secret": {"wrong"}, "access_token": {short}}.Encode()
		}, "GET", nil, 400, 100},
		{"wrong grant type", func(short, _ string) string {
			return "access_token?" + url.Values{"grant_type": {"ig_refresh_token"}, "client_secret": {"mock-client-secret"}, "access_token": {short}}.Encode()
		}, "GET", nil, 400, 100},
		{"unsupported parameter", func(short, _ string) string {
			return "access_token?" + url.Values{"grant_type": {"ig_exchange_token"}, "client_secret": {"mock-client-secret"}, "access_token": {short}, "debug": {"1"}}.Encode()
		}, "GET", nil, 400, 100},
		{"unknown token", func(string, string) string {
			return "access_token?" + url.Values{"grant_type": {"ig_exchange_token"}, "client_secret": {"mock-client-secret"}, "access_token": {"missing"}}.Encode()
		}, "GET", nil, 400, 190},
		{"legacy fixture token", func(string, string) string {
			return "access_token?" + url.Values{"grant_type": {"ig_exchange_token"}, "client_secret": {"mock-client-secret"}, "access_token": {"mock-token"}}.Encode()
		}, "GET", nil, 400, 190},
		{"expired short-lived token", func(short, _ string) string {
			return "access_token?" + url.Values{"grant_type": {"ig_exchange_token"}, "client_secret": {"mock-client-secret"}, "access_token": {short}}.Encode()
		}, "GET", func(t *testing.T, h http.Handler) { advance(t, h, 3600) }, 400, 190},
		{"exchange a long-lived token", func(_, long string) string {
			return "access_token?" + url.Values{"grant_type": {"ig_exchange_token"}, "client_secret": {"mock-client-secret"}, "access_token": {long}}.Encode()
		}, "GET", nil, 400, 100},
		{"refresh a short-lived token", func(short, _ string) string {
			return "refresh_access_token?" + url.Values{"grant_type": {"ig_refresh_token"}, "access_token": {short}}.Encode()
		}, "GET", func(t *testing.T, h http.Handler) { advance(t, h, 3000) }, 400, 100},
		{"refresh with client secret", func(_, long string) string {
			return "refresh_access_token?" + url.Values{"grant_type": {"ig_refresh_token"}, "access_token": {long}, "client_secret": {"mock-client-secret"}}.Encode()
		}, "GET", nil, 400, 100},
		{"app approval revoked", func(_, long string) string {
			return "refresh_access_token?" + url.Values{"grant_type": {"ig_refresh_token"}, "access_token": {long}}.Encode()
		}, "GET", func(t *testing.T, h http.Handler) {
			advance(t, h, 86400)
			s := instagramLoginSeed("v25.0")
			s.Apps[0].ApprovedScopes = []string{"instagram_business_content_publish"}
			s.Tokens = nil
			object(t, request(h, "POST", "/admin/oauth/fixtures", "application/json", fixtureJSON(t, s), ""), 200)
		}, 400, 190},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := instagramLoginSeed("v25.0")
			h := setup(t, s, 100, "")
			short := issueOAuthToken(t, h, s)
			q := url.Values{"grant_type": {"ig_exchange_token"}, "client_secret": {"mock-client-secret"}, "access_token": {short}}
			long := object(t, request(h, "GET", igGraph+"access_token?"+q.Encode(), "", "", ""), 200)["access_token"].(string)
			if tc.setup != nil {
				tc.setup(t, h)
			}
			body := object(t, request(h, tc.method, igGraph+tc.path(short, long), "", "", ""), tc.status)
			e := body["error"].(map[string]any)
			if e["code"] != tc.code || e["type"] != "OAuthException" {
				t.Fatalf("error = %v, want code %v", e, tc.code)
			}
		})
	}
	t.Run("POST is not routed", func(t *testing.T) {
		s := instagramLoginSeed("v25.0")
		h := setup(t, s, 100, "")
		w := request(h, "POST", igGraph+"access_token", "application/x-www-form-urlencoded", "grant_type=ig_exchange_token", "")
		if w.Code != 405 {
			t.Fatalf("POST status %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestInstagramMeSQLite(t *testing.T) {
	cases := []struct {
		name    string
		version string
		path    string
		token   string // "oauth" uses an Instagram Login token
		status  int
		want    map[string]any
		absent  []string
	}{
		{"versioned default fields", "", graph + "me", "oauth", 200, map[string]any{}, []string{"username", "user_id"}},
		{"posty connect fields", "", graph + "me?fields=id,user_id,username,name,profile_picture_url,account_type", "oauth", 200,
			map[string]any{"user_id": account, "username": "mock_creator", "account_type": "Business"}, []string{"name", "profile_picture_url"}},
		{"media count", "", graph + "me?fields=media_count", "oauth", 200, map[string]any{"media_count": float64(0)}, nil},
		{"unsupported field", "", graph + "me?fields=email", "oauth", 400, nil, nil},
		{"v24.0 legacy fixture token", "", "/https://graph.instagram.com/v24.0/me?fields=user_id", "mock-token", 200, map[string]any{"user_id": account}, nil},
		{"facebook host", "", "/https://graph.facebook.com/v25.0/me", "mock-token", 404, nil, nil},
		{"instagram login token on facebook host", "", "/https://graph.facebook.com/v25.0/me", "oauth", 400, nil, nil},
		{"unversioned uses app version", "v25.0", igGraph + "me?fields=user_id,account_type", "oauth", 200, map[string]any{"user_id": account, "account_type": "Business"}, nil},
		{"unversioned v24.0 app", "v24.0", igGraph + "me?fields=user_id", "oauth", 200, map[string]any{"user_id": account}, nil},
		{"unversioned without app version", "", igGraph + "me", "oauth", 400, nil, nil},
		{"unversioned unsupported app version", "v26.0", igGraph + "me", "oauth", 404, nil, nil},
		{"unversioned legacy fixture token", "v25.0", igGraph + "me", "mock-token", 400, nil, nil},
		{"unversioned unknown token", "v25.0", igGraph + "me", "missing", 400, nil, nil},
		{"unversioned media read", "v25.0", igGraph + account + "?fields=username", "oauth", 200, map[string]any{"username": "mock_creator"}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := instagramLoginSeed(tc.version)
			h := setup(t, s, 100, "")
			token := tc.token
			if token == "oauth" {
				token = issueOAuthToken(t, h, s)
			}
			body := object(t, request(h, "GET", tc.path, "", "", token), tc.status)
			for k, v := range tc.want {
				if body[k] != v {
					t.Fatalf("%s = %v, want %v (%v)", k, body[k], v, body)
				}
			}
			for _, k := range tc.absent {
				if _, ok := body[k]; ok {
					t.Fatalf("%s present without fixture data: %v", k, body)
				}
			}
			if tc.status == 200 && tc.want != nil && body["id"] == nil {
				t.Fatalf("id missing: %v", body)
			}
		})
	}
}

func TestInstagramMeScopedIDAndAccountTypesSQLite(t *testing.T) {
	s := instagramLoginSeed("v25.0")
	s.Accounts = append(s.Accounts, store.Account{ID: "17841400000000009", Username: "creator", Type: "CREATOR"}, store.Account{ID: "17841400000000010", Username: "personal", Type: "PERSONAL"})
	s.Tokens = append(s.Tokens,
		store.Token{Value: "creator-token", AccountID: "17841400000000009", Scopes: []string{"instagram_business_basic"}},
		store.Token{Value: "personal-token", AccountID: "17841400000000010", Scopes: []string{"instagram_business_basic"}})
	h := setup(t, s, 100, "")
	oauth := issueOAuthToken(t, h, s)
	first := object(t, request(h, "GET", graph+"me", "", "", oauth), 200)["id"]
	second := object(t, request(h, "GET", graph+"me", "", "", oauth), 200)["id"]
	legacy := object(t, request(h, "GET", graph+"me", "", "", "mock-token"), 200)["id"]
	if first != second || first == legacy || first == account {
		t.Fatalf("scoped IDs: oauth=%v again=%v legacy=%v", first, second, legacy)
	}
	creator := object(t, request(h, "GET", graph+"me?fields=account_type", "", "", "creator-token"), 200)
	if creator["account_type"] != "Media_Creator" {
		t.Fatalf("creator account type: %v", creator)
	}
	object(t, request(h, "GET", graph+"me?fields=account_type", "", "", "personal-token"), 403)
}

func fixtureJSON(t *testing.T, s store.Seed) string {
	t.Helper()
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// TestPostyInstagramConnectAndPublishSQLite replays the request sequence in Posty's
// Instagram connect callback (workspaces/workers/web/src/routes/api/instagram/callback.ts)
// and publisher, using its URLs and parameters with Social Mockery's prefix.
func TestPostyInstagramConnectAndPublishSQLite(t *testing.T) {
	s := instagramLoginSeed("v25.0")
	h := setup(t, s, 100, "")

	q := url.Values{"client_id": {"test-app"}, "enable_fb_login": {"0"}, "force_authentication": {"1"}, "redirect_uri": {callbackURI}, "response_type": {"code"}, "scope": {"instagram_business_basic,instagram_business_content_publish"}, "state": {"state + & 雪"}}
	w := request(h, "GET", "/https://www.instagram.com/oauth/authorize/?"+q.Encode(), "", "", "")
	if w.Code != 200 {
		t.Fatalf("trailing-slash authorize %d: %s", w.Code, w.Body.String())
	}
	session := regexp.MustCompile(`name="session" value="([^"]+)"`).FindStringSubmatch(w.Body.String())
	if len(session) != 2 || len(w.Result().Cookies()) != 1 {
		t.Fatal("consent session missing")
	}
	code := codeFrom(t, submitConsent(h, session[1], w.Result().Cookies()[0], url.Values{"account": {account}, "decision": {"approve"}, "scope": s.Apps[0].ApprovedScopes}))

	exchange := url.Values{"client_id": {"test-app"}, "client_secret": {"mock-client-secret"}, "code": {code}, "grant_type": {"authorization_code"}, "redirect_uri": {callbackURI}}
	short := object(t, request(h, "POST", "/https://api.instagram.com/oauth/access_token", "application/x-www-form-urlencoded", exchange.Encode(), ""), 200)
	if short["user_id"] != account || short["access_token"] == nil {
		t.Fatalf("code exchange: %v", short)
	}

	ll := url.Values{"grant_type": {"ig_exchange_token"}, "client_secret": {"mock-client-secret"}, "access_token": {short["access_token"].(string)}}
	long := object(t, request(h, "GET", igGraph+"access_token?"+ll.Encode(), "", "", ""), 200)
	token := long["access_token"].(string)

	me := url.Values{"fields": {"id,user_id,username,name,profile_picture_url,account_type"}, "access_token": {token}}
	profile := object(t, request(h, "GET", igGraph+"me?"+me.Encode(), "", "", ""), 200)
	if profile["user_id"] != account || profile["account_type"] != "Business" || profile["username"] != "mock_creator" {
		t.Fatalf("profile: %v", profile)
	}

	create := url.Values{"image_url": {"https://example.test/a.jpg"}, "caption": {"from posty"}, "access_token": {token}}
	container := object(t, request(h, "POST", graph+account+"/media", "application/x-www-form-urlencoded", create.Encode(), ""), 200)["id"].(string)
	status := object(t, request(h, "GET", graph+container+"?fields=id,status,status_code&access_token="+url.QueryEscape(token), "", "", ""), 200)
	if status["status_code"] != "IN_PROGRESS" {
		t.Fatalf("status before processing: %v", status)
	}
	advance(t, h, 5)
	published := object(t, request(h, "POST", graph+account+"/media_publish", "application/x-www-form-urlencoded", url.Values{"creation_id": {container}, "access_token": {token}}.Encode(), ""), 200)["id"].(string)
	permalink := object(t, request(h, "GET", graph+published+"?fields=permalink&access_token="+url.QueryEscape(token), "", "", ""), 200)
	if permalink["permalink"] == nil {
		t.Fatalf("permalink: %v", permalink)
	}

	// api.instagram.com has no trailing-slash evidence and stays unsupported.
	if w := request(h, "GET", "/https://api.instagram.com/oauth/authorize/?"+q.Encode(), "", "", ""); w.Code == 200 {
		t.Fatalf("api.instagram.com trailing slash accepted: %d", w.Code)
	}
}
