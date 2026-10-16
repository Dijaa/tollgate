// Package ratelimit implements an in-memory token bucket rate limiter.
//
// How a token bucket works:
//   - Each client has a bucket that holds up to `burst` tokens.
//   - Every request spends 1 token. With no tokens left, the request is rejected.
//   - Tokens refill continuously at `rate` tokens per second, up to `burst`.
//
// This allows short bursts of traffic while enforcing an average rate.
package ratelimit

import (
	"math"
	"sync"
	"time"
)

// Clock returns the current time. It is a function so tests can control time.
type Clock func() time.Time

// bucket holds the state for a single client.
type bucket struct {
	tokens   float64   // tokens currently available
	lastSeen time.Time // last time we refilled this bucket
}

// Limiter keeps one bucket per key (for example, per client IP).
type Limiter struct {
	rate  float64 // tokens added per second
	burst float64 // maximum tokens a bucket can hold
	now   Clock

	mu      sync.Mutex         // protects buckets: many requests run concurrently
	buckets map[string]*bucket // key -> bucket
}

// New creates a Limiter that allows `rate` requests per second on average,
// with bursts of up to `burst` requests.
func New(rate float64, burst int, clock Clock) *Limiter {
	if clock == nil {
		clock = time.Now
	}
	return &Limiter{
		rate:    rate,
		burst:   float64(burst),
		now:     clock,
		buckets: make(map[string]*bucket),
	}
}

// Decision is the result of asking the limiter about one request.
type Decision struct {
	Allowed    bool
	Remaining  int           // whole tokens left after this request
	RetryAfter time.Duration // how long until 1 token is available (when blocked)
}

// Allow spends one token from the bucket identified by key, if possible.
func (l *Limiter) Allow(key string) Decision {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	b, ok := l.buckets[key]
	if !ok {
		// New clients start with a full bucket.
		b = &bucket{tokens: l.burst, lastSeen: now}
		l.buckets[key] = b
	}

	// Refill: add tokens for the time elapsed since we last saw this client.
	elapsed := now.Sub(b.lastSeen).Seconds()
	b.tokens = math.Min(l.burst, b.tokens+elapsed*l.rate)
	b.lastSeen = now

	if b.tokens >= 1 {
		b.tokens--
		return Decision{Allowed: true, Remaining: int(b.tokens)}
	}

	// Not enough tokens: compute how long until one full token exists.
	missing := 1 - b.tokens
	wait := time.Duration(missing / l.rate * float64(time.Second))
	return Decision{Allowed: false, Remaining: 0, RetryAfter: wait}
}
