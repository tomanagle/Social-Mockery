package dbstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"slices"

	"github.com/tomanagle/Social-Mockery/internal/profiles"
	"github.com/tomanagle/Social-Mockery/internal/store"
)

const maxOAuthRecords = 2000

func decodeRow(row *sql.Row, v any) error {
	var b string
	if err := row.Scan(&b); err != nil {
		return mapped(err)
	}
	return json.Unmarshal([]byte(b), v)
}
func (s *Store) OAuthApp(ctx context.Context, id string) (v store.OAuthApp, err error) {
	err = decodeRow(s.db.QueryRowContext(ctx, "SELECT data FROM oauth_apps WHERE id=?", id), &v)
	return
}
func (s *Store) OAuthGrant(ctx context.Context, app, account string) (v store.OAuthGrant, err error) {
	err = decodeRow(s.db.QueryRowContext(ctx, "SELECT data FROM oauth_grants WHERE app_id=? AND account_id=?", app, account), &v)
	return
}
func (s *Store) OAuthAccounts(ctx context.Context, id string) ([]store.Account, error) {
	app, err := s.OAuthApp(ctx, id)
	if err != nil {
		return nil, err
	}
	out := []store.Account{}
	for _, id := range app.AccountIDs {
		a, err := s.Account(ctx, id)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, nil
}
func (s *Store) OAuthAssets(ctx context.Context, id string) ([]store.OAuthAsset, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT data FROM oauth_assets WHERE account_id=? ORDER BY id", id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []store.OAuthAsset{}
	for rows.Next() {
		var b string
		var v store.OAuthAsset
		if err = rows.Scan(&b); err != nil {
			return nil, err
		}
		if err = json.Unmarshal([]byte(b), &v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func upsertFixtures(ctx context.Context, tx *sql.Tx, seed store.Seed) error {
	for _, v := range seed.Accounts {
		if _, err := tx.ExecContext(ctx, "INSERT INTO accounts(id,username,type) VALUES(?,?,?) ON CONFLICT(id) DO UPDATE SET username=excluded.username,type=excluded.type", v.ID, v.Username, v.Type); err != nil {
			return err
		}
	}
	for _, v := range seed.Apps {
		b, err := json.Marshal(v)
		if err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO oauth_apps VALUES(?,?) ON CONFLICT(id) DO UPDATE SET data=excluded.data", v.ID, string(b)); err != nil {
			return err
		}
	}
	for _, v := range seed.Assets {
		b, err := json.Marshal(v)
		if err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO oauth_assets VALUES(?,?,?) ON CONFLICT(id) DO UPDATE SET account_id=excluded.account_id,data=excluded.data", v.ID, v.AccountID, string(b)); err != nil {
			return err
		}
	}
	for _, v := range seed.Grants {
		if err := putGrant(ctx, tx, v); err != nil {
			return err
		}
	}
	return nil
}
func putGrant(ctx context.Context, tx *sql.Tx, v store.OAuthGrant) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO oauth_grants VALUES(?,?,?) ON CONFLICT(app_id,account_id) DO UPDATE SET data=excluded.data", v.AppID, v.AccountID, string(b))
	return err
}
func (s *Store) UpsertOAuthFixtures(ctx context.Context, seed store.Seed) error {
	if err := profiles.Validate(seed); err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = upsertFixtures(ctx, tx, seed); err != nil {
		return err
	}
	return tx.Commit()
}
func pruneOAuth(ctx context.Context, tx *sql.Tx) error {
	for _, table := range []string{"oauth_sessions", "oauth_codes"} {
		if _, err := tx.ExecContext(ctx, "DELETE FROM "+table+" WHERE expires_at<=(SELECT now FROM settings WHERE id=1)"); err != nil {
			return err
		}
	}
	return nil
}
func oauthCapacity(ctx context.Context, tx *sql.Tx, table string) error {
	var count int
	if err := tx.QueryRowContext(ctx, "SELECT count(*) FROM "+table).Scan(&count); err != nil {
		return err
	}
	if count >= maxOAuthRecords {
		return store.ErrCapacity
	}
	return nil
}
func insertCode(ctx context.Context, tx *sql.Tx, v store.OAuthCode) error {
	if err := oauthCapacity(ctx, tx, "oauth_codes"); err != nil {
		return err
	}
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO oauth_codes VALUES(?,?,?,?,?)", v.Value, v.AppID, v.AccountID, v.ExpiresAt, string(b))
	return err
}
func (s *Store) PutOAuthSession(ctx context.Context, v store.OAuthSession) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = pruneOAuth(ctx, tx); err != nil {
		return err
	}
	if err = oauthCapacity(ctx, tx, "oauth_sessions"); err != nil {
		return err
	}
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO oauth_sessions VALUES(?,?,?,?)", v.ID, v.AppID, v.ExpiresAt, string(b)); err != nil {
		return err
	}
	return tx.Commit()
}
func (s *Store) GetOAuthSession(ctx context.Context, id string) (v store.OAuthSession, err error) {
	err = decodeRow(s.db.QueryRowContext(ctx, "SELECT data FROM oauth_sessions WHERE id=? AND expires_at>(SELECT now FROM settings WHERE id=1)", id), &v)
	return
}
func (s *Store) DeleteOAuthSession(ctx context.Context, id string) error {
	r, err := s.db.ExecContext(ctx, "DELETE FROM oauth_sessions WHERE id=? AND expires_at>(SELECT now FROM settings WHERE id=1)", id)
	if err != nil {
		return err
	}
	n, err := r.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return store.ErrNotFound
	}
	return nil
}
func (s *Store) PutOAuthCode(ctx context.Context, v store.OAuthCode) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = pruneOAuth(ctx, tx); err != nil {
		return err
	}
	if err = insertCode(ctx, tx, v); err != nil {
		return err
	}
	return tx.Commit()
}
func (s *Store) CompleteOAuthSession(ctx context.Context, id string, v store.OAuthCode) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = pruneOAuth(ctx, tx); err != nil {
		return err
	}
	var session store.OAuthSession
	if err = decodeRow(tx.QueryRowContext(ctx, "DELETE FROM oauth_sessions WHERE id=? RETURNING data", id), &session); err != nil {
		return err
	}
	if v.AppID != session.AppID || v.RedirectURI != session.RedirectURI || v.Challenge != session.Challenge {
		return store.ErrConflict
	}
	for _, scope := range v.Scopes {
		if !slices.Contains(session.Scopes, scope) {
			return store.ErrConflict
		}
	}
	var app store.OAuthApp
	if err = decodeRow(tx.QueryRowContext(ctx, "SELECT data FROM oauth_apps WHERE id=?", v.AppID), &app); err != nil {
		return err
	}
	if !slices.Contains(app.AccountIDs, v.AccountID) {
		return store.ErrConflict
	}
	for _, id := range v.AssetIDs {
		var owner string
		if err = tx.QueryRowContext(ctx, "SELECT account_id FROM oauth_assets WHERE id=?", id).Scan(&owner); err != nil {
			return mapped(err)
		}
		if owner != v.AccountID {
			return store.ErrConflict
		}
	}

	if err = insertCode(ctx, tx, v); err != nil {
		return err
	}
	return tx.Commit()
}
func (s *Store) GetOAuthCode(ctx context.Context, id string) (v store.OAuthCode, err error) {
	err = decodeRow(s.db.QueryRowContext(ctx, "SELECT data FROM oauth_codes WHERE id=? AND expires_at>(SELECT now FROM settings WHERE id=1)", id), &v)
	return
}
func (s *Store) RedeemOAuthCode(ctx context.Context, id string, t store.Token, g store.OAuthGrant) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = pruneOAuth(ctx, tx); err != nil {
		return err
	}
	var code store.OAuthCode
	if err = decodeRow(tx.QueryRowContext(ctx, "DELETE FROM oauth_codes WHERE id=? RETURNING data", id), &code); err != nil {
		return err
	}
	if t.GrantAccountID != "" && t.GrantAccountID != code.AccountID {
		return store.ErrConflict
	}
	if t.AppID != code.AppID || g.AppID != code.AppID || g.AccountID != code.AccountID || !slices.Equal(t.Scopes, code.Scopes) || !slices.Equal(t.AssetIDs, code.AssetIDs) || !slices.Equal(g.Scopes, code.Scopes) || !slices.Equal(g.AssetIDs, code.AssetIDs) {
		return store.ErrConflict
	}
	var app store.OAuthApp
	if err = decodeRow(tx.QueryRowContext(ctx, "SELECT data FROM oauth_apps WHERE id=?", code.AppID), &app); err != nil {
		return err
	}
	if t.Product != app.Product {
		return store.ErrConflict
	}
	if t.AccountID != code.AccountID {
		allowed := false
		if app.Product == "instagram_facebook" {
			for _, id := range code.AssetIDs {
				var asset store.OAuthAsset
				if err = decodeRow(tx.QueryRowContext(ctx, "SELECT data FROM oauth_assets WHERE id=?", id), &asset); err != nil {
					return err
				}
				if asset.AccountID == code.AccountID && asset.InstagramAccountID == t.AccountID {
					allowed = true
				}
			}
		}
		if !allowed {
			return store.ErrConflict
		}
	}
	// Bound sequential authorization growth while preserving legacy fixture tokens.
	if _, err = tx.ExecContext(ctx, "DELETE FROM tokens WHERE app_id<>'' AND expires_at>0 AND expires_at<=(SELECT now FROM settings WHERE id=1) AND (refresh_token='' OR refresh_expires_at<=(SELECT now FROM settings WHERE id=1))"); err != nil {
		return err
	}
	var tokenCount int
	if err = tx.QueryRowContext(ctx, "SELECT count(*) FROM tokens WHERE app_id<>''").Scan(&tokenCount); err != nil {
		return err
	}
	if tokenCount >= maxOAuthRecords {
		return store.ErrCapacity
	}
	scopes, err := json.Marshal(t.Scopes)
	if err != nil {
		return err
	}
	assets, err := json.Marshal(t.AssetIDs)
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO tokens("+tokenColumns+") VALUES(?,?,?,?,?,?,?,?,?,?,?,?)", t.Value, t.AccountID, string(scopes), t.ExpiresAt, t.AppID, t.Product, string(assets), code.AccountID, t.RefreshToken, t.RefreshExpiresAt, t.Kind, t.IssuedAt); err != nil {
		return err
	}
	if err = putGrant(ctx, tx, g); err != nil {
		return err
	}
	return tx.Commit()
}

