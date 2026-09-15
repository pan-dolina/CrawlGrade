package crawler

import (
	"context"
	"sync"
	"time"
)

// Limiter spaces requests evenly at a fixed rate. It is safe for concurrent
// use. A nil *Limiter or a non-positive rate does not limit.
type Limiter struct {
	mu       sync.Mutex
	interval time.Duration
	next     time.Time
	now      func() time.Time
}

// NewLimiter returns a limiter allowing perSecond requests per second.
func NewLimiter(perSecond float64) *Limiter {
	if perSecond <= 0 {
		return nil
	}
	return &Limiter{interval: time.Duration(float64(time.Second) / perSecond), now: time.Now}
}

// Wait blocks until the next request may start or ctx is done.
func (l *Limiter) Wait(ctx context.Context) error {
	if l == nil {
		return ctx.Err()
	}
	l.mu.Lock()
	now := l.now()
	t := l.next
	if t.Before(now) {
		t = now
	}
	l.next = t.Add(l.interval)
	l.mu.Unlock()

	d := t.Sub(now)
	if d <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
