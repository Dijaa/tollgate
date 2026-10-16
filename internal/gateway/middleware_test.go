package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dijaa/tollgate/internal/ratelimit"
)

func TestReturns429WhenLimited(t *testing.T) {
	now := time.Unix(0, 0)
	l := ratelimit.New(1, 2, func() time.Time { return now })

	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := RateLimit(l, ok)

	codes := []int{}
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "203.0.113.7:5000"
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		codes = append(codes, rec.Code)

		if rec.Code == http.StatusTooManyRequests && rec.Header().Get("Retry-After") != "1" {
			t.Fatalf("expected Retry-After: 1, got %q", rec.Header().Get("Retry-After"))
		}
	}

	want := []int{200, 200, 429}
	for i := range want {
		if codes[i] != want[i] {
			t.Fatalf("request %d: expected %d, got %d", i+1, want[i], codes[i])
		}
	}
}
