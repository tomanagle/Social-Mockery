package db

import (
	"database/sql"
	"path/filepath"
	"testing"
)

func TestMigrationV1PreservesExistingState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.db")
	conn, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = conn.Exec(`
 CREATE TABLE settings(id INTEGER PRIMARY KEY,now INTEGER,processing_seconds INTEGER,sequence INTEGER,initialized INTEGER);
 INSERT INTO settings VALUES(1,123,5,10,1);
 CREATE TABLE accounts(id TEXT PRIMARY KEY,username TEXT,type TEXT);
 INSERT INTO accounts VALUES('1','demo','BUSINESS');
 CREATE TABLE tokens(value TEXT PRIMARY KEY,account_id TEXT,scopes TEXT,expires_at INTEGER);
 INSERT INTO tokens VALUES('existing','1','[]',0);
 CREATE TABLE posts(id TEXT PRIMARY KEY,account_id TEXT,media_type TEXT,url TEXT,caption TEXT,created_at INTEGER);
 INSERT INTO posts VALUES('post','1','IMAGE','https://example.test/a.jpg','keep',123);
 PRAGMA user_version=1;`)
	if err != nil {
		t.Fatal(err)
	}
	if err = conn.Close(); err != nil {
		t.Fatal(err)
	}
	conn, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	var caption, app string
	if err = conn.QueryRow("SELECT caption FROM posts WHERE id='post'").Scan(&caption); err != nil || caption != "keep" {
		t.Fatalf("post lost: %s %v", caption, err)
	}
	if err = conn.QueryRow("SELECT app_id FROM tokens WHERE value='existing'").Scan(&app); err != nil || app != "" {
		t.Fatalf("token lost: %s %v", app, err)
	}
	var now, version int
	if err = conn.QueryRow("SELECT now FROM settings").Scan(&now); err != nil || now != 123 {
		t.Fatalf("clock lost: %d %v", now, err)
	}
	if err = conn.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != 2 {
		t.Fatalf("schema version: %d %v", version, err)
	}
}
