// Package oauth implements local, synthetic consent against seeded identities.
// It never authenticates a real person or contacts a provider.
package oauth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"html/template"
	"mime"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/tomanagle/Social-Mockery/internal/providers/meta"
	"github.com/tomanagle/Social-Mockery/internal/simulation"
	"github.com/tomanagle/Social-Mockery/internal/store"
)

type Provider struct {
	engine  *simulation.Engine
	routers map[string]http.Handler
}

func New(e *simulation.Engine) *Provider {
	p := &Provider{engine: e}
	p.routers = contractRouters(p)
	return p
}

// Match validates the full upstream host/path pair before any redirect or parsing.
func Match(r *http.Request) (product, operation string, ok bool) {
	u, err := url.Parse(strings.TrimPrefix(r.RequestURI, "/"))
	if err != nil || u.Scheme != "https" || u.User != nil || u.Fragment != "" || strings.Contains(u.EscapedPath(), "%") {
		return
	}
	switch {
	case (u.Host == "www.instagram.com" || u.Host == "api.instagram.com") && u.Path == "/oauth/authorize":
		return "instagram", "authorize", true
	// Live-verified (owner-reported Posty production use, 2026-10-19): www.instagram.com
	// accepts the trailing-slash form, although Meta's guide shows none.
	case u.Host == "www.instagram.com" && u.Path == "/oauth/authorize/":
		return "instagram", "authorize", true
	case u.Host == "api.instagram.com" && u.Path == "/oauth/access_token":
		return "instagram", "token", true
	case u.Host == "graph.instagram.com" && u.Path == "/access_token":
		return "instagram", "long_lived", true
	case u.Host == "graph.instagram.com" && u.Path == "/refresh_access_token":
		return "instagram", "refresh", true
	case u.Host == "www.tiktok.com" && u.Path == "/v2/auth/authorize/":
		return "tiktok", "authorize", true
	case u.Host == "open.tiktokapis.com" && u.Path == "/v2/oauth/token/":
		return "tiktok", "token", true
	}
	for _, v := range []string{"v24.0", "v25.0", "v26.0"} {
		if u.Host == "www.facebook.com" && u.Path == "/"+v+"/dialog/oauth" {
			return "facebook", "authorize", true
		}
		if u.Host == "graph.facebook.com" && u.Path == "/"+v+"/oauth/access_token" {
			return "facebook", "token", true
		}
	}
	return
}
func (p *Provider) serve(w http.ResponseWriter, r *http.Request) {
	securityHeaders(w)
	product, op, ok := Match(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	method := "GET"
	if op == "token" && product != "facebook" {
		method = "POST"
	}
	if r.Method != method {
		w.Header().Set("Allow", method)
		fail(w, product, 405, "invalid_request", "Unsupported method")
		return
	}
	u, _ := url.Parse(strings.TrimPrefix(r.RequestURI, "/"))
	if product == "tiktok" && op == "token" {
		typ, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || typ != "application/x-www-form-urlencoded" {
			fail(w, product, 400, "invalid_request", "TikTok token requests require application/x-www-form-urlencoded")
			return
		}
	}
	values, err := parameters(w, r, u.RawQuery)
	if err != nil {
		fail(w, product, 400, "invalid_request", "Malformed or duplicate parameters")
		return
	}
	p.engine.Mu.Lock()
	defer p.engine.Mu.Unlock()
	if op == "authorize" {
		p.authorize(w, r, product, values)
	} else {
		p.exchange(w, r, product, values)
	}
}
func parameters(w http.ResponseWriter, r *http.Request, query string) (url.Values, error) {
	v, err := url.ParseQuery(query)
	if err != nil {
		return nil, err
	}
	if r.Method == "POST" {
		r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
		typ, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil {
			return nil, err
		}
		switch typ {
		case "application/x-www-form-urlencoded":
			err = r.ParseForm()
		case "multipart/form-data":
			err = r.ParseMultipartForm(64 << 10)
		default:
			return nil, errors.New("form encoding required")
		}
		if r.MultipartForm != nil {
			defer r.MultipartForm.RemoveAll()
			if len(r.MultipartForm.File) > 0 {
				return nil, errors.New("files not accepted")
			}
		}
		if err != nil {
			return nil, err
		}
		for k, vals := range r.PostForm {
			if _, ok := v[k]; ok {
				return nil, errors.New("duplicate parameter")
			}
			v[k] = vals
		}
	}
	for _, vals := range v {
		if len(vals) != 1 {
			return nil, errors.New("duplicate parameter")
		}
	}
	return v, nil
}
func appKey(product string) string {
	if product == "tiktok" {
		return "client_key"
	}
	return "client_id"
}
func productMatches(app, route string) bool {
	return app == route || route == "facebook" && app == "instagram_facebook"
}
func (p *Provider) authorize(w http.ResponseWriter, r *http.Request, product string, v url.Values) {
	app, err := p.engine.Store.OAuthApp(r.Context(), v.Get(appKey(product)))
	if err != nil || !productMatches(app.Product, product) {
		fail(w, product, 400, "invalid_client", "Unknown simulated app")
		return
	}
	redirect := v.Get("redirect_uri")
	if !slices.Contains(app.RedirectURIs, redirect) || redirect == "" {
		fail(w, product, 400, "invalid_request", "redirect_uri must exactly match a registered URI")
		return
	}
	// Errors deliberately stay local until the client and redirect have been verified.
	reject := func(code, msg string) {
		callback(w, r, redirect, v.Get("state"), url.Values{"error": {code}, "error_description": {msg}})
	}
	if v.Get("response_type") != "code" && !(product == "facebook" && v.Get("response_type") == "") {
		reject("unsupported_response_type", "Only authorization code flow is simulated")
		return
	}
	if v.Has("config_id") {
		reject("invalid_request", "Facebook Login for Business configurations are not simulated; use a standard user-token app")
		return
	}
	scopeValue := v.Get("scope")
	if product == "facebook" && scopeValue == "" {
		scopeValue = "public_profile"
	}
	scopes, ok := scopeList(scopeValue)
	if !ok || len(scopes) == 0 {
		reject("invalid_scope", "A comma-separated scope list is required")
		return
	}
	for _, scope := range scopes {
		if !knownScope(app.Product, scope) || !slices.Contains(app.ApprovedScopes, scope) {
			reject("invalid_scope", "Requested permission is unsupported or not app-approved")
			return
		}
	}
	challenge := v.Get("code_challenge")
	if product == "tiktok" && !app.RequirePKCE && (challenge != "" || v.Has("code_challenge_method")) {
		reject("invalid_request", "Web PKCE is not simulated; use a desktop fixture with require_pkce")
		return
	}
	if challenge != "" || app.RequirePKCE {
		if v.Get("code_challenge_method") != "S256" || !validChallenge(product, challenge) {
			reject("invalid_request", "PKCE requires an S256 challenge")
			return
		}
	} else if v.Has("code_challenge_method") {
		reject("invalid_request", "Missing code_challenge")
		return
	}
	accounts, err := p.engine.Store.OAuthAccounts(r.Context(), app.ID)
	if err != nil {
		fail(w, product, 500, "server_error", "Storage failure")
		return
	}
	if len(accounts) == 0 {
		reject("access_denied", "No seeded test accounts are assigned to this app")
		return
	}
	now, _, err := p.engine.Store.Clock(r.Context())
	if err != nil {
		fail(w, product, 500, "server_error", "Storage failure")
		return
	}
	id, err := randomValue()
	if err != nil {
		fail(w, product, 500, "server_error", "Could not create consent session")
		return
	}
	s := store.OAuthSession{ID: id, AppID: app.ID, RedirectURI: redirect, State: v.Get("state"), Scopes: scopes, Challenge: challenge, ExpiresAt: now + 600}
	if err = p.engine.Store.PutOAuthSession(r.Context(), s); err != nil {
		fail(w, product, 503, "server_error", "Consent session capacity or storage failure")
		return
	}
	data := consentData{Session: id, App: app, Accounts: accounts, Scopes: scopes, Provider: providerName(app.Product), Redirect: redirect}
	for _, a := range accounts {
		assets, err := p.engine.Store.OAuthAssets(r.Context(), a.ID)
		if err != nil {
			fail(w, product, 500, "server_error", "Storage failure")
			return
		}
		data.Assets = append(data.Assets, assets...)
	}
	http.SetCookie(w, &http.Cookie{Name: "sm_consent_" + id, Value: id, Path: "/oauth/consent", HttpOnly: true, Secure: r.TLS != nil, SameSite: http.SameSiteLaxMode, MaxAge: 600})
	allowCallback(w, redirect)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// Parse errors are handled, not hidden behind template.Must runtime panics.
	page, err := template.New("consent").Funcs(template.FuncMap{"permissionDescription": permissionDescription}).Parse(consentHTML)
	if err != nil {
		fail(w, product, 500, "server_error", "Template failure")
		return
	}
	_ = page.Execute(w, data)
}

func (p *Provider) Consent(w http.ResponseWriter, r *http.Request) {
	securityHeaders(w)
	if r.Method != "POST" {
		w.Header().Set("Allow", "POST")
		http.Error(w, "POST required", 405)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	typ, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || typ != "application/x-www-form-urlencoded" || r.ParseForm() != nil || r.URL.RawQuery != "" {
		http.Error(w, "Invalid consent form", 400)
		return
	}
	v := r.PostForm
	for key, values := range v {
		if key != "scope" && key != "asset" && len(values) != 1 {
			http.Error(w, "Duplicate form field", 400)
			return
		}
		if !slices.Contains([]string{"session", "account", "decision", "scope", "asset"}, key) {
			http.Error(w, "Unknown form field", 400)
			return
		}
	}
	id := v.Get("session")
	cookie, err := r.Cookie("sm_consent_" + id)
	if err != nil || len(id) != 43 || subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(id)) != 1 {
		http.Error(w, "Consent session cookie is missing or invalid. Reopen authorization.", 403)
		return
	}
	if origin := r.Header.Get("Origin"); origin != "" {
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		if origin != scheme+"://"+r.Host {
			http.Error(w, "Cross-origin consent is disabled", 403)
			return
		}
	}
	p.engine.Mu.Lock()
	defer p.engine.Mu.Unlock()
	s, err := p.engine.Store.GetOAuthSession(r.Context(), id)
	if err != nil {
		http.Error(w, "Consent session expired or already used. Reopen authorization.", 400)
		return
	}
	app, err := p.engine.Store.OAuthApp(r.Context(), s.AppID)
	if err != nil || !slices.Contains(app.RedirectURIs, s.RedirectURI) {
		http.Error(w, "App registration changed. Reopen authorization.", 400)
		return
	}
	if v.Get("decision") == "deny" {
		if err = p.engine.Store.DeleteOAuthSession(r.Context(), id); err != nil {
			http.Error(w, "Consent session already used", 400)
			return
		}
		clearCookie(w, r, id)
		denied := url.Values{"error": {"access_denied"}, "error_description": {"Permissions error"}}
		if app.Product != "tiktok" {
			denied.Set("error_reason", "user_denied")
		}
		callback(w, r, s.RedirectURI, s.State, denied)
		return
	}
	if v.Get("decision") != "approve" {
		http.Error(w, "Choose approve or deny", 400)
		return
	}
	account := v.Get("account")
	if !slices.Contains(app.AccountIDs, account) {
		http.Error(w, "Account is not assigned to this app", 400)
		return
	}
	scopes := v["scope"]
	if (app.Product == "facebook" || app.Product == "instagram_facebook") && slices.Contains(s.Scopes, "public_profile") && !slices.Contains(scopes, "public_profile") {
		scopes = append(scopes, "public_profile")
	}
	if len(scopes) > 32 || duplicate(scopes) {
		http.Error(w, "Invalid permission selection", 400)
		return
	}
	for _, scope := range scopes {
		if !slices.Contains(s.Scopes, scope) || !slices.Contains(app.ApprovedScopes, scope) {
			http.Error(w, "Permission was not requested or is no longer app-approved", 400)
			return
		}
	}
	assets := v["asset"]
	if duplicate(assets) {
		http.Error(w, "Duplicate Page selection", 400)
		return
	}
	if _, err = p.validateAssets(r, app, account, scopes, assets); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	now, _, err := p.engine.Store.Clock(r.Context())
	if err != nil {
		http.Error(w, "Storage failure", 500)
		return
	}
	code, err := randomValue()
	if err != nil {
		http.Error(w, "Could not create authorization code", 500)
		return
	}
	ttl := int64(300)
	if app.Product == "instagram" {
		ttl = 3600
	}
	c := store.OAuthCode{Value: code, AppID: app.ID, AccountID: account, RedirectURI: s.RedirectURI, Scopes: scopes, AssetIDs: assets, Challenge: s.Challenge, ExpiresAt: now + ttl}
	if err = p.engine.Store.CompleteOAuthSession(r.Context(), id, c); err != nil {
		http.Error(w, "Consent session expired, used, or storage unavailable", 400)
		return
	}
	clearCookie(w, r, id)
	result := url.Values{"code": {code}}
	if app.Product == "tiktok" {
		result.Set("scopes", strings.Join(scopes, ","))
	}
	callback(w, r, s.RedirectURI, s.State, result)
}
func (p *Provider) validateAssets(r *http.Request, app store.OAuthApp, account string, scopes, selected []string) (string, error) {
	target := account
	if app.Product == "instagram" {
		a, err := p.engine.Store.Account(r.Context(), account)
		if err != nil || a.Type == "PERSONAL" {
			return "", errors.New("Instagram Login requires a seeded professional account")
		}
	}
	if len(selected) == 0 && app.Product == "instagram_facebook" {
		return "", errors.New("Select a Page linked to a professional Instagram account")
	}
	if len(selected) > 1 && app.Product == "instagram_facebook" {
		return "", errors.New("This simulator supports one linked Instagram Page per authorization")
	}
	if len(selected) > 0 && app.Product != "facebook" && app.Product != "instagram_facebook" {
		return "", errors.New("This login product does not select Facebook Pages")
	}
	assets, err := p.engine.Store.OAuthAssets(r.Context(), account)
	if err != nil {
		return "", errors.New("Could not read seeded Pages")
	}
	for _, id := range selected {
		idx := slices.IndexFunc(assets, func(a store.OAuthAsset) bool { return a.ID == id })
		if idx < 0 {
			return "", errors.New("Selected Page is not assigned to the chosen account")
		}
		a := assets[idx]
		if app.Product == "instagram_facebook" {
			linked, err := p.engine.Store.Account(r.Context(), a.InstagramAccountID)
			if err != nil || linked.Type == "PERSONAL" {
				return "", errors.New("Selected Page must link to a professional Instagram account")
			}
			target = linked.ID
		}
		for _, scope := range scopes {
			if !simulation.PageTaskAllows(a.Tasks, scope) {
				return "", errors.New("Selected Page tasks do not allow a selected permission")
			}
		}
	}
	return target, nil
}
func (p *Provider) exchange(w http.ResponseWriter, r *http.Request, product string, v url.Values) {
	app, err := p.engine.Store.OAuthApp(r.Context(), v.Get(appKey(product)))
	if err != nil || !productMatches(app.Product, product) || app.Secret == "" || subtle.ConstantTimeCompare([]byte(app.Secret), []byte(v.Get("client_secret"))) != 1 {
		fail(w, product, 400, "invalid_client", "Invalid simulated client credentials")
		return
	}
	if product == "tiktok" && v.Get("grant_type") == "refresh_token" {
		p.refresh(w, r, app, v)
		return
	}
	if v.Get("grant_type") != "authorization_code" && !(product == "facebook" && v.Get("grant_type") == "") {
		fail(w, product, 400, "unsupported_grant_type", "Only authorization_code is implemented")
		return
	}
	c, err := p.engine.Store.GetOAuthCode(r.Context(), v.Get("code"))
	if err != nil || c.AppID != app.ID || c.RedirectURI != v.Get("redirect_uri") || !slices.Contains(app.RedirectURIs, c.RedirectURI) {
		fail(w, product, 400, "invalid_grant", "Invalid, expired, used, or mismatched authorization code")
		return
	}
	if c.Challenge != "" {
		verifier := v.Get("code_verifier")
		sum := sha256.Sum256([]byte(verifier))
		got := base64.RawURLEncoding.EncodeToString(sum[:])
		if product == "tiktok" {
			got = hex.EncodeToString(sum[:])
		}
		if !validVerifier(verifier) || subtle.ConstantTimeCompare([]byte(got), []byte(c.Challenge)) != 1 {
			fail(w, product, 400, "invalid_grant", "Invalid PKCE verifier")
			return
		}
	}
	if app.RequirePKCE && c.Challenge == "" {
		fail(w, product, 400, "invalid_grant", "App now requires PKCE; authorize again")
		return
	}
	if !slices.Contains(app.AccountIDs, c.AccountID) {
		fail(w, product, 400, "invalid_grant", "Account is no longer assigned to app")
		return
	}
	for _, scope := range c.Scopes {
		if !slices.Contains(app.ApprovedScopes, scope) {
			fail(w, product, 400, "invalid_grant", "App approval changed; authorize again")
			return
		}
	}
	target, err := p.validateAssets(r, app, c.AccountID, c.Scopes, c.AssetIDs)
	if err != nil {
		fail(w, product, 400, "invalid_grant", "Page assignment or tasks changed; authorize again")
		return
	}
	now, _, err := p.engine.Store.Clock(r.Context())
	if err != nil {
		fail(w, product, 500, "server_error", "Storage failure")
		return
	}
	value, err := randomValue()
	if err != nil {
		fail(w, product, 500, "server_error", "Token generation failure")
		return
	}
	ttl := int64(3600)
	if product == "tiktok" {
		ttl = 86400
	}
	token := store.Token{Value: "mock_" + value, AccountID: target, AppID: app.ID, Product: app.Product, Scopes: c.Scopes, AssetIDs: c.AssetIDs, ExpiresAt: now + ttl, IssuedAt: now}
	if product == "tiktok" {
		refresh, err := randomValue()
		if err != nil {
			fail(w, product, 500, "server_error", "Token generation failure")
			return
		}
		token.RefreshToken = "mock_refresh_" + refresh
		token.RefreshExpiresAt = now + 31536000
	}
	grant := store.OAuthGrant{AppID: app.ID, AccountID: c.AccountID, Scopes: c.Scopes, AssetIDs: c.AssetIDs}
	if err = p.engine.Store.RedeemOAuthCode(r.Context(), c.Value, token, grant); err != nil {
		fail(w, product, 400, "invalid_grant", "Code already used, expired, or storage unavailable")
		return
	}
	switch product {
	case "instagram":
		// Live-verified (owner-reported Posty production use, 2026-10-19): the response
		// is a flat object. Meta's guide wraps it in a data array; see docs/oauth-consent.md.
		meta.JSON(w, 200, map[string]any{"access_token": token.Value, "user_id": target, "permissions": strings.Join(c.Scopes, ",")})
	case "tiktok":
		meta.JSON(w, 200, map[string]any{"access_token": token.Value, "expires_in": ttl, "open_id": target, "scope": strings.Join(c.Scopes, ","), "token_type": "Bearer", "refresh_token": token.RefreshToken, "refresh_expires_in": token.RefreshExpiresAt - now})
	default:
		meta.JSON(w, 200, map[string]any{"access_token": token.Value, "token_type": "bearer", "expires_in": ttl})
	}
}
func randomValue() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b), err
}
func callback(w http.ResponseWriter, r *http.Request, redirect, state string, result url.Values) {
	allowCallback(w, redirect)
	u, _ := url.Parse(redirect)
	q := u.Query()
	for _, key := range []string{"code", "error", "error_reason", "error_description", "scopes", "state"} {
		q.Del(key)
	}
	for k, v := range result {
		q[k] = v
	}
	if state != "" {
		q.Set("state", state)
	}
	u.RawQuery = q.Encode()
	http.Redirect(w, r, u.String(), http.StatusFound)
}
func clearCookie(w http.ResponseWriter, r *http.Request, id string) {
	http.SetCookie(w, &http.Cookie{Name: "sm_consent_" + id, Value: "", Path: "/oauth/consent", MaxAge: -1, HttpOnly: true, Secure: r.TLS != nil, SameSite: http.SameSiteLaxMode})
}
func securityHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; form-action 'self'; base-uri 'none'; frame-ancestors 'none'")
}
func fail(w http.ResponseWriter, product string, status int, code, message string) {
	switch product {
	case "instagram":
		meta.JSON(w, status, map[string]any{"error_type": "OAuthException", "code": 400, "error_message": message})
	case "facebook":
		meta.JSON(w, status, map[string]any{"error": map[string]any{"message": message, "type": "OAuthException", "code": 100}})
	default:
		meta.JSON(w, status, map[string]any{"error": code, "error_description": message, "log_id": "mock-oauth"})
	}
}
func scopeList(s string) ([]string, bool) {
	parts := strings.Split(s, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
		if parts[i] == "" || strings.ContainsAny(parts[i], " \t\r\n") {
			return nil, false
		}
	}
	return parts, len(parts) <= 32 && !duplicate(parts)
}
func duplicate(s []string) bool {
	seen := map[string]bool{}
	for _, v := range s {
		if seen[v] {
			return true
		}
		seen[v] = true
	}
	return false
}
func validChallenge(product, s string) bool {
	if product == "tiktok" {
		b, e := hex.DecodeString(s)
		return e == nil && len(b) == 32 && len(s) == 64 && s == strings.ToLower(s)
	}
	b, e := base64.RawURLEncoding.DecodeString(s)
	return e == nil && len(b) == 32 && len(s) == 43
}
func validVerifier(s string) bool {
	if len(s) < 43 || len(s) > 128 {
		return false
	}
	for _, c := range s {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || strings.ContainsRune("-._~", c)) {
			return false
		}
	}
	return true
}
func providerName(p string) string {
	switch p {
	case "facebook", "instagram_facebook":
		return "Facebook"
	case "instagram":
		return "Instagram"
	case "tiktok":
		return "TikTok"
	}
	return fmt.Sprint(p)
}
func knownScope(product, scope string) bool {
	common := []string{"public_profile", "email", "pages_show_list", "pages_read_engagement", "pages_read_user_content", "pages_manage_posts", "pages_manage_engagement", "pages_manage_metadata", "read_insights"}
	igfb := []string{"instagram_basic", "instagram_content_publish", "instagram_manage_comments", "instagram_manage_insights", "instagram_manage_messages"}
	switch product {
	case "instagram":
		return slices.Contains([]string{"instagram_business_basic", "instagram_business_content_publish", "instagram_business_manage_comments", "instagram_business_manage_messages", "instagram_business_manage_insights"}, scope)
	case "facebook":
		return slices.Contains(common, scope)
	case "instagram_facebook":
		return slices.Contains(common, scope) || slices.Contains(igfb, scope)
	case "tiktok":
		return slices.Contains([]string{"user.info.basic", "user.info.profile", "user.info.stats", "video.list", "video.upload", "video.publish"}, scope)
	}
	return false
}

