package oauth

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/tomanagle/Social-Mockery/internal/providers/meta"
	"github.com/tomanagle/Social-Mockery/internal/store"
)

// instagramLongLivedSeconds is the documented 60-day long-lived lifetime. The
// exact second count is a simulator choice; Meta's example shows 5183944.
const instagramLongLivedSeconds = 60 * 24 * 60 * 60

// instagramLongLived serves the documented Instagram Login long-lived token
// exchange (ig_exchange_token) and refresh (ig_refresh_token) on graph.instagram.com.
func (p *Provider) instagramLongLived(w http.ResponseWriter, r *http.Request, refresh bool) {
	securityHeaders(w)
	u, err := url.Parse(strings.TrimPrefix(r.RequestURI, "/"))
	if err != nil {
		graphFail(w, 400, 100, "Malformed request URL")
		return
	}
	values, err := parameters(w, r, u.RawQuery)
	if err != nil {
		graphFail(w, 400, 100, "Malformed or duplicate parameters")
		return
	}
	grant, allowed := "ig_exchange_token", map[string]bool{"grant_type": true, "access_token": true, "client_secret": true}
	if refresh {
		grant, allowed = "ig_refresh_token", map[string]bool{"grant_type": true, "access_token": true}
	}
	for key := range values {
		if !allowed[key] {
			graphFail(w, 400, 100, "Unsupported parameter: "+key)
			return
		}
	}
	if values.Get("grant_type") != grant {
		graphFail(w, 400, 100, "grant_type must be "+grant)
		return
	}
	p.engine.Mu.Lock()
	defer p.engine.Mu.Unlock()
	ctx := r.Context()
	t, err := p.engine.Store.Token(ctx, values.Get("access_token"))
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			graphFail(w, 400, 190, "Invalid OAuth access token")
		} else {
			graphFail(w, 500, 1, "Storage failure")
		}
		return
	}
	now, _, err := p.engine.Store.Clock(ctx)
	if err != nil {
		graphFail(w, 500, 1, "Storage failure")
		return
	}
	if t.AppID == "" || t.Product != "instagram" {
		graphFail(w, 400, 190, "Access token was not issued by Instagram Login")
		return
	}
	if t.ExpiresAt > 0 && now >= t.ExpiresAt {
		graphFail(w, 400, 190, "Access token has expired")
		return
	}
	app, err := p.engine.Store.OAuthApp(ctx, t.AppID)
	if err != nil || !p.instagramGrantCurrent(ctx, app, t) {
		graphFail(w, 400, 190, "Access token grant or app approval is no longer valid")
		return
	}
	if !slices.Contains(t.Scopes, "instagram_business_basic") {
		graphFail(w, 403, 200, "instagram_business_basic was not granted")
		return
	}
	if refresh {
		if t.Kind != "long_lived" {
			graphFail(w, 400, 100, "Only long-lived Instagram User access tokens can be refreshed")
			return
		}
		if now-t.IssuedAt < 86400 {
			graphFail(w, 400, 100, "Long-lived token must be at least 24 hours old to refresh")
			return
		}
		if err = p.engine.Store.ExtendInstagramToken(ctx, t.Value, now+instagramLongLivedSeconds, now); err != nil {
			graphFail(w, 400, 100, "Token could not be refreshed")
			return
		}
		meta.JSON(w, 200, map[string]any{"access_token": t.Value, "token_type": "bearer", "expires_in": instagramLongLivedSeconds})
		return
	}
	if subtle.ConstantTimeCompare([]byte(values.Get("client_secret")), []byte(app.Secret)) != 1 {
		graphFail(w, 400, 100, "Invalid client_secret for the token's app")
		return
	}
	if t.Kind == "long_lived" {
		graphFail(w, 400, 100, "A short-lived Instagram User access token is required")
		return
	}
	long := t
	long.Value = longLivedValue(t.Value)
	long.Kind = "long_lived"
	long.IssuedAt = now
	long.ExpiresAt = now + instagramLongLivedSeconds
	long.RefreshToken, long.RefreshExpiresAt = "", 0
	if err = p.engine.Store.ExchangeInstagramToken(ctx, t.Value, long); err != nil {
		if errors.Is(err, store.ErrCapacity) {
			graphFail(w, 507, 1, "OAuth token capacity reached; reset the simulator")
		} else {
			graphFail(w, 400, 100, "Token could not be exchanged")
		}
		return
	}
	meta.JSON(w, 200, map[string]any{"access_token": long.Value, "token_type": "bearer", "expires_in": instagramLongLivedSeconds})
}

// instagramGrantCurrent applies the same app-approval and grant checks that
// Instagram resource calls apply to OAuth-issued tokens.
func (p *Provider) instagramGrantCurrent(ctx context.Context, app store.OAuthApp, t store.Token) bool {
	grant, err := p.engine.Store.OAuthGrant(ctx, t.AppID, t.GrantAccountID)
	if err != nil || app.Product != "instagram" || !slices.Contains(app.AccountIDs, t.GrantAccountID) {
		return false
	}
	for _, scope := range t.Scopes {
		if !slices.Contains(app.ApprovedScopes, scope) || !slices.Contains(grant.Scopes, scope) {
			return false
		}
	}
	return true
}

// longLivedValue derives one long-lived token per short-lived token, which
// bounds state when a client repeats the exchange.
func longLivedValue(short string) string {
	sum := sha256.Sum256([]byte("instagram-long-lived\x00" + short))
	return "mock_ll_" + hex.EncodeToString(sum[:20])
}

func graphFail(w http.ResponseWriter, status, code int, message string) {
	meta.JSON(w, status, map[string]any{"error": map[string]any{"message": message, "type": "OAuthException", "code": code}})
}
