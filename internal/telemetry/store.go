package telemetry

import (
	"context"
	"errors"
	"github.com/tomanagle/Social-Mockery/internal/store"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"time"
)

type observedStore struct {
	store.Repository
	t *Telemetry
}

// ObserveStore records operations, never statements, parameters, IDs or rows.
func ObserveStore(repo store.Repository, t *Telemetry) store.Repository {
	if !t.enabled {
		return repo
	}
	return &observedStore{repo, t}
}
func (s *observedStore) finish(ctx context.Context, op string, started time.Time, err error) {
	outcome := "success"
	if err != nil {
		outcome = "error"
	}
	if s.t.logger != nil {
		s.t.logger.DebugContext(ctx, "database operation completed", "operation", op, "outcome", outcome, "duration_seconds", time.Since(started).Seconds())
	}
	s.t.dbDuration.Record(ctx, time.Since(started).Seconds(), metric.WithAttributes(attribute.String("db.operation.name", op), attribute.String("outcome", outcome)))
}
func expectedStoreError(err error) bool {
	return errors.Is(err, store.ErrNotFound) || errors.Is(err, store.ErrConflict) || errors.Is(err, store.ErrCapacity) || errors.Is(err, context.Canceled)
}
func (s *observedStore) Reset(ctx context.Context, seed store.Seed, now int64) (err error) {
	ctx, span := s.t.Start(ctx, "db.reset")
	started := time.Now()
	defer func() { s.finish(ctx, "reset", started, err); Finish(span, err, expectedStoreError(err)) }()
	err = s.Repository.Reset(ctx, seed, now)
	if err == nil {
		s.state(ctx)
	}
	return err
}
func (s *observedStore) Clock(ctx context.Context) (now int64, delay int64, err error) {
	ctx, span := s.t.Start(ctx, "db.clock")
	started := time.Now()
	defer func() { s.finish(ctx, "clock", started, err); Finish(span, err, expectedStoreError(err)) }()
	return s.Repository.Clock(ctx)
}
func (s *observedStore) Advance(ctx context.Context, seconds int64) (now int64, err error) {
	ctx, span := s.t.Start(ctx, "db.advance")
	started := time.Now()
	defer func() { s.finish(ctx, "advance", started, err); Finish(span, err, expectedStoreError(err)) }()
	return s.Repository.Advance(ctx, seconds)
}
func (s *observedStore) Account(ctx context.Context, id string) (a store.Account, err error) {
	ctx, span := s.t.Start(ctx, "db.account")
	started := time.Now()
	defer func() { s.finish(ctx, "account", started, err); Finish(span, err, expectedStoreError(err)) }()
	return s.Repository.Account(ctx, id)
}
func (s *observedStore) Token(ctx context.Context, value string) (token store.Token, err error) {
	ctx, span := s.t.Start(ctx, "db.token")
	started := time.Now()
	defer func() { s.finish(ctx, "token", started, err); Finish(span, err, expectedStoreError(err)) }()
	return s.Repository.Token(ctx, value)
}
func (s *observedStore) CreateContainer(ctx context.Context, c store.Container, max int) (result store.Container, err error) {
	ctx, span := s.t.Start(ctx, "db.createcontainer")
	started := time.Now()
	defer func() { s.finish(ctx, "createcontainer", started, err); Finish(span, err, expectedStoreError(err)) }()
	return s.Repository.CreateContainer(ctx, c, max)
}
func (s *observedStore) Container(ctx context.Context, id string) (c store.Container, err error) {
	ctx, span := s.t.Start(ctx, "db.container")
	started := time.Now()
	defer func() { s.finish(ctx, "container", started, err); Finish(span, err, expectedStoreError(err)) }()
	return s.Repository.Container(ctx, id)
}
func (s *observedStore) Publish(ctx context.Context, account string, id string, max int) (post store.Post, err error) {
	ctx, span := s.t.Start(ctx, "db.publish")
	started := time.Now()
	defer func() { s.finish(ctx, "publish", started, err); Finish(span, err, expectedStoreError(err)) }()
	post, err = s.Repository.Publish(ctx, account, id, max)
	if err == nil {
		s.state(ctx)
	}
	return post, err
}
func (s *observedStore) Post(ctx context.Context, id string) (post store.Post, err error) {
	ctx, span := s.t.Start(ctx, "db.post")
	started := time.Now()
	defer func() { s.finish(ctx, "post", started, err); Finish(span, err, expectedStoreError(err)) }()
	return s.Repository.Post(ctx, id)
}
func (s *observedStore) Posts(ctx context.Context, account string, limit int, offset int) (posts []store.Post, err error) {
	ctx, span := s.t.Start(ctx, "db.posts")
	started := time.Now()
	defer func() { s.finish(ctx, "posts", started, err); Finish(span, err, expectedStoreError(err)) }()
	return s.Repository.Posts(ctx, account, limit, offset)
}

func (s *observedStore) state(ctx context.Context) {
	if counts, ok := s.Repository.(interface {
		Counts(context.Context) (int64, int64, error)
	}); ok {
		a, p, err := counts.Counts(ctx)
		if err == nil {
			s.t.State(ctx, a, p)
		}
	}
}
