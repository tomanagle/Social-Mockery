// Package simulation centralizes the clock and repeatable seed lifecycle.
package simulation

import (
	"context"
	"github.com/tomanagle/Social-Mockery/internal/profiles"
	"github.com/tomanagle/Social-Mockery/internal/store"
	"sync"
	"time"
)

// Engine serializes complete HTTP operations against reset/clock changes. SQLite
// transactions additionally protect atomic publishing and persisted state.
type Engine struct {
	Mu         sync.Mutex
	Store      store.Repository
	Seed       store.Seed
	MaxRecords int
}

func New(repo store.Repository, seed store.Seed, max int) *Engine {
	return &Engine{Store: repo, Seed: seed, MaxRecords: max}
}
func (e *Engine) Reset(ctx context.Context, seed store.Seed) error {
	if err := profiles.Validate(seed); err != nil {
		return err
	}
	clock, _ := time.Parse(time.RFC3339, seed.Clock)
	return e.Store.Reset(ctx, seed, clock.Unix())
}
func Status(c store.Container, now int64) string {
	if c.PublishedID != "" {
		return "PUBLISHED"
	}
	if now >= c.CreatedAt+86400 {
		return "EXPIRED"
	}
	if now >= c.ReadyAt {
		return "FINISHED"
	}
	return "IN_PROGRESS"
}
