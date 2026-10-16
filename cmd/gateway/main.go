// Command gateway is a reverse proxy that rate-limits clients before
// forwarding their requests to an upstream API.
package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"

	"github.com/dijaa/tollgate/internal/gateway"
	"github.com/dijaa/tollgate/internal/ratelimit"
)

func main() {
	listen := env("LISTEN_ADDR", ":8080")
	upstream := env("UPSTREAM_URL", "http://localhost:9000")
	rate := envFloat("RATE_PER_SECOND", 5)
	burst := int(envFloat("BURST", 10))

	target, err := url.Parse(upstream)
	if err != nil {
		log.Fatalf("invalid UPSTREAM_URL %q: %v", upstream, err)
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	limiter := ratelimit.New(rate, burst, nil)
	handler := gateway.RateLimit(limiter, proxy)

	log.Printf("tollgate listening on %s -> %s (%.1f req/s, burst %d)", listen, upstream, rate, burst)
	log.Fatal(http.ListenAndServe(listen, handler))
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envFloat(key string, fallback float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		log.Fatalf("invalid %s %q: %v", key, v, err)
	}
	return f
}
