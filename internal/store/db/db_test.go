package db

import (
	"path/filepath"
	"testing"
)

func TestMustInitializesDatabase(t *testing.T) {
	connection := Must(":memory:")
	defer connection.Close()
	if err := connection.Ping(); err != nil {
		t.Fatal(err)
	}
	var version, foreignKeys, initialized int
	for query, dest := range map[string]*int{
		"PRAGMA user_version":                         &version,
		"PRAGMA foreign_keys":                         &foreignKeys,
		"SELECT initialized FROM settings WHERE id=1": &initialized,
	} {
		if err := connection.QueryRow(query).Scan(dest); err != nil {
			t.Fatal(err)
		}
	}
	if version != 2 || foreignKeys != 1 || initialized != 0 {
		t.Fatalf("database not initialized: version=%d foreign_keys=%d initialized=%d", version, foreignKeys, initialized)
	}
	if got := connection.Stats().MaxOpenConnections; got != 1 {
		t.Fatalf("max open connections = %d, want 1", got)
	}
}

func TestMustPanicsOnInitializationFailure(t *testing.T) {
	futureSchema := filepath.Join(t.TempDir(), "future.db")
	connection := Must(futureSchema)
	if _, err := connection.Exec("PRAGMA user_version=3"); err != nil {
		t.Fatal(err)
	}
	if err := connection.Close(); err != nil {
		t.Fatal(err)
	}
	for name, path := range map[string]string{
		"connection":                filepath.Join(t.TempDir(), "missing", "database.db"),
		"driver option with secret": "file::memory:?_journal_mode=secret-do-not-log",
		"migration":                 futureSchema,
	} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if got := recover(); got != "db: initialization failed" {
					t.Fatalf("panic = %v, want sanitized initialization failure", got)
				}
			}()
			connection := Must(path)
			connection.Close()
			t.Fatal("Must did not panic")
		})
	}
}

func TestOpenReturnsInitializationError(t *testing.T) {
	connection, err := Open(filepath.Join(t.TempDir(), "missing", "database.db"))
	if err == nil || connection != nil {
		t.Fatalf("Open = (%v, %v), want (nil, error)", connection, err)
	}
}
