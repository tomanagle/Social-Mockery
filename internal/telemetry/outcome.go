package telemetry

import (
	"context"
	"sync/atomic"
)

type outcomeKey struct{}
type outcome struct{ expected atomic.Bool }

func withOutcome(ctx context.Context) context.Context {
	return context.WithValue(ctx, outcomeKey{}, &outcome{})
}

// ExpectedFailure marks a synthetic or native provider rejection, including an
// intentionally injected 5xx, without hiding genuine internal failures.
func ExpectedFailure(ctx context.Context) {
	if o, ok := ctx.Value(outcomeKey{}).(*outcome); ok {
		o.expected.Store(true)
	}
}
func expectedFailure(ctx context.Context) bool {
	o, ok := ctx.Value(outcomeKey{}).(*outcome)
	return ok && o.expected.Load()
}
