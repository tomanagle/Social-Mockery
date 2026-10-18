package dbstore_test

import (
	"errors"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/tomanagle/Social-Mockery/internal/profiles"
	"github.com/tomanagle/Social-Mockery/internal/store"
	"github.com/tomanagle/Social-Mockery/internal/store/db"
	"github.com/tomanagle/Social-Mockery/internal/store/dbstore"
)

func oauthSeed() store.Seed {
	s := profiles.Default()
	s.Apps = []store.OAuthApp{{ID: "app", Name: "Demo", Secret: "secret", Product: "instagram", RedirectURIs: []string{"http://localhost/callback"}, ApprovedScopes: []string{"instagram_business_basic"}, AccountIDs: []string{s.Accounts[0].ID}}}
	return s
}
func TestOAuthAtomicPersistenceAndUpsert(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.db")
	conn, err := db.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	repo := dbstore.New(conn)
	s := oauthSeed()
	const now = 1767225600
	if err = repo.Reset(ctx, s, now); err != nil {
		t.Fatal(err)
	}
	c, err := repo.CreateContainer(ctx, store.Container{AccountID: s.Accounts[0].ID}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = repo.Advance(ctx, 5); err != nil {
		t.Fatal(err)
	}
	p, err := repo.Publish(ctx, c.AccountID, c.ID, 10)
	if err != nil {
		t.Fatal(err)
	}
	session := store.OAuthSession{ID: "session", AppID: "app", RedirectURI: "http://localhost/callback", ExpiresAt: now + 600, Scopes: s.Apps[0].ApprovedScopes}
	code := store.OAuthCode{Value: "code", AppID: "app", AccountID: c.AccountID, RedirectURI: session.RedirectURI, Scopes: session.Scopes, ExpiresAt: now + 600}
	if err = repo.PutOAuthSession(ctx, session); err != nil {
		t.Fatal(err)
	}
	var wins atomic.Int32
	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			e := repo.CompleteOAuthSession(ctx, session.ID, code)
			if e == nil {
				wins.Add(1)
			} else if !errors.Is(e, store.ErrNotFound) {
				t.Errorf("complete: %v", e)
			}
		}()
	}
	wg.Wait()
	if wins.Load() != 1 {
		t.Fatalf("session winners %d", wins.Load())
	}
	if err = conn.Close(); err != nil {
		t.Fatal(err)
	}
	conn, err = db.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	repo = dbstore.New(conn)
	if _, err = repo.GetOAuthCode(ctx, code.Value); err != nil {
		t.Fatal(err)
	}
	token := store.Token{Value: "new-token", AppID: "app", Product: "instagram", AccountID: c.AccountID, Scopes: code.Scopes, ExpiresAt: now + 3600}
	grant := store.OAuthGrant{AppID: "app", AccountID: c.AccountID, Scopes: code.Scopes}
	wins.Store(0)
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			e := repo.RedeemOAuthCode(ctx, code.Value, token, grant)
			if e == nil {
				wins.Add(1)
			} else if !errors.Is(e, store.ErrNotFound) {
				t.Errorf("redeem: %v", e)
			}
		}()
	}
	wg.Wait()
	if wins.Load() != 1 {
		t.Fatalf("code winners %d", wins.Load())
	}
	if _, err = repo.OAuthGrant(ctx, "app", c.AccountID); err != nil {
		t.Fatal(err)
	}
	s.Apps[0].Name = "Updated"
	if err = repo.UpsertOAuthFixtures(ctx, s); err != nil {
		t.Fatal(err)
	}
	if _, err = repo.Token(ctx, token.Value); err != nil {
		t.Fatal(err)
	}
	if _, err = repo.Post(ctx, p.ID); err != nil {
		t.Fatal(err)
	}
	if err = repo.Reset(ctx, s, now); err != nil {
		t.Fatal(err)
	}
	if _, err = repo.Token(ctx, token.Value); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("reset token: %v", err)
	}
	if _, err = repo.OAuthGrant(ctx, "app", c.AccountID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("reset grant: %v", err)
	}
}
func TestOAuthExpiryAndRollback(t *testing.T) {
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	repo := dbstore.New(conn)
	s := oauthSeed()
	const now = 1767225600
	if err = repo.Reset(ctx, s, now); err != nil {
		t.Fatal(err)
	}
	session := store.OAuthSession{ID: "s", AppID: "app", ExpiresAt: now + 10}
	if err = repo.PutOAuthSession(ctx, session); err != nil {
		t.Fatal(err)
	}
	if _, err = repo.Advance(ctx, 10); err != nil {
		t.Fatal(err)
	}
	if _, err = repo.GetOAuthSession(ctx, "s"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("expired session: %v", err)
	}
	code := store.OAuthCode{Value: "c", AppID: "app", AccountID: s.Accounts[0].ID, ExpiresAt: now + 20}
	if err = repo.PutOAuthCode(ctx, code); err != nil {
		t.Fatal(err)
	}
	bad := store.Token{Value: "bad", AppID: "wrong"}
	if err = repo.RedeemOAuthCode(ctx, "c", bad, store.OAuthGrant{}); !errors.Is(err, store.ErrConflict) {
		t.Fatalf("mismatched redemption: %v", err)
	}
	if _, err = repo.GetOAuthCode(ctx, "c"); err != nil {
		t.Fatalf("failed redemption consumed code: %v", err)
	}
	if _, err = repo.Advance(ctx, 10); err != nil {
		t.Fatal(err)
	}
	if _, err = repo.GetOAuthCode(ctx, "c"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("expired code: %v", err)
	}
}

