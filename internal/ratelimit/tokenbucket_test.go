package ratelimit

import (
	"sync"
	"testing"
	"time"
)

// fakeClock lets tests move time forward without sleeping.
type fakeClock struct{ t time.Time }

func (c *fakeClock) Now() time.Time          { return c.t }
func (c *fakeClock) Advance(d time.Duration) { c.t = c.t.Add(d) }

func newFakeClock() *fakeClock { return &fakeClock{t: time.Unix(0, 0)} }

func TestAllowsUpToBurst(t *testing.T) {
	clock := newFakeClock()
	l := New(1, 3, clock.Now) // 1 req/s, burst of 3

	for i := 0; i < 3; i++ {
		if d := l.Allow("client"); !d.Allowed {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}
	if d := l.Allow("client"); d.Allowed {
		t.Fatal("4th request should be blocked: burst is 3")
	}
}

func TestRefillsOverTime(t *testing.T) {
	clock := newFakeClock()
	l := New(2, 1, clock.Now) // 2 tokens per second, burst of 1

	l.Allow("client") // spends the only token
	if d := l.Allow("client"); d.Allowed {
		t.Fatal("should be blocked right after spending the token")
	}

	clock.Advance(500 * time.Millisecond) // 0.5s * 2/s = 1 token
	if d := l.Allow("client"); !d.Allowed {
		t.Fatal("should be allowed after the bucket refilled")
	}
}

func TestRetryAfter(t *testing.T) {
	clock := newFakeClock()
	l := New(1, 1, clock.Now) // 1 token per second

	l.Allow("client")
	d := l.Allow("client")
	if d.Allowed {
		t.Fatal("expected block")
	}
	if d.RetryAfter != time.Second {
		t.Fatalf("expected RetryAfter of 1s, got %v", d.RetryAfter)
	}
}

func TestClientsAreIsolated(t *testing.T) {
	clock := newFakeClock()
	l := New(1, 1, clock.Now)

	l.Allow("alice")
	if d := l.Allow("bob"); !d.Allowed {
		t.Fatal("bob has his own bucket and should be allowed")
	}
}

// Run with: go test -race ./...
// The race detector checks that the mutex really protects the map.
func TestConcurrentRequests(t *testing.T) {
	clock := newFakeClock()
	l := New(1, 100, clock.Now)

	var wg sync.WaitGroup
	var mu sync.Mutex
	allowed := 0

	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if l.Allow("client").Allowed {
				mu.Lock()
				allowed++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	if allowed != 100 {
		t.Fatalf("expected exactly 100 allowed requests, got %d", allowed)
	}
}
