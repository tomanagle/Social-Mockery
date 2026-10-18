// Package db owns SQLite connections and versioned schema migrations.
package db

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
)

// Must opens and initializes a database, panicking if either step fails.
// The panic deliberately omits driver errors, which may contain DSN secrets.
// Use Open when the caller needs to handle initialization errors.
func Must(path string) *sql.DB {
	connection, err := Open(path)
	if err != nil {
		panic("db: initialization failed")
	}
	return connection
}

// Open opens a database, configures its connection, and applies migrations.
func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	// One connection keeps :memory: consistent and serializes SQLite writes.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if _, err = db.Exec(`PRAGMA foreign_keys=ON; PRAGMA busy_timeout=5000;`); err != nil {
		db.Close()
		return nil, err
	}
	if err = migrate(db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}
func migrate(db *sql.DB) error {
	var version int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		return err
	}
	if version > 2 {
		return fmt.Errorf("database schema %d is newer than supported version 2", version)
	}
	if version == 2 {
		return nil
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if version == 0 {
		_, err = tx.Exec(`
 CREATE TABLE settings (id INTEGER PRIMARY KEY CHECK(id=1), now INTEGER NOT NULL, processing_seconds INTEGER NOT NULL, sequence INTEGER NOT NULL, initialized INTEGER NOT NULL);
 INSERT INTO settings VALUES(1,1767225600,5,0,0);
 CREATE TABLE accounts (id TEXT PRIMARY KEY, username TEXT NOT NULL, type TEXT NOT NULL);
 CREATE TABLE tokens (value TEXT PRIMARY KEY, account_id TEXT NOT NULL REFERENCES accounts(id), scopes TEXT NOT NULL, expires_at INTEGER NOT NULL);
 CREATE TABLE containers (id TEXT PRIMARY KEY, account_id TEXT NOT NULL REFERENCES accounts(id), media_type TEXT NOT NULL, url TEXT NOT NULL, caption TEXT NOT NULL, created_at INTEGER NOT NULL, ready_at INTEGER NOT NULL, published_id TEXT NOT NULL DEFAULT '');
 CREATE TABLE posts (id TEXT PRIMARY KEY, account_id TEXT NOT NULL REFERENCES accounts(id), media_type TEXT NOT NULL, url TEXT NOT NULL, caption TEXT NOT NULL, created_at INTEGER NOT NULL);
 CREATE INDEX posts_account ON posts(account_id,id);
 PRAGMA user_version=1;
 `)
		if err != nil {
			return err
		}
	}
	if _, err = tx.Exec(`
 ALTER TABLE tokens ADD COLUMN refresh_token TEXT NOT NULL DEFAULT '';
 ALTER TABLE tokens ADD COLUMN refresh_expires_at INTEGER NOT NULL DEFAULT 0;
 CREATE UNIQUE INDEX tokens_refresh_unique ON tokens(refresh_token) WHERE refresh_token<>'';
 ALTER TABLE tokens ADD COLUMN grant_account_id TEXT NOT NULL DEFAULT '';
 ALTER TABLE tokens ADD COLUMN app_id TEXT NOT NULL DEFAULT '';
 ALTER TABLE tokens ADD COLUMN product TEXT NOT NULL DEFAULT '';
 ALTER TABLE tokens ADD COLUMN asset_ids TEXT NOT NULL DEFAULT '[]';
 CREATE TABLE oauth_apps (id TEXT PRIMARY KEY, data TEXT NOT NULL);
 CREATE TABLE oauth_assets (id TEXT PRIMARY KEY, account_id TEXT NOT NULL REFERENCES accounts(id), data TEXT NOT NULL);
 CREATE TABLE oauth_grants (app_id TEXT NOT NULL REFERENCES oauth_apps(id), account_id TEXT NOT NULL REFERENCES accounts(id), data TEXT NOT NULL, PRIMARY KEY(app_id,account_id));
 CREATE TABLE oauth_sessions (id TEXT PRIMARY KEY, app_id TEXT NOT NULL REFERENCES oauth_apps(id), expires_at INTEGER NOT NULL, data TEXT NOT NULL);
 CREATE TABLE oauth_codes (id TEXT PRIMARY KEY, app_id TEXT NOT NULL REFERENCES oauth_apps(id), account_id TEXT NOT NULL REFERENCES accounts(id), expires_at INTEGER NOT NULL, data TEXT NOT NULL);
 CREATE INDEX oauth_sessions_expiry ON oauth_sessions(expires_at);
 CREATE INDEX oauth_codes_expiry ON oauth_codes(expires_at);
 PRAGMA user_version=2;
 `); err != nil {
		return err
	}
	return tx.Commit()
}
