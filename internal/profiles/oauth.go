package profiles

import (
	"fmt"
	"net/url"
	"strings"
	"unicode"

	"github.com/tomanagle/Social-Mockery/internal/store"
)

func validList(v []string, max int) bool {
	if len(v) > max {
		return false
	}
	seen := map[string]bool{}
	for _, s := range v {
		if strings.TrimSpace(s) == "" || len(s) > 100 || seen[s] {
			return false
		}
		seen[s] = true
	}
	return true
}
func contains(v []string, s string) bool {
	for _, x := range v {
		if x == s {
			return true
		}
	}
	return false
}
func validRedirect(raw string) bool {
	if len(raw) > 2048 || strings.TrimSpace(raw) != raw {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil || u.Fragment != "" || strings.Contains(raw, "#") {
		return false
	}
	// This authority is also rendered as a CSP callback source. Forbid source
	// delimiters, wildcard syntax and whitespace rather than trusting URL parsing.
	if strings.ContainsAny(u.Host, ";'\"`,*\\") || strings.ContainsFunc(u.Host, unicode.IsSpace) {
		return false
	}
	if u.Scheme == "https" {
		return true
	}
	return u.Scheme == "http" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1" || u.Hostname() == "::1")
}

// graphVersion accepts Graph release labels such as v25.0.
func graphVersion(v string) bool {
	major, ok := strings.CutSuffix(strings.TrimPrefix(v, "v"), ".0")
	return ok && strings.HasPrefix(v, "v") && len(major) >= 1 && len(major) <= 2 && validID(major)
}

func validateOAuth(s store.Seed, ids map[string]bool) error {
	if len(s.Apps) > 100 || len(s.Grants) > 2000 || len(s.Assets) > 2000 {
		return fmt.Errorf("OAuth fixture limit exceeded")
	}
	apps := map[string]store.OAuthApp{}
	for _, a := range s.Apps {
		if a.ID == "" || len(a.ID) > 128 || strings.TrimSpace(a.ID) != a.ID {
			return fmt.Errorf("invalid OAuth app id")
		}
		if _, ok := apps[a.ID]; ok {
			return fmt.Errorf("duplicate OAuth app id")
		}
		if a.Name == "" || len(a.Name) > 200 || a.Secret == "" || len(a.Secret) > 1024 {
			return fmt.Errorf("invalid OAuth app name or secret")
		}
		switch a.Product {
		case "facebook", "instagram_facebook", "instagram", "tiktok":
		default:
			return fmt.Errorf("unsupported OAuth product")
		}
		if len(a.RedirectURIs) == 0 || len(a.RedirectURIs) > 20 {
			return fmt.Errorf("invalid OAuth redirect URI count")
		}
		uris := map[string]bool{}
		for _, uri := range a.RedirectURIs {
			if !validRedirect(uri) || uris[uri] {
				return fmt.Errorf("invalid or duplicate OAuth redirect URI")
			}
			if a.Product == "tiktok" {
				u, err := url.Parse(uri)
				if err != nil {
					return fmt.Errorf("invalid TikTok redirect URI")
				}
				if u.RawQuery != "" || u.ForceQuery {
					return fmt.Errorf("TikTok redirect URI cannot contain query")
				}
				if a.RequirePKCE {
					if u.Hostname() != "localhost" && u.Hostname() != "127.0.0.1" {
						return fmt.Errorf("TikTok desktop redirect must be loopback")
					}
				} else if u.Scheme != "https" {
					return fmt.Errorf("TikTok web redirect must be HTTPS")
				}
			}
			uris[uri] = true
		}
		if !validList(a.ApprovedScopes, 32) || !validList(a.AccountIDs, 1000) {
			return fmt.Errorf("invalid OAuth scopes or accounts")
		}
		if a.GraphAPIVersion != "" && (a.Product == "tiktok" || !graphVersion(a.GraphAPIVersion)) {
			return fmt.Errorf("graph_api_version must look like v25.0 and applies only to Meta apps")
		}
		for _, id := range a.AccountIDs {
			if !ids[id] {
				return fmt.Errorf("OAuth app references missing account")
			}
		}
		apps[a.ID] = a
	}
	assets := map[string]store.OAuthAsset{}
	for _, a := range s.Assets {
		if !validID(a.ID) || !ids[a.AccountID] || len(a.Name) > 200 || !validList(a.Tasks, 32) {
			return fmt.Errorf("invalid OAuth asset")
		}
		if _, ok := assets[a.ID]; ok {
			return fmt.Errorf("duplicate OAuth asset")
		}
		if a.InstagramAccountID != "" && !ids[a.InstagramAccountID] {
			return fmt.Errorf("OAuth asset references missing Instagram account")
		}
		assets[a.ID] = a
	}
	grants := map[string]bool{}
	for _, g := range s.Grants {
		a, ok := apps[g.AppID]
		key := g.AppID + "\x00" + g.AccountID
		if !ok || !contains(a.AccountIDs, g.AccountID) || grants[key] || !validList(g.Scopes, 32) || !validList(g.AssetIDs, 2000) {
			return fmt.Errorf("invalid OAuth grant")
		}
		for _, scope := range g.Scopes {
			if !contains(a.ApprovedScopes, scope) {
				return fmt.Errorf("OAuth grant scope not approved")
			}
		}
		for _, id := range g.AssetIDs {
			asset, ok := assets[id]
			if !ok || asset.AccountID != g.AccountID {
				return fmt.Errorf("OAuth grant asset not owned by account")
			}
		}
		grants[key] = true
	}
	refreshTokens := map[string]bool{}
	for _, t := range s.Tokens {
		if t.IssuedAt < 0 || (t.Kind != "" && (t.Kind != "long_lived" || t.Product != "instagram" || t.AppID == "" || t.ExpiresAt <= 0)) {
			return fmt.Errorf("token kind long_lived applies only to expiring Instagram Login app tokens")
		}
		if t.RefreshToken != "" {
			if t.Product != "tiktok" || t.AppID == "" || len(t.RefreshToken) > 1024 || t.RefreshExpiresAt <= 0 || refreshTokens[t.RefreshToken] {
				return fmt.Errorf("invalid OAuth refresh token")
			}
			refreshTokens[t.RefreshToken] = true
		} else if t.RefreshExpiresAt != 0 {
			return fmt.Errorf("refresh expiry requires refresh token")
		}
		if t.AppID == "" {
			if t.Product != "" || len(t.AssetIDs) > 0 || t.GrantAccountID != "" {
				return fmt.Errorf("OAuth token requires app")
			}
			continue
		}
		a, ok := apps[t.AppID]
		if !ok || t.Product != a.Product {
			return fmt.Errorf("invalid OAuth token app")
		}
		actor := t.AccountID
		if t.GrantAccountID != "" {
			actor = t.GrantAccountID
		}
		if t.GrantAccountID == "" && !contains(a.AccountIDs, actor) && a.Product == "instagram_facebook" {
			for _, id := range t.AssetIDs {
				asset, exists := assets[id]
				if exists && asset.InstagramAccountID == t.AccountID && contains(a.AccountIDs, asset.AccountID) {
					actor = asset.AccountID
					break
				}
			}
		}
		if !contains(a.AccountIDs, actor) {
			return fmt.Errorf("invalid OAuth token account")
		}
		if actor != t.AccountID {
			linked := false
			if a.Product == "instagram_facebook" {
				for _, id := range t.AssetIDs {
					asset, exists := assets[id]
					if exists && asset.AccountID == actor && asset.InstagramAccountID == t.AccountID {
						linked = true
					}
				}
			}
			if !linked {
				return fmt.Errorf("OAuth token account is not linked to grant account")
			}
		}
		for _, scope := range t.Scopes {
			if !contains(a.ApprovedScopes, scope) {
				return fmt.Errorf("OAuth token scope not approved")
			}
		}
		for _, id := range t.AssetIDs {
			asset, ok := assets[id]
			if !ok || asset.AccountID != actor {
				return fmt.Errorf("invalid OAuth token asset")
			}
		}
	}
	return nil
}
