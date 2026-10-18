package profiles

import (
	"bytes"
	"fmt"
	"github.com/tomanagle/Social-Mockery/internal/store"
	"gopkg.in/yaml.v3"
	"io"
	"os"
	"strings"
	"time"
)

const MaxBytes = 1 << 20

func Load(path string) (store.Seed, error) {
	f, err := os.Open(path)
	if err != nil {
		return store.Seed{}, err
	}
	defer f.Close()
	return Decode(f)
}
func Decode(r io.Reader) (store.Seed, error) {
	var s store.Seed
	b, err := io.ReadAll(io.LimitReader(r, MaxBytes+1))
	if err != nil {
		return s, err
	}
	if len(b) > MaxBytes {
		return s, fmt.Errorf("profile exceeds 1 MiB")
	}
	d := yaml.NewDecoder(bytes.NewReader(b))
	d.KnownFields(true)
	if err = d.Decode(&s); err != nil {
		return s, err
	}
	var extra any
	if err = d.Decode(&extra); err != io.EOF {
		return s, fmt.Errorf("exactly one profile document is required")
	}
	return s, Validate(s)
}
func Validate(s store.Seed) error {
	if s.Version != 1 {
		return fmt.Errorf("profile version must be 1")
	}
	if _, err := time.Parse(time.RFC3339, s.Clock); err != nil {
		return fmt.Errorf("clock must be RFC3339: %w", err)
	}
	if s.ProcessingSeconds < 0 || s.ProcessingSeconds > 86400 {
		return fmt.Errorf("processing_seconds must be 0..86400")
	}
	if len(s.Accounts) > 1000 || len(s.Tokens) > 2000 {
		return fmt.Errorf("profile account/token limit exceeded")
	}
	ids := map[string]bool{}
	tokens := map[string]bool{}
	for _, a := range s.Accounts {
		if !validID(a.ID) || ids[a.ID] {
			return fmt.Errorf("invalid or duplicate account id %q", a.ID)
		}
		ids[a.ID] = true
		if len(a.Username) > 100 {
			return fmt.Errorf("username too long")
		}
		if a.Type != "BUSINESS" && a.Type != "CREATOR" && a.Type != "PERSONAL" {
			return fmt.Errorf("unsupported account type %q", a.Type)
		}
	}
	for _, t := range s.Tokens {
		if t.Value == "" || len(t.Value) > 1024 || tokens[t.Value] {
			return fmt.Errorf("invalid or duplicate token")
		}
		tokens[t.Value] = true
		if !ids[t.AccountID] {
			return fmt.Errorf("token references missing account")
		}
		if len(t.Scopes) > 32 {
			return fmt.Errorf("too many scopes")
		}
		for _, scope := range t.Scopes {
			if len(scope) > 100 || strings.TrimSpace(scope) == "" {
				return fmt.Errorf("invalid scope")
			}
		}
	}
	return nil
}
func validID(s string) bool {
	if len(s) == 0 || len(s) > 64 {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
func Default() store.Seed {
	return store.Seed{Version: 1, Clock: "2026-01-01T00:00:00Z", ProcessingSeconds: 5, Accounts: []store.Account{{ID: "17841400000000001", Username: "mock_creator", Type: "BUSINESS"}}, Tokens: []store.Token{{Value: "mock-token", AccountID: "17841400000000001", Scopes: []string{"instagram_business_basic", "instagram_business_content_publish", "instagram_basic", "instagram_content_publish"}}}}
}
