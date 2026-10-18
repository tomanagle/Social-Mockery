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
