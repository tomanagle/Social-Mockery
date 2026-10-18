package dbstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/tomanagle/Social-Mockery/internal/profiles"
	"github.com/tomanagle/Social-Mockery/internal/store"
	"math"
	"slices"
)

type Store struct{ db *sql.DB }

func New(db *sql.DB) *Store { return &Store{db: db} }
func mapped(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return store.ErrNotFound
	}
	return err
}
func (s *Store) Reset(ctx context.Context, seed store.Seed, now int64) error {
	if err := profiles.Validate(seed); err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, table := range []string{"oauth_codes", "oauth_sessions", "oauth_grants", "oauth_assets", "oauth_apps", "posts", "containers", "tokens", "accounts"} {
		if _, err = tx.ExecContext(ctx, "DELETE FROM "+table); err != nil {
			return err
		}
	}
	for _, a := range seed.Accounts {
		if _, err = tx.ExecContext(ctx, `INSERT INTO accounts VALUES(?,?,?)`, a.ID, a.Username, a.Type); err != nil {
			return err
		}
	}
	for _, t := range seed.Tokens {
		if t.AppID != "" && t.GrantAccountID == "" {
			t.GrantAccountID = t.AccountID
			for _, app := range seed.Apps {
				if app.ID != t.AppID || slices.Contains(app.AccountIDs, t.AccountID) {
					continue
				}
				for _, asset := range seed.Assets {
					if slices.Contains(t.AssetIDs, asset.ID) && asset.InstagramAccountID == t.AccountID && slices.Contains(app.AccountIDs, asset.AccountID) {
						t.GrantAccountID = asset.AccountID
						break
					}
				}
			}
		}
		b, e := json.Marshal(t.Scopes)
		if e != nil {
			return e
		}
		assets, e := json.Marshal(t.AssetIDs)
		if e != nil {
			return e
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO tokens(value,account_id,scopes,expires_at,app_id,product,asset_ids,grant_account_id,refresh_token,refresh_expires_at) VALUES(?,?,?,?,?,?,?,?,?,?)`, t.Value, t.AccountID, string(b), t.ExpiresAt, t.AppID, t.Product, string(assets), t.GrantAccountID, t.RefreshToken, t.RefreshExpiresAt); err != nil {
			return err
		}
	}
	if err = upsertFixtures(ctx, tx, seed); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE settings SET now=?,processing_seconds=?,sequence=0,initialized=1 WHERE id=1`, now, seed.ProcessingSeconds); err != nil {
		return err
	}
	return tx.Commit()
}
func (s *Store) Clock(ctx context.Context) (int64, int64, error) {
	var now, delay int64
	err := s.db.QueryRowContext(ctx, `SELECT now,processing_seconds FROM settings WHERE id=1`).Scan(&now, &delay)
	return now, delay, err
}
func (s *Store) Advance(ctx context.Context, seconds int64) (int64, error) {
	if seconds < 0 || seconds > 31536000 {
		return 0, fmt.Errorf("seconds must be 0..31536000")
	}
	var now int64
	err := s.db.QueryRowContext(ctx, `UPDATE settings SET now=now+? WHERE id=1 AND now<=? RETURNING now`, seconds, math.MaxInt64-seconds).Scan(&now)
	return now, err
}
func (s *Store) Account(ctx context.Context, id string) (store.Account, error) {
	var a store.Account
	err := s.db.QueryRowContext(ctx, `SELECT id,username,type FROM accounts WHERE id=?`, id).Scan(&a.ID, &a.Username, &a.Type)
	return a, mapped(err)
}

const tokenColumns = "value,account_id,scopes,expires_at,app_id,product,asset_ids,grant_account_id,refresh_token,refresh_expires_at"

func scanToken(row *sql.Row) (store.Token, error) {
	var t store.Token
	var scopes, assets string
	err := row.Scan(&t.Value, &t.AccountID, &scopes, &t.ExpiresAt, &t.AppID, &t.Product, &assets, &t.GrantAccountID, &t.RefreshToken, &t.RefreshExpiresAt)
	if err != nil {
		return t, mapped(err)
	}
	if err = json.Unmarshal([]byte(assets), &t.AssetIDs); err != nil {
		return t, err
	}
	err = json.Unmarshal([]byte(scopes), &t.Scopes)
	return t, err
}
func (s *Store) Token(ctx context.Context, value string) (store.Token, error) {
	return scanToken(s.db.QueryRowContext(ctx, "SELECT "+tokenColumns+" FROM tokens WHERE value=?", value))
}

func nextID(ctx context.Context, tx *sql.Tx) (string, error) {
	for {
		var n int64
		if err := tx.QueryRowContext(ctx, `UPDATE settings SET sequence=sequence+1 WHERE id=1 RETURNING sequence`).Scan(&n); err != nil {
			return "", err
		}
		id := fmt.Sprintf("18%015d", n)
		var exists bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM accounts WHERE id=?)`, id).Scan(&exists); err != nil {
			return "", err
		}
		if !exists {
			return id, nil
		}
	}
}
func (s *Store) Initialized(ctx context.Context) (bool, error) {
	var initialized bool
	err := s.db.QueryRowContext(ctx, `SELECT initialized FROM settings WHERE id=1`).Scan(&initialized)
	return initialized, err
}

func capacity(ctx context.Context, tx *sql.Tx, max int) error {
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT (SELECT count(*) FROM containers)+(SELECT count(*) FROM posts)`).Scan(&count); err != nil {
		return err
	}
	if count >= max {
		return store.ErrCapacity
	}
	return nil
}
func (s *Store) CreateContainer(ctx context.Context, c store.Container, max int) (store.Container, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return c, err
	}
	defer tx.Rollback()
	if err = capacity(ctx, tx, max); err != nil {
		return c, err
	}
	if c.ID, err = nextID(ctx, tx); err != nil {
		return c, err
	}
	var delay int64
	if err = tx.QueryRowContext(ctx, `SELECT now,processing_seconds FROM settings WHERE id=1`).Scan(&c.CreatedAt, &delay); err != nil {
		return c, err
	}
	c.ReadyAt = c.CreatedAt + delay
	_, err = tx.ExecContext(ctx, `INSERT INTO containers VALUES(?,?,?,?,?,?,?,?)`, c.ID, c.AccountID, c.MediaType, c.URL, c.Caption, c.CreatedAt, c.ReadyAt, "")
	if err != nil {
		return c, err
	}
	return c, tx.Commit()
}
func scanContainer(row *sql.Row) (store.Container, error) {
	var c store.Container
	err := row.Scan(&c.ID, &c.AccountID, &c.MediaType, &c.URL, &c.Caption, &c.CreatedAt, &c.ReadyAt, &c.PublishedID)
	return c, mapped(err)
}
func (s *Store) Container(ctx context.Context, id string) (store.Container, error) {
	return scanContainer(s.db.QueryRowContext(ctx, `SELECT * FROM containers WHERE id=?`, id))
}
func (s *Store) Publish(ctx context.Context, account, id string, max int) (store.Post, error) {
	var p store.Post
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return p, err
	}
	defer tx.Rollback()
	c, err := scanContainer(tx.QueryRowContext(ctx, `SELECT * FROM containers WHERE id=? AND account_id=?`, id, account))
	if err != nil {
		return p, err
	}
	var now int64
	if err = tx.QueryRowContext(ctx, `SELECT now FROM settings WHERE id=1`).Scan(&now); err != nil {
		return p, err
	}
	if c.PublishedID != "" || now < c.ReadyAt || now >= c.CreatedAt+86400 {
		return p, store.ErrConflict
	}
	if err = capacity(ctx, tx, max); err != nil {
		return p, err
	}
	p = store.Post{AccountID: account, MediaType: c.MediaType, URL: c.URL, Caption: c.Caption, CreatedAt: now}
	if p.ID, err = nextID(ctx, tx); err != nil {
		return p, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO posts VALUES(?,?,?,?,?,?)`, p.ID, p.AccountID, p.MediaType, p.URL, p.Caption, p.CreatedAt); err != nil {
		return p, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE containers SET published_id=? WHERE id=?`, p.ID, id); err != nil {
		return p, err
	}
	return p, tx.Commit()
}
func (s *Store) Post(ctx context.Context, id string) (store.Post, error) {
	var p store.Post
	err := s.db.QueryRowContext(ctx, `SELECT * FROM posts WHERE id=?`, id).Scan(&p.ID, &p.AccountID, &p.MediaType, &p.URL, &p.Caption, &p.CreatedAt)
	return p, mapped(err)
}
func (s *Store) Posts(ctx context.Context, account string, limit, offset int) ([]store.Post, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT * FROM posts WHERE account_id=? ORDER BY id LIMIT ? OFFSET ?`, account, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []store.Post{}
	for rows.Next() {
		var p store.Post
		if err = rows.Scan(&p.ID, &p.AccountID, &p.MediaType, &p.URL, &p.Caption, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// Counts provides aggregate state telemetry without materializing stored rows.
func (s *Store) Counts(ctx context.Context) (accounts, posts int64, err error) {
	err = s.db.QueryRowContext(ctx, `SELECT (SELECT COUNT(*) FROM accounts), (SELECT COUNT(*) FROM posts)`).Scan(&accounts, &posts)
	return
}
