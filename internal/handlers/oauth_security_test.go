package handlers_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tomanagle/Social-Mockery/internal/handlers"
	"github.com/tomanagle/Social-Mockery/internal/profiles"
	"github.com/tomanagle/Social-Mockery/internal/simulation"
	"github.com/tomanagle/Social-Mockery/internal/store"
	"github.com/tomanagle/Social-Mockery/internal/store/db"
	"github.com/tomanagle/Social-Mockery/internal/store/dbstore"
)

func issueOAuthToken(t *testing.T, h http.Handler, s store.Seed) string {
	t.Helper()
	id, cookie := startConsent(t, h, s, nil)
	form := url.Values{"account": {account}, "decision": {"approve"}, "scope": s.Apps[0].ApprovedScopes}
	if len(s.Assets) > 0 {
		form.Set("asset", s.Assets[0].ID)
	}
	code := codeFrom(t, submitConsent(h, id, cookie, form))
	body := object(t, redeem(t, h, s, code, nil), 200)
	return body["access_token"].(string)
}
func TestOAuthFixtureAdminAuthenticationAndOriginSQLite(t *testing.T) {
	s := oauthSeed("instagram")
	h := setup(t, s, 100, "admin-secret")
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{"", "wrong"} {
		object(t, request(h, "POST", "/admin/oauth/fixtures", "application/json", string(b), token), 401)
	}
	for _, origin := range []string{"https://evil.test", "null", "http://localhost"} {
		r := httptest.NewRequest("POST", "/admin/oauth/fixtures", strings.NewReader(string(b)))
		r.Header.Set("Authorization", "Bearer admin-secret")
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Origin", origin)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		object(t, w, 403)
	}
	object(t, request(h, "POST", "/admin/oauth/fixtures", "application/json", string(b), "admin-secret"), 200)
}
func TestOAuthLiveGrantAndAppApprovalRevocationSQLite(t *testing.T) {
	for _, mode := range []string{"grant scope", "app scope", "asset"} {
		t.Run(mode, func(t *testing.T) {
			product := "instagram"
			if mode == "asset" {
				product = "instagram_facebook"
			}
			s := oauthSeed(product)
			h := setup(t, s, 100, "")
			token := issueOAuthToken(t, h, s)
			target := graph + account + "?fields=id"
			if mode == "asset" {
				target = "/https://graph.facebook.com/v25.0/" + s.Assets[0].InstagramAccountID + "?fields=id"
			}
			object(t, request(h, "GET", target, "", "", token), 200)
			switch mode {
			case "grant scope":
				s.Grants = []store.OAuthGrant{{AppID: s.Apps[0].ID, AccountID: account, Scopes: []string{s.Apps[0].ApprovedScopes[0]}}}
			case "app scope":
				s.Apps[0].ApprovedScopes = s.Apps[0].ApprovedScopes[:1]
			case "asset":
				s.Grants = []store.OAuthGrant{{AppID: s.Apps[0].ID, AccountID: account, Scopes: s.Apps[0].ApprovedScopes, AssetIDs: []string{}}}
			}
			b, err := json.Marshal(s)
			if err != nil {
				t.Fatal(err)
			}
			object(t, request(h, "POST", "/admin/oauth/fixtures", "application/json", string(b), ""), 200)
			rejected := object(t, request(h, "GET", target, "", "", token), 400)
			if rejected["error"].(map[string]any)["code"] != float64(190) {
				t.Fatalf("unexpected token error: %v", rejected)
			}
		})
	}
}
func TestOAuthCrossProductTokensRejectedSQLite(t *testing.T) {
	for _, product := range []string{"instagram", "instagram_facebook", "facebook", "tiktok"} {
		t.Run(product, func(t *testing.T) {
			s := oauthSeed(product)
			h := setup(t, s, 100, "")
			token := issueOAuthToken(t, h, s)
			target := graph + account + "?fields=id"
			if product == "instagram" {
				target = "/https://graph.facebook.com/v25.0/" + account + "?fields=id"
			}
			object(t, request(h, "GET", target, "", "", token), 400)
		})
	}
}
func TestOAuthHTTPPersistenceRestartAndResetSQLite(t *testing.T) {
	s := oauthSeed("instagram")
	path := filepath.Join(t.TempDir(), "state.db")
	conn, err := db.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	engine := simulation.New(dbstore.New(conn), s, 100)
	if err = engine.Reset(context.Background(), s); err != nil {
		t.Fatal(err)
	}
	h := handlers.New(engine, "")
	token := issueOAuthToken(t, h, s)
	pendingID, cookie := startConsent(t, h, s, nil)
	if err = conn.Close(); err != nil {
		t.Fatal(err)
	}
	conn, err = db.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	h = handlers.New(simulation.New(dbstore.New(conn), s, 100), "")
	server := httptest.NewServer(h)
	defer server.Close()
	client := server.Client()
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	send := func(method, path, body, token string, want int) *http.Response {
		t.Helper()
		r, err := http.NewRequest(method, server.URL+path, strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		if token != "" {
			r.Header.Set("Authorization", "Bearer "+token)
		}
		if body != "" {
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
		response, err := client.Do(r)
		if err != nil {
			t.Fatal(err)
		}
		if response.StatusCode != want {
			b, _ := io.ReadAll(response.Body)
			response.Body.Close()
			t.Fatalf("%s %s: %d %s", method, path, response.StatusCode, b)
		}
		return response
	}
	response := send("GET", graph+account+"?fields=id", "", token, 200)
	response.Body.Close()
	form := url.Values{"session": {pendingID}, "account": {account}, "decision": {"approve"}, "scope": s.Apps[0].ApprovedScopes}
	req, err := http.NewRequest("POST", server.URL+"/oauth/consent", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	response, err = client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 302 {
		t.Fatalf("persisted consent: %d", response.StatusCode)
	}
	codeURL, err := url.Parse(response.Header.Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	response = send("POST", "/https://api.instagram.com/oauth/access_token", url.Values{"client_id": {"test-app"}, "client_secret": {"mock-client-secret"}, "grant_type": {"authorization_code"}, "redirect_uri": {callbackURI}, "code": {codeURL.Query().Get("code")}}.Encode(), "", 200)
	response.Body.Close()
	response = send("POST", "/admin/reset", "", "", 200)
	response.Body.Close()
	response = send("GET", graph+account+"?fields=id", "", token, 400)
	response.Body.Close()
	response = send("GET", graph+account+"?fields=id", "", "mock-token", 200)
	response.Body.Close()
}
func TestOAuthDemoProfileLoads(t *testing.T) {
	s, err := profiles.Load("../../profiles/oauth-demo.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Apps) != 4 || len(s.Accounts) != 2 || len(s.Assets) != 1 || s.Tokens[0].Value != "mock-token" {
		t.Fatalf("incomplete demo fixture")
	}
}
