package profiles

import (
	"github.com/tomanagle/Social-Mockery/internal/store"
	"testing"
)

func TestOAuthFixtureValidation(t *testing.T) {
	for _, uri := range []string{"https://example.test/callback", "http://localhost:8080/callback", "http://127.0.0.1/callback", "http://[::1]/callback"} {
		s := Default()
		s.Apps = []store.OAuthApp{{ID: "app", Name: "Demo", Secret: "secret", Product: "instagram", RedirectURIs: []string{uri}, AccountIDs: []string{s.Accounts[0].ID}}}
		if err := Validate(s); err != nil {
			t.Fatalf("%s: %v", uri, err)
		}
	}
	for _, uri := range []string{"http://example.test/callback", "https://user@example.test/callback", "https://example.test/callback#fragment", "javascript:alert(1)", "//example.test/callback"} {
		s := Default()
		s.Apps = []store.OAuthApp{{ID: "app", Name: "Demo", Secret: "secret", Product: "instagram", RedirectURIs: []string{uri}}}
		if err := Validate(s); err == nil {
			t.Fatalf("accepted %s", uri)
		}
	}
	s := Default()
	s.Apps = []store.OAuthApp{{ID: "app", Name: "Demo", Secret: "secret", Product: "instagram", RedirectURIs: []string{"https://example.test/callback"}, AccountIDs: []string{"999"}}}
	if err := Validate(s); err == nil {
		t.Fatal("accepted dangling account")
	}
}

func TestOAuthRedirectAuthoritySafeForCSP(t *testing.T) {
	for _, uri := range []string{
		"https://client.example.test;evil.example/callback",
		"https://client.example.test'unsafe-inline'/callback",
		"https://client.example.test\"/callback",
		"https://client.example.test evil.example/callback",
		"https://*.example.test/callback",
		"https://client.example.test,evil.example/callback",
	} {
		s := Default()
		s.Apps = []store.OAuthApp{{ID: "app", Name: "Demo", Secret: "secret", Product: "instagram", RedirectURIs: []string{uri}}}
		if Validate(s) == nil {
			t.Fatalf("accepted unsafe callback authority: %q", uri)
		}
	}
	s := Default()
	s.Apps = []store.OAuthApp{{ID: "app", Name: "Demo", Secret: "secret", Product: "instagram", RedirectURIs: []string{"https://client.example.test/callback?return=%2Fhome%3Fx%3D1&label=hello%20world"}}}
	if err := Validate(s); err != nil {
		t.Fatalf("valid encoded query rejected: %v", err)
	}
}

func TestTokenLifecycleFixtureValidation(t *testing.T) {
	app := func(product, version string) store.OAuthApp {
		return store.OAuthApp{ID: "app", Name: "Demo", Secret: "secret", Product: product, RedirectURIs: []string{"https://example.test/callback"}, ApprovedScopes: []string{"instagram_business_basic"}, AccountIDs: []string{"17841400000000001"}, GraphAPIVersion: version}
	}
	token := func(kind string, expires int64) store.Token {
		return store.Token{Value: "long", AccountID: "17841400000000001", AppID: "app", Product: "instagram", Scopes: []string{"instagram_business_basic"}, Kind: kind, ExpiresAt: expires, IssuedAt: 10}
	}
	cases := []struct {
		name  string
		app   store.OAuthApp
		token *store.Token
		valid bool
	}{
		{"meta app version", app("instagram", "v25.0"), nil, true},
		{"no app version", app("instagram", ""), nil, true},
		{"malformed version", app("instagram", "25.0"), nil, false},
		{"minor version", app("instagram", "v25.1"), nil, false},
		{"tiktok version", store.OAuthApp{ID: "app", Name: "Demo", Secret: "secret", Product: "tiktok", RedirectURIs: []string{"https://example.test/callback"}, GraphAPIVersion: "v25.0"}, nil, false},
		{"long-lived token", app("instagram", ""), ptr(token("long_lived", 100)), true},
		{"non-expiring long-lived token", app("instagram", ""), ptr(token("long_lived", 0)), false},
		{"unknown kind", app("instagram", ""), ptr(token("forever", 100)), false},
		{"long-lived facebook token", func() store.OAuthApp {
			a := app("instagram_facebook", "")
			a.ApprovedScopes = []string{"instagram_basic"}
			return a
		}(),
			func() *store.Token {
				t := token("long_lived", 100)
				t.Product = "instagram_facebook"
				t.Scopes = []string{"instagram_basic"}
				return &t
			}(), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := Default()
			s.Apps = []store.OAuthApp{tc.app}
			if tc.token != nil {
				s.Tokens = append(s.Tokens, *tc.token)
			}
			if err := Validate(s); (err == nil) != tc.valid {
				t.Fatalf("valid=%v err=%v", tc.valid, err)
			}
		})
	}
}

func ptr[T any](v T) *T { return &v }
