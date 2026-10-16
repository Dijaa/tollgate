// Package gateway wires the rate limiter into HTTP.
package gateway

import (
	"math"
	"net"
	"net/http"
	"strconv"

	"github.com/dijaa/tollgate/internal/ratelimit"
)

// RateLimit wraps an http.Handler and rejects clients that exceed the limit
// with 429 Too Many Requests.
func RateLimit(l *ratelimit.Limiter, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		d := l.Allow(clientIP(r))

		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(d.Remaining))

		if !d.Allowed {
			// Retry-After is in whole seconds, rounded up.
			secs := int(math.Ceil(d.RetryAfter.Seconds()))
			w.Header().Set("Retry-After", strconv.Itoa(secs))
			http.Error(w, "too many requests", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// clientIP returns the caller's IP address.
//
// TODO (your next step): when the gateway runs behind a load balancer, the real
// client IP comes in the X-Forwarded-For header. Read it only from trusted
// proxies, otherwise anyone can fake it to bypass the limit.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
