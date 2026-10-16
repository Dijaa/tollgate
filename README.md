# tollgate

A rate-limiting API gateway written in Go. It sits in front of an API, forwards
requests, and blocks clients that send too many of them, answering
`429 Too Many Requests` with a `Retry-After` header.

```
client ──> tollgate (rate limit per IP) ──> upstream API
                 │
                 └── 429 + Retry-After when the client is over the limit
```

## Run it

```bash
# terminal 1: a tiny upstream API on :9000
go run ./cmd/echo

# terminal 2: the gateway on :8080 (1 req/s, bursts of 3)
RATE_PER_SECOND=1 BURST=3 go run ./cmd/gateway

# terminal 3: send 5 quick requests
for i in 1 2 3 4 5; do curl -s -o /dev/null -w "%{http_code} " localhost:8080/; done
# -> 200 200 200 429 429
```

| Variable          | Default                 | Meaning                          |
| ----------------- | ----------------------- | -------------------------------- |
| `LISTEN_ADDR`     | `:8080`                 | Where the gateway listens        |
| `UPSTREAM_URL`    | `http://localhost:9000` | API that receives the requests   |
| `RATE_PER_SECOND` | `5`                     | Average requests per second      |
| `BURST`           | `10`                    | Requests allowed in a short burst |

## Test it

```bash
go test -race ./...
```

## How it works

Each client IP gets a **token bucket**: it holds up to `BURST` tokens, every
request spends one, and tokens refill at `RATE_PER_SECOND`. This allows short
bursts while enforcing an average rate.

## Roadmap

- [x] Reverse proxy + in-memory token bucket per IP, with tests
- [ ] Store buckets in Redis so several gateway instances share limits
- [ ] Per API key limits and an IP blocklist
- [ ] Docker Compose (gateway + Redis + echo API)
- [ ] CI with GitHub Actions (`go vet`, `go test -race`)
- [ ] Prometheus metrics (requests, blocked requests, latency)
- [ ] Load test with results in this README
