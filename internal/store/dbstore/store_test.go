package dbstore_test

import (
	"context"
	"errors"
	"github.com/tomanagle/Social-Mockery/internal/profiles"
	"github.com/tomanagle/Social-Mockery/internal/store"
	"github.com/tomanagle/Social-Mockery/internal/store/db"
	"github.com/tomanagle/Social-Mockery/internal/store/dbstore"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
)

var ctx = context.Background()

func TestPersistenceResetAndPublish(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.db")
	conn, err := db.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	repo := dbstore.New(conn)
	seed := profiles.Default()
	if err = repo.Reset(ctx, seed, 1767225600); err != nil {
		t.Fatal(err)
	}
	c, err := repo.CreateContainer(ctx, store.Container{AccountID: seed.Accounts[0].ID, MediaType: "IMAGE", URL: "https://example.test/image.jpg", Caption: "fixture"}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = repo.Publish(ctx, c.AccountID, c.ID, 10); !errors.Is(err, store.ErrConflict) {
		t.Fatalf("premature publish: %v", err)
	}
	if _, err = repo.Advance(ctx, 5); err != nil {
		t.Fatal(err)
	}
	p, err := repo.Publish(ctx, c.AccountID, c.ID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = repo.Publish(ctx, c.AccountID, c.ID, 10); !errors.Is(err, store.ErrConflict) {
		t.Fatalf("repeat publish: %v", err)
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
	got, err := repo.Post(ctx, p.ID)
	if err != nil || got != p {
		t.Fatalf("reopen post=%+v err=%v", got, err)
	}
	now, _, err := repo.Clock(ctx)
	if err != nil || now != 1767225605 {
		t.Fatalf("persisted clock %d %v", now, err)
	}
	if err = repo.Reset(ctx, seed, 1767225600); err != nil {
		t.Fatal(err)
	}
	if _, err = repo.Post(ctx, p.ID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("reset retained post: %v", err)
	}
	again, err := repo.CreateContainer(ctx, store.Container{AccountID: c.AccountID, MediaType: "IMAGE", URL: c.URL}, 10)
	if err != nil || again.ID != c.ID {
		t.Fatalf("reset ID %s != %s: %v", again.ID, c.ID, err)
	}
}
func TestCapacityAndConcurrentPublish(t *testing.T) {
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	repo := dbstore.New(conn)
	seed := profiles.Default()
	if err = repo.Reset(ctx, seed, 1767225600); err != nil {
		t.Fatal(err)
	}
	c, err := repo.CreateContainer(ctx, store.Container{AccountID: seed.Accounts[0].ID, MediaType: "REELS", URL: "https://example.test/video.mp4"}, 2)
	if err != nil {
		t.Fatal(err)
	}
	repo.Advance(ctx, 5)
	var wins atomic.Int32
	var wg sync.WaitGroup
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := repo.Publish(ctx, c.AccountID, c.ID, 2)
			if err == nil {
				wins.Add(1)
			} else if !errors.Is(err, store.ErrConflict) {
				t.Errorf("publish error: %v", err)
			}
		}()
	}
	wg.Wait()
	if wins.Load() != 1 {
		t.Fatalf("publish winners=%d", wins.Load())
	}
	if _, err = repo.CreateContainer(ctx, store.Container{AccountID: c.AccountID}, 2); !errors.Is(err, store.ErrCapacity) {
		t.Fatalf("capacity error=%v", err)
	}
	posts, err := repo.Posts(ctx, c.AccountID, 100, 0)
	if err != nil || len(posts) != 1 {
		t.Fatalf("posts=%v error=%v", posts, err)
	}
	if _, err = repo.Advance(ctx, -1); err == nil {
		t.Fatal("negative advance accepted")
	}
}

func TestGeneratedIDsSkipAccountIDs(t *testing.T) {
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	repo := dbstore.New(conn)
	seed := profiles.Default()
	seed.Accounts[0].ID = "18000000000000001"
	seed.Tokens[0].AccountID = seed.Accounts[0].ID
	if err = repo.Reset(ctx, seed, 1767225600); err != nil {
		t.Fatal(err)
	}
	c, err := repo.CreateContainer(ctx, store.Container{AccountID: seed.Accounts[0].ID, MediaType: "IMAGE", URL: "https://example.test/i.jpg"}, 100)
	if err != nil {
		t.Fatal(err)
	}
	if c.ID == seed.Accounts[0].ID {
		t.Fatal("generated container ID collides with account ID")
	}
}
func TestInitializedPersistsEmptySeed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.db")
	conn, err := db.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	repo := dbstore.New(conn)
	initialized, err := repo.Initialized(ctx)
	if err != nil || initialized {
		t.Fatalf("fresh initialized=%v err=%v", initialized, err)
	}
	seed := profiles.Default()
	seed.Accounts = nil
	seed.Tokens = nil
	if err = repo.Reset(ctx, seed, 1767225600); err != nil {
		t.Fatal(err)
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
	initialized, err = repo.Initialized(ctx)
	if err != nil || !initialized {
		t.Fatalf("reopened initialized=%v err=%v", initialized, err)
	}
	if _, err = repo.Account(ctx, "17841400000000001"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("empty seed unexpectedly populated: %v", err)
	}
}