// OAuthRefreshToken resolves only an unexpired refresh credential.
func (s *Store) OAuthRefreshToken(ctx context.Context, refresh string) (store.Token, error) {
	if refresh == "" {
		return store.Token{}, store.ErrNotFound
	}
	return scanToken(s.db.QueryRowContext(ctx, "SELECT "+tokenColumns+" FROM tokens WHERE refresh_token=? AND refresh_expires_at>(SELECT now FROM settings WHERE id=1)", refresh))
}

// RefreshOAuthToken rotates both credentials atomically. The simulator invalidates
// the old access token immediately rather than simulating an overlap window.
func (s *Store) RefreshOAuthToken(ctx context.Context, refresh string, t store.Token) error {
	if refresh == "" || t.RefreshToken == "" || t.Value == "" {
		return store.ErrConflict
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	old, err := scanToken(tx.QueryRowContext(ctx, "SELECT "+tokenColumns+" FROM tokens WHERE refresh_token=? AND refresh_expires_at>(SELECT now FROM settings WHERE id=1)", refresh))
	if err != nil {
		return err
	}
	if old.Product != "tiktok" || old.AppID != t.AppID || old.Product != t.Product || old.AccountID != t.AccountID || old.GrantAccountID != t.GrantAccountID || !slices.Equal(old.Scopes, t.Scopes) || !slices.Equal(old.AssetIDs, t.AssetIDs) || t.RefreshToken == refresh || t.Value == old.Value {
		return store.ErrConflict
	}
	var now int64
	if err = tx.QueryRowContext(ctx, "SELECT now FROM settings WHERE id=1").Scan(&now); err != nil {
		return err
	}
	if t.ExpiresAt <= now || t.RefreshExpiresAt <= now {
		return store.ErrConflict
	}
	if _, err = tx.ExecContext(ctx, "DELETE FROM tokens WHERE value=?", old.Value); err != nil {
		return err
	}
	scopes, err := json.Marshal(t.Scopes)
	if err != nil {
		return err
	}
	assets, err := json.Marshal(t.AssetIDs)
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO tokens("+tokenColumns+") VALUES(?,?,?,?,?,?,?,?,?,?,?,?)", t.Value, t.AccountID, string(scopes), t.ExpiresAt, t.AppID, t.Product, string(assets), t.GrantAccountID, t.RefreshToken, t.RefreshExpiresAt, t.Kind, t.IssuedAt); err != nil {
		return err
	}
	return tx.Commit()
}