func TestOAuthInstagramFacebookLinkedAccount(t *testing.T) {
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	repo := dbstore.New(conn)
	s := oauthSeed()
	s.Apps[0].Product = "instagram_facebook"
	actor := s.Accounts[0].ID
	s.Accounts = append(s.Accounts, store.Account{ID: "42", Username: "linked", Type: "BUSINESS"})
	s.Assets = []store.OAuthAsset{{ID: "99", Name: "Page", AccountID: actor, InstagramAccountID: "42", Tasks: []string{"CREATE_CONTENT"}}}
	if err = repo.Reset(ctx, s, 1767225600); err != nil {
		t.Fatal(err)
	}
	code := store.OAuthCode{Value: "code", AppID: "app", AccountID: actor, Scopes: s.Apps[0].ApprovedScopes, AssetIDs: []string{"99"}, ExpiresAt: 1767226200}
	if err = repo.PutOAuthCode(ctx, code); err != nil {
		t.Fatal(err)
	}
	token := store.Token{Value: "linked-token", AppID: "app", Product: "instagram_facebook", AccountID: "42", Scopes: code.Scopes, AssetIDs: code.AssetIDs}
	grant := store.OAuthGrant{AppID: "app", AccountID: actor, Scopes: code.Scopes, AssetIDs: code.AssetIDs}
	if err = repo.RedeemOAuthCode(ctx, code.Value, token, grant); err != nil {
		t.Fatal(err)
	}
	got, err := repo.Token(ctx, token.Value)
	if err != nil || got.AccountID != "42" {
		t.Fatalf("linked token: %+v %v", got, err)
	}
	if _, err = repo.OAuthGrant(ctx, "app", actor); err != nil {
		t.Fatal(err)
	}
}

func TestOAuthPendingCapacityPrunesExpiry(t *testing.T) {
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	repo := dbstore.New(conn)
	if err = repo.Reset(ctx, oauthSeed(), 1767225600); err != nil {
		t.Fatal(err)
	}
	_, err = conn.Exec("WITH RECURSIVE n(x) AS (SELECT 1 UNION ALL SELECT x+1 FROM n WHERE x<2000) INSERT INTO oauth_sessions SELECT 's'||x,'app',1767225610,'{}' FROM n")
	if err != nil {
		t.Fatal(err)
	}
	session := store.OAuthSession{ID: "new", AppID: "app", ExpiresAt: 1767226200}
	if err = repo.PutOAuthSession(ctx, session); !errors.Is(err, store.ErrCapacity) {
		t.Fatalf("capacity: %v", err)
	}
	if _, err = repo.Advance(ctx, 10); err != nil {
		t.Fatal(err)
	}
	if err = repo.PutOAuthSession(ctx, session); err != nil {
		t.Fatal(err)
	}
	var count int
	if err = conn.QueryRow("SELECT count(*) FROM oauth_sessions").Scan(&count); err != nil || count != 1 {
		t.Fatalf("expiry cleanup: count=%d err=%v", count, err)
	}
}

