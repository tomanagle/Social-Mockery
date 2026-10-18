package handlers_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/tomanagle/Social-Mockery/internal/profiles"
	"github.com/tomanagle/Social-Mockery/internal/store"
)

const callbackURI = "https://client.example.test/callback"

func oauthSeed(product string) store.Seed {
	s := profiles.Default()
	scopes := []string{"instagram_business_basic", "instagram_business_content_publish"}
	switch product {
	case "facebook":
		scopes = []string{"public_profile", "pages_manage_posts"}
	case "instagram_facebook":
		scopes = []string{"instagram_basic", "instagram_content_publish"}
	case "tiktok":
		scopes = []string{"user.info.basic", "video.publish"}
	}
	s.Apps = []store.OAuthApp{{ID: "test-app", Name: "Example & <App>", Secret: "mock-client-secret", Product: product, RedirectURIs: []string{callbackURI}, ApprovedScopes: scopes, AccountIDs: []string{account}}}
	if product == "facebook" || product == "instagram_facebook" {
		s.Accounts = append(s.Accounts, store.Account{ID: "17841400000000002", Username: "linked_professional", Type: "BUSINESS"})
		s.Assets = []store.OAuthAsset{{ID: "1001", Name: "Example Page", AccountID: account, InstagramAccountID: "17841400000000002", Tasks: []string{"CREATE_CONTENT"}}}
	}
	return s
}
func oauthPaths(product string) (string, string, string) {
	switch product {
	case "facebook", "instagram_facebook":
		return "/https://www.facebook.com/v25.0/dialog/oauth", "/https://graph.facebook.com/v25.0/oauth/access_token", "client_id"
	case "tiktok":
		return "/https://www.tiktok.com/v2/auth/authorize/", "/https://open.tiktokapis.com/v2/oauth/token/", "client_key"
	}
	return "/https://www.instagram.com/oauth/authorize", "/https://api.instagram.com/oauth/access_token", "client_id"
}
func startConsent(t *testing.T, h http.Handler, s store.Seed, extra url.Values) (string, *http.Cookie) {
	t.Helper()
	auth, _, key := oauthPaths(s.Apps[0].Product)
	q := url.Values{key: {"test-app"}, "redirect_uri": {s.Apps[0].RedirectURIs[0]}, "response_type": {"code"}, "scope": {strings.Join(s.Apps[0].ApprovedScopes, ",")}, "state": {"state + & 雪"}}
	for k, v := range extra {
		q[k] = v
	}
	w := request(h, "GET", auth+"?"+q.Encode(), "", "", "")
	if w.Code != 200 {
		t.Fatalf("authorize %d: %s %s", w.Code, w.Body.String(), w.Header().Get("Location"))
	}
	if strings.Contains(w.Body.String(), s.Apps[0].Secret) || !strings.Contains(w.Body.String(), "SIMULATION ONLY") || !strings.Contains(w.Body.String(), "Example &amp; &lt;App&gt;") {
		t.Fatal("consent secret/simulation/escaping failure")
	}
	found := regexp.MustCompile(`name="session" value="([^"]+)"`).FindStringSubmatch(w.Body.String())
	if len(found) != 2 {
		t.Fatal("session field missing")
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteLaxMode {
		t.Fatal("cookie protection missing")
	}
	return found[1], cookies[0]
}
func submitConsent(h http.Handler, id string, cookie *http.Cookie, v url.Values) *httptest.ResponseRecorder {
	v.Set("session", id)
	r := httptest.NewRequest("POST", "/oauth/consent", strings.NewReader(v.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if cookie != nil {
		r.AddCookie(cookie)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}
func redeem(t *testing.T, h http.Handler, s store.Seed, code string, extra url.Values) *httptest.ResponseRecorder {
	t.Helper()
	_, path, key := oauthPaths(s.Apps[0].Product)
	v := url.Values{key: {"test-app"}, "client_secret": {"mock-client-secret"}, "grant_type": {"authorization_code"}, "redirect_uri": {s.Apps[0].RedirectURIs[0]}, "code": {code}}
	for k, x := range extra {
		v[k] = x
	}
	if strings.HasPrefix(s.Apps[0].Product, "facebook") || s.Apps[0].Product == "instagram_facebook" {
		return request(h, "GET", path+"?"+v.Encode(), "", "", "")
	}
	return request(h, "POST", path, "application/x-www-form-urlencoded", v.Encode(), "")
}
func codeFrom(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	if w.Code != 302 {
		t.Fatalf("consent %d %s", w.Code, w.Body.String())
	}
	u, err := url.Parse(w.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	if u.Query().Get("state") != "state + & 雪" {
		t.Fatalf("state not preserved: %s", u)
	}
	code := u.Query().Get("code")
	if code == "" {
		t.Fatalf("no code: %s", u)
	}
	return code
}
func TestOAuthConsentLifecycleSQLite(t *testing.T) {
	for _, product := range []string{"instagram", "instagram_facebook", "facebook", "tiktok"} {
		t.Run(product, func(t *testing.T) {
			s := oauthSeed(product)
			h := setup(t, s, 100, "")
			id, cookie := startConsent(t, h, s, nil)
			selected := []string{s.Apps[0].ApprovedScopes[0]}
			v := url.Values{"account": {account}, "decision": {"approve"}, "scope": selected}
			if len(s.Assets) > 0 {
				v.Set("asset", s.Assets[0].ID)
			}
			code := codeFrom(t, submitConsent(h, id, cookie, v))
			if got := submitConsent(h, id, cookie, v); got.Code != 400 {
				t.Fatalf("consent replay %d", got.Code)
			}
			response := object(t, redeem(t, h, s, code, nil), 200)
			token, _ := response["access_token"].(string)
			if product == "instagram" {
				data := response["data"].([]any)[0].(map[string]any)
				token = data["access_token"].(string)
				if data["permissions"] != selected[0] {
					t.Fatal("granted scopes mismatch")
				}
			}
			if product == "tiktok" && response["scope"] != selected[0] {
				t.Fatal("granted scopes mismatch")
			}
			if token == "" {
				t.Fatal("no token")
			}
			object(t, redeem(t, h, s, code, nil), 400)
			if product == "instagram" || product == "instagram_facebook" {
				host := "graph.instagram.com"
				target := account
				if product == "instagram_facebook" {
					host = "graph.facebook.com"
					target = s.Assets[0].InstagramAccountID
				}
				base := "/https://" + host + "/v25.0/" + target
				object(t, request(h, "GET", base+"?fields=id", "", "", token), 200)
				object(t, request(h, "POST", base+"/media", "application/x-www-form-urlencoded", url.Values{"image_url": {"https://example.test/image.jpg"}}.Encode(), token), 403)
			}
		})
	}
}
func TestOAuthDenyExpiryAndTamperingSQLite(t *testing.T) {
	for _, name := range []string{"deny", "cookie", "scope", "account", "expired_session", "expired_code", "wrong_redirect", "wrong_secret", "duplicate"} {
		t.Run(name, func(t *testing.T) {
			s := oauthSeed("instagram")
			h := setup(t, s, 100, "")
			id, cookie := startConsent(t, h, s, nil)
			v := url.Values{"account": {account}, "decision": {"approve"}, "scope": s.Apps[0].ApprovedScopes}
			switch name {
			case "deny":
				v.Set("decision", "deny")
				w := submitConsent(h, id, cookie, v)
				if w.Code != 302 {
					t.Fatal(w.Code)
				}
				u, _ := url.Parse(w.Header().Get("Location"))
				if u.Query().Get("error") != "access_denied" || u.Query().Get("code") != "" || u.Query().Get("state") != "state + & 雪" {
					t.Fatal(u)
				}
				return
			case "cookie":
				if w := submitConsent(h, id, nil, v); w.Code != 403 {
					t.Fatal(w.Code)
				}
				return
			case "scope":
				v.Add("scope", "unapproved")
				if w := submitConsent(h, id, cookie, v); w.Code != 400 {
					t.Fatal(w.Code)
				}
				return
			case "account":
				v.Set("account", "999")
				if w := submitConsent(h, id, cookie, v); w.Code != 400 {
					t.Fatal(w.Code)
				}
				return
			case "expired_session":
				advance(t, h, 600)
				if w := submitConsent(h, id, cookie, v); w.Code != 400 {
					t.Fatal(w.Code)
				}
				return
			case "duplicate":
				v.Add("decision", "deny")
				if w := submitConsent(h, id, cookie, v); w.Code != 400 {
					t.Fatal(w.Code)
				}
				return
			}
			code := codeFrom(t, submitConsent(h, id, cookie, v))
			extra := url.Values{}
			switch name {
			case "expired_code":
				advance(t, h, 3600)
			case "wrong_redirect":
				extra.Set("redirect_uri", callbackURI+"/evil")
			case "wrong_secret":
				extra.Set("client_secret", "wrong")
			}
			object(t, redeem(t, h, s, code, extra), 400)
		})
	}
}
func TestOAuthAuthorizeValidationSQLite(t *testing.T) {
	for _, tc := range []struct {
		name, key, value string
		status           int
	}{{"redirect", "redirect_uri", "https://evil.test", 400}, {"credentials in redirect", "redirect_uri", "https://client.example.test@evil.test/callback", 400}, {"unknown app", "client_id", "missing", 400}, {"scope", "scope", "unapproved", 302}, {"response type", "response_type", "token", 302}, {"business config", "config_id", "business-system-user", 302}} {
		t.Run(tc.name, func(t *testing.T) {
			s := oauthSeed("instagram")
			h := setup(t, s, 100, "")
			path, _, key := oauthPaths("instagram")
			q := url.Values{key: {"test-app"}, "redirect_uri": {callbackURI}, "scope": {s.Apps[0].ApprovedScopes[0]}, "response_type": {"code"}}
			q.Set(tc.key, tc.value)
			w := request(h, "GET", path+"?"+q.Encode(), "", "", "")
			if w.Code != tc.status {
				t.Fatalf("%d %s", w.Code, w.Body.String())
			}
			if tc.status == 400 && w.Header().Get("Location") != "" {
				t.Fatal("untrusted redirect followed")
			}
		})
	}
}
func TestTikTokDesktopPKCESQLite(t *testing.T) {
	s := oauthSeed("tiktok")
	s.Apps[0].RequirePKCE = true
	s.Apps[0].RedirectURIs = []string{"http://127.0.0.1:8999/callback"}
	h := setup(t, s, 100, "")
	verifier := strings.Repeat("a", 64)
	hash := sha256.Sum256([]byte(verifier))
	id, cookie := startConsent(t, h, s, url.Values{"code_challenge": {hex.EncodeToString(hash[:])}, "code_challenge_method": {"S256"}})
	code := codeFrom(t, submitConsent(h, id, cookie, url.Values{"account": {account}, "decision": {"approve"}, "scope": s.Apps[0].ApprovedScopes}))
	object(t, redeem(t, h, s, code, url.Values{"code_verifier": {strings.Repeat("b", 64)}}), 400)
	object(t, redeem(t, h, s, code, url.Values{"code_verifier": {verifier}}), 200)
}
func TestOAuthFixtureUpsertPreservesPostsSQLite(t *testing.T) {
	s := oauthSeed("instagram")
	h := setup(t, s, 100, "")
	id := create(t, h)
	advance(t, h, 5)
	post := object(t, publish(h, id), 200)["id"].(string)
	body, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	object(t, request(h, "POST", "/admin/oauth/fixtures", "application/json", string(body), ""), 200)
	object(t, request(h, "GET", graph+post, "", "", "mock-token"), 200)
	startConsent(t, h, s, nil)
}

func TestTikTokRefreshAndEncodingSQLite(t *testing.T) {
	s := oauthSeed("tiktok")
	h := setup(t, s, 100, "")
	id, cookie := startConsent(t, h, s, nil)
	code := codeFrom(t, submitConsent(h, id, cookie, url.Values{"account": {account}, "decision": {"approve"}, "scope": s.Apps[0].ApprovedScopes}))
	body := object(t, redeem(t, h, s, code, nil), 200)
	refresh, ok := body["refresh_token"].(string)
	if !ok || refresh == "" || body["refresh_expires_in"] != float64(31536000) {
		t.Fatalf("refresh missing %v", body)
	}
	_, path, _ := oauthPaths("tiktok")
	refreshRequest := func(value string) *httptest.ResponseRecorder {
		return request(h, "POST", path, "application/x-www-form-urlencoded", url.Values{"client_key": {"test-app"}, "client_secret": {"mock-client-secret"}, "grant_type": {"refresh_token"}, "refresh_token": {value}}.Encode(), "")
	}
	advance(t, h, 86400)
	next := object(t, refreshRequest(refresh), 200)
	if next["refresh_token"] == refresh || next["scope"] != strings.Join(s.Apps[0].ApprovedScopes, ",") {
		t.Fatalf("rotation %v", next)
	}
	object(t, refreshRequest(refresh), 400)
	for i := 0; i < 12; i++ {
		advance(t, h, 2592000)
	} // 361 days including the initial day.
	advance(t, h, 345600) // original 365-day expiry, not a sliding lifetime.
	object(t, refreshRequest(next["refresh_token"].(string)), 400)

	// Use a fresh, otherwise-valid code so this cannot pass on invalid_client.
	id, cookie = startConsent(t, h, s, nil)
	code = codeFrom(t, submitConsent(h, id, cookie, url.Values{"account": {account}, "decision": {"approve"}, "scope": s.Apps[0].ApprovedScopes}))
	var multipartBody bytes.Buffer
	writer := multipart.NewWriter(&multipartBody)
	for key, value := range map[string]string{"client_key": "test-app", "client_secret": "mock-client-secret", "grant_type": "authorization_code", "code": code, "redirect_uri": callbackURI} {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	result := object(t, request(h, "POST", path, writer.FormDataContentType(), multipartBody.String(), ""), 400)
	if result["error"] != "invalid_request" {
		t.Fatalf("wrong encoding error %v", result)
	}
	object(t, redeem(t, h, s, code, nil), 200)

}
func TestOAuthAssetChangesInvalidateTokenSQLite(t *testing.T) {
	for _, change := range []string{"owner", "link", "tasks", "account_type"} {
		t.Run(change, func(t *testing.T) {
			s := oauthSeed("instagram_facebook")
			h := setup(t, s, 100, "")
			id, cookie := startConsent(t, h, s, nil)
			code := codeFrom(t, submitConsent(h, id, cookie, url.Values{"account": {account}, "decision": {"approve"}, "scope": s.Apps[0].ApprovedScopes, "asset": {"1001"}}))
			token := object(t, redeem(t, h, s, code, nil), 200)["access_token"].(string)
			target := s.Assets[0].InstagramAccountID
			switch change {
			case "owner":
				s.Assets[0].AccountID = target
			case "link":
				s.Assets[0].InstagramAccountID = account
			case "tasks":
				s.Assets[0].Tasks = []string{"ANALYZE"}
			case "account_type":
				s.Accounts[1].Type = "PERSONAL"
			}
			b, _ := json.Marshal(s)
			object(t, request(h, "POST", "/admin/oauth/fixtures", "application/json", string(b), ""), 200)
			object(t, request(h, "GET", "/https://graph.facebook.com/v25.0/"+target+"?fields=id", "", "", token), 400)
		})
	}
}
func TestOAuthCallbackCSPAndVersions(t *testing.T) {
	for _, version := range []string{"v24.0", "v25.0", "v26.0"} {
		t.Run(version, func(t *testing.T) {
			s := oauthSeed("facebook")
			h := setup(t, s, 100, "")
			q := url.Values{"client_id": {"test-app"}, "redirect_uri": {callbackURI}, "scope": {"public_profile"}, "response_type": {"code"}, "state": {"x"}}
			w := request(h, "GET", "/https://www.facebook.com/"+version+"/dialog/oauth?"+q.Encode(), "", "", "")
			if w.Code != 200 {
				t.Fatal(w.Code, w.Body.String())
			}
			csp := w.Header().Get("Content-Security-Policy")
			if !strings.Contains(csp, "form-action 'self' https://client.example.test;") {
				t.Fatal(csp)
			}
		})
	}
	s := oauthSeed("facebook")
	h := setup(t, s, 100, "")
	w := request(h, "GET", "/https://www.facebook.com/v99.0/dialog/oauth?client_id=test-app", "", "", "")
	if w.Code == 200 || w.Code == 302 {
		t.Fatal("unsupported version accepted")
	}
}

func TestOAuthRealHTTPCrossOriginCallback(t *testing.T) {
	var received url.Values
	callbackServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = r.URL.Query()
		_, _ = w.Write([]byte("callback accepted"))
	}))
	defer callbackServer.Close()
	s := oauthSeed("instagram")
	s.Apps[0].RedirectURIs = []string{callbackServer.URL + "/callback"}
	server := httptest.NewServer(setup(t, s, 100, ""))
	defer server.Close()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Jar: jar}
	auth, _, _ := oauthPaths("instagram")
	q := url.Values{"client_id": {"test-app"}, "response_type": {"code"}, "scope": {strings.Join(s.Apps[0].ApprovedScopes, ",")}, "redirect_uri": s.Apps[0].RedirectURIs, "state": {"cross-origin-state"}}
	response, err := client.Get(server.URL + auth + "?" + q.Encode())
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(response.Header.Get("Content-Security-Policy"), "form-action 'self' "+callbackServer.URL+";") {
		t.Fatal("callback missing from CSP")
	}
	match := regexp.MustCompile(`name="session" value="([^"]+)"`).FindSubmatch(body)
	if len(match) != 2 {
		t.Fatalf("no consent form %s", body)
	}
	response, err = client.PostForm(server.URL+"/oauth/consent", url.Values{"session": {string(match[1])}, "account": {account}, "scope": s.Apps[0].ApprovedScopes, "decision": {"approve"}})
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err = io.ReadAll(response.Body)
	if err != nil || string(body) != "callback accepted" || received.Get("code") == "" || received.Get("state") != "cross-origin-state" {
		t.Fatalf("callback body=%s query=%v error=%v", body, received, err)
	}
}