// ExchangeInstagramToken stores the long-lived token issued for a short-lived
// Instagram Login token. Repeating the exchange for the same short-lived token
// replaces the same long-lived row, so repeated calls cannot grow state.
func (s *Store) ExchangeInstagramToken(ctx context.Context, short string, t store.Token) error {
	if short == "" || t.Value == "" || t.Value == short || t.Kind != "long_lived" || t.Product != "instagram" {
		return store.ErrConflict
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	old, err := scanToken(tx.QueryRowContext(ctx, "SELECT "+tokenColumns+" FROM tokens WHERE value=? AND kind='' AND product='instagram' AND app_id<>'' AND expires_at>(SELECT now FROM settings WHERE id=1)", short))
	if err != nil {
		return err
	}
	if old.AppID != t.AppID || old.AccountID != t.AccountID || old.GrantAccountID != t.GrantAccountID || !slices.Equal(old.Scopes, t.Scopes) || !slices.Equal(old.AssetIDs, t.AssetIDs) {
		return store.ErrConflict
	}
	var exists bool
	if err = tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM tokens WHERE value=?)", t.Value).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		var tokenCount int
		if err = tx.QueryRowContext(ctx, "SELECT count(*) FROM tokens WHERE app_id<>''").Scan(&tokenCount); err != nil {
			return err
		}
		if tokenCount >= maxOAuthRecords {
			return store.ErrCapacity
		}
	}
	scopes, err := json.Marshal(t.Scopes)
	if err != nil {
		return err
	}
	assets, err := json.Marshal(t.AssetIDs)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, "INSERT INTO tokens("+tokenColumns+") VALUES(?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(value) DO UPDATE SET expires_at=excluded.expires_at, issued_at=excluded.issued_at WHERE tokens.kind='long_lived' AND tokens.app_id=excluded.app_id AND tokens.account_id=excluded.account_id", t.Value, t.AccountID, string(scopes), t.ExpiresAt, t.AppID, t.Product, string(assets), t.GrantAccountID, "", 0, t.Kind, t.IssuedAt)
	if err != nil {
		return err
	}
	if n, err := result.RowsAffected(); err != nil || n != 1 {
		return store.ErrConflict
	}
	return tx.Commit()
}

// ExtendInstagramToken refreshes an unexpired long-lived Instagram Login token
// in place. Keeping the token value is a simulator choice; see docs/oauth-consent.md.
func (s *Store) ExtendInstagramToken(ctx context.Context, value string, expiresAt, issuedAt int64) error {
	result, err := s.db.ExecContext(ctx, "UPDATE tokens SET expires_at=?, issued_at=? WHERE value=? AND kind='long_lived' AND product='instagram' AND expires_at>(SELECT now FROM settings WHERE id=1) AND ?>expires_at", expiresAt, issuedAt, value, expiresAt)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return store.ErrConflict
	}
	return nil
}