func TestOAuthTokenCapacityPrunesOnlyExpiredIssuedTokens(t *testing.T) {
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	repo := dbstore.New(conn)
	s := oauthSeed()
	const now = 1767225600
	if err = repo.Reset(ctx, s, now); err != nil {
		t.Fatal(err)
	}
	_, err = conn.Exec("WITH RECURSIVE n(x) AS (SELECT 1 UNION ALL SELECT x+1 FROM n WHERE x<2000) INSERT INTO tokens(value,account_id,scopes,expires_at,app_id,product) SELECT 'issued'||x,?,'[]',?,'app','instagram' FROM n", s.Accounts[0].ID, now+10)
	if err != nil {
		t.Fatal(err)
	}
	code := store.OAuthCode{Value: "new-code", AppID: "app", AccountID: s.Accounts[0].ID, ExpiresAt: now + 600}
	if err = repo.PutOAuthCode(ctx, code); err != nil {
		t.Fatal(err)
	}
	token := store.Token{Value: "new-token", AppID: "app", Product: "instagram", AccountID: code.AccountID, ExpiresAt: now + 3600}
	grant := store.OAuthGrant{AppID: "app", AccountID: code.AccountID}
	if err = repo.RedeemOAuthCode(ctx, code.Value, token, grant); !errors.Is(err, store.ErrCapacity) {
		t.Fatalf("token capacity: %v", err)
	}
	if _, err = repo.GetOAuthCode(ctx, code.Value); err != nil {
		t.Fatalf("capacity failure consumed code: %v", err)
	}
	if _, err = repo.Advance(ctx, 10); err != nil {
		t.Fatal(err)
	}
	if err = repo.RedeemOAuthCode(ctx, code.Value, token, grant); err != nil {
		t.Fatal(err)
	}
	var count int
	if err = conn.QueryRow("SELECT count(*) FROM tokens").Scan(&count); err != nil || count != 2 {
		t.Fatalf("token cleanup: %d %v", count, err)
	}
	if _, err = repo.Token(ctx, "mock-token"); err != nil {
		t.Fatalf("legacy token deleted: %v", err)
	}
}

func TestOAuthRefreshRotationIsAtomicAndScoped(t *testing.T) {
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	repo := dbstore.New(conn)
	s := oauthSeed()
	s.Apps[0].Product = "tiktok"
	s.Apps[0].RedirectURIs = []string{"https://client.example.test/callback"}
	const now = 1767225600
	if err = repo.Reset(ctx, s, now); err != nil {
		t.Fatal(err)
	}
	code := store.OAuthCode{Value: "code", AppID: "app", AccountID: s.Accounts[0].ID, Scopes: s.Apps[0].ApprovedScopes, ExpiresAt: now + 600}
	if err = repo.PutOAuthCode(ctx, code); err != nil {
		t.Fatal(err)
	}
	token := store.Token{Value: "old-access", AppID: "app", Product: "tiktok", GrantAccountID: code.AccountID, AccountID: code.AccountID, Scopes: code.Scopes, ExpiresAt: now + 10, RefreshToken: "old-refresh", RefreshExpiresAt: now + 1000}
	grant := store.OAuthGrant{AppID: "app", AccountID: code.AccountID, Scopes: code.Scopes}
	if err = repo.RedeemOAuthCode(ctx, code.Value, token, grant); err != nil {
		t.Fatal(err)
	}
	if _, err = repo.Advance(ctx, 10); err != nil {
		t.Fatal(err)
	}
	old, err := repo.OAuthRefreshToken(ctx, "old-refresh")
	if err != nil || old.Value != "old-access" {
		t.Fatalf("lookup: %+v %v", old, err)
	}
	next := token
	next.Value = "new-access"
	next.RefreshToken = "new-refresh"
	next.ExpiresAt = now + 100
	next.RefreshExpiresAt = now + 2000
	broadened := next
	broadened.Scopes = []string{"unauthorized"}
	if err = repo.RefreshOAuthToken(ctx, "old-refresh", broadened); !errors.Is(err, store.ErrConflict) {
		t.Fatalf("refresh scope escalation: %v", err)
	}
	var wins atomic.Int32
	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := repo.RefreshOAuthToken(ctx, "old-refresh", next)
			if err == nil {
				wins.Add(1)
			} else if !errors.Is(err, store.ErrNotFound) {
				t.Errorf("refresh: %v", err)
			}
		}()
	}
	wg.Wait()
	if wins.Load() != 1 {
		t.Fatalf("refresh winners: %d", wins.Load())
	}
	if _, err = repo.Token(ctx, "old-access"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("old access retained: %v", err)
	}
	if _, err = repo.OAuthRefreshToken(ctx, "old-refresh"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("old refresh retained: %v", err)
	}
	if _, err = repo.OAuthRefreshToken(ctx, "new-refresh"); err != nil {
		t.Fatal(err)
	}
	if _, err = repo.Advance(ctx, 2000); err != nil {
		t.Fatal(err)
	}
	if _, err = repo.OAuthRefreshToken(ctx, "new-refresh"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("expired refresh: %v", err)
	}
}
