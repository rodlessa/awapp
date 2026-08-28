package weather

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNewPollerDefaultsInterval(t *testing.T) {
	p := NewPoller(nil, 0)
	if p.interval != 5*time.Minute {
		t.Fatalf("non-positive interval should default to 5m, got %v", p.interval)
	}
}

type errFetcher struct{ err error }

func (f errFetcher) Fetch(ctx context.Context) (Report, error) { return Report{}, f.err }

// Errors must drain-then-send like Reports, so the consumer always sees
// the latest failure reason rather than a stale one.
func TestPollerErrorsLatestWins(t *testing.T) {
	p := NewPoller(errFetcher{err: errors.New("first")}, time.Hour)
	p.tick(context.Background())
	p.fetcher = errFetcher{err: errors.New("second")}
	p.tick(context.Background())
	if got := <-p.Errors; got == nil || got.Error() != "second" {
		t.Fatalf("expected the latest error to win, got %v", got)
	}
}