// Refresh rotation is a synthetic deterministic policy: both old tokens are retired.
// TikTok permits refresh-token rotation; this simulator always rotates.
func (p *Provider) refresh(w http.ResponseWriter, r *http.Request, app store.OAuthApp, v url.Values) {
	old, err := p.engine.Store.OAuthRefreshToken(r.Context(), v.Get("refresh_token"))
	if err != nil || old.AppID != app.ID || old.Product != "tiktok" {
		fail(w, "tiktok", 400, "invalid_grant", "Invalid or expired refresh token")
		return
	}
	grant, err := p.engine.Store.OAuthGrant(r.Context(), app.ID, old.GrantAccountID)
	valid := err == nil && slices.Contains(app.AccountIDs, old.GrantAccountID)
	for _, scope := range old.Scopes {
		valid = valid && slices.Contains(app.ApprovedScopes, scope) && slices.Contains(grant.Scopes, scope)
	}
	if !valid {
		fail(w, "tiktok", 400, "invalid_grant", "App approval or user grant changed; authorize again")
		return
	}
	now, _, err := p.engine.Store.Clock(r.Context())
	if err != nil {
		fail(w, "tiktok", 500, "server_error", "Storage failure")
		return
	}
	access, err := randomValue()
	if err != nil {
		fail(w, "tiktok", 500, "server_error", "Token generation failure")
		return
	}
	refresh, err := randomValue()
	if err != nil {
		fail(w, "tiktok", 500, "server_error", "Token generation failure")
		return
	}
	next := old
	next.Value = "mock_" + access
	next.RefreshToken = "mock_refresh_" + refresh
	next.ExpiresAt = now + 86400
	// Rotation does not prolong the original refresh authorization lifetime.
	if err = p.engine.Store.RefreshOAuthToken(r.Context(), old.RefreshToken, next); err != nil {
		fail(w, "tiktok", 400, "invalid_grant", "Refresh token already used or expired")
		return
	}
	meta.JSON(w, 200, map[string]any{"access_token": next.Value, "expires_in": int64(86400), "refresh_token": next.RefreshToken, "refresh_expires_in": next.RefreshExpiresAt - now, "open_id": next.AccountID, "scope": strings.Join(next.Scopes, ","), "token_type": "Bearer"})
}

// Browsers enforce form-action across the redirect chain as well as the POST.
func allowCallback(w http.ResponseWriter, redirect string) {
	u, err := url.Parse(redirect)
	if err != nil {
		return
	}
	source := u.Scheme + "://" + u.Host
	if strings.ContainsAny(source, "; '\"<>\t\r\n") {
		return
	}
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; form-action 'self' "+source+"; base-uri 'none'; frame-ancestors 'none'")
}
