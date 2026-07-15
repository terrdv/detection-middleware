# detection-middleware

A middleware-based bot detection system in Go. It wraps existing `net/http`
handlers, scores each incoming request across several independent signals, and
takes a tiered action — **allow**, **challenge**, or **reject** — based on the
aggregate suspicion score.

## How it works

A request flows through the stack like this:

```
net/http  ->  middleware  ->  detector  ->  store
                   |              |            |
                   |              |            └─ per-client history (request timestamps)
                   |              └─ runs every Signal, sums weighted scores, picks an action
                   └─ applies the action: pass through / 429 / 403
```

### Signals

Each signal implements a small interface and returns a suspicion contribution in
`[0, 1]`. It sees the request plus a snapshot of the client's recent history:

```go
type Signal interface {
    Name() string
    Score(r *http.Request, state *store.ClientState) float64
}
```

| Signal        | What it flags                                                                 | Weight |
|---------------|-------------------------------------------------------------------------------|--------|
| `user_agent`  | Empty UA, or known bot patterns (`curl`, `python-requests`, `scrapy`, headless browsers). Non-`Mozilla/` UAs get a partial score. | 0.25 |
| `headers`     | Missing headers real browsers send (`Accept`, `Accept-Language`, `Accept-Encoding`). Graded by how many are absent. | 0.20 |
| `timing`      | Unnaturally regular request intervals — low coefficient of variation across timestamps. | 0.50 |
| `rate_limit`  | Request count over a sliding window. 0 up to the limit, ramping to 1 at twice the limit. | 0.50 |
| `honeypot`    | Any request to a hidden trap endpoint (`/wp-admin`) — instant high confidence. | 1.00 |

### Scoring and actions

The detector computes `score = Σ (signal_score × weight)` and maps it onto a
tier via two thresholds (`config.Default()`):

```
score < 0.4          -> ActionAllow      (pass through)
0.4 <= score < 0.7   -> ActionChallenge  (429 Too Many Requests)
score >= 0.7         -> ActionReject      (403 Forbidden)
```

### Client identity and the store

State is tracked per client. By default the client key is the TCP source IP
(`RemoteAddr` with the port stripped). When `TrustForwardedFor` is on, the first
hop of `X-Forwarded-For` is used instead — required behind a trusted proxy, and
what lets a single machine simulate many clients during load testing. **It's off
by default because the header is spoofable.**

The store is an in-memory, IP-keyed history of request timestamps. It is split
into **256 lock-striped shards** (keyed by an FNV hash of the client key) so
independent clients rarely contend on the same mutex. On each `Record` it drops
timestamps older than the retention window, caps per-client samples (128), and
returns a **copy** of the history so signals can read it outside the lock.

## Layout

```
cmd/server/            demo server + load-test target
internal/config/       thresholds, weights, listen address
internal/store/        per-client state (sharded, in-memory, IP-keyed)
internal/detector/     signal interface + concrete signals + scoring engine
internal/middleware/   net/http middleware adapter
loadtest/              vegeta scripts for end-to-end load testing
```

## Setup

Requires **Go 1.23+**. There are no third-party dependencies for the server
itself; load testing uses [vegeta](https://github.com/tsenart/vegeta).

```bash
git clone <repo-url>
cd detection-middleware
go build ./...
```

## Running the server

```bash
go run ./cmd/server
# listening on :8080
```

It serves `/` (returns `ok`) wrapped in the detection middleware. Try it:

```bash
# Allowed — looks like a browser
curl -s -H 'Accept: text/html' -H 'Accept-Language: en' -H 'Accept-Encoding: gzip' \
     -H 'User-Agent: Mozilla/5.0' http://localhost:8080/

# Rejected — honeypot path
curl -i http://localhost:8080/wp-admin        # 403 Forbidden

# Scores up — bot UA + missing headers
curl -i http://localhost:8080/                 # default curl UA
```

To trust `X-Forwarded-For` (needed for load testing distinct clients from one
host), start with `TRUST_XFF=1`:

```bash
TRUST_XFF=1 go run ./cmd/server
```

## Testing

```bash
go test ./...                 # all unit tests
go test -race ./...           # with the race detector (recommended for the store)
```

The detector tests cover each signal (user-agent, headers, timing, rate limit,
honeypot), client-key derivation, and end-to-end evaluation. The store tests
include a concurrency test that hammers a single key from many goroutines —
run it under `-race` to verify the snapshot-under-lock behavior:

```bash
go test -race -run TestConcurrentRecordNoRace ./internal/store/
```

## Load testing

End-to-end load tests live in `loadtest/` and drive the whole stack
(`net/http -> middleware -> detector -> store`) with
[vegeta](https://github.com/tsenart/vegeta).

### Prerequisites

```bash
brew install vegeta
TRUST_XFF=1 go run ./cmd/server     # server must trust X-Forwarded-For
```

### 1. Generate a targets file

Vegeta needs a list of request targets. `gen_targets.sh` builds one, using
`X-Forwarded-For` to simulate distinct clients:

```bash
# 1000 rotating client IPs (spreads load across shards)
cd loadtest
./gen_targets.sh > targets.txt

# a single hot client (single shard — the contrast case)
MODE=one ./gen_targets.sh > targets_one.txt

# custom count / URL
CLIENTS=5000 URL=http://localhost:8080/ ./gen_targets.sh > targets.txt
```

### 3. Concurrency

```bash
./run_concurrency.sh
CONCURRENCY="1 8 64 256 1000" DURATION=10s TARGETS=targets.txt ./run_concurrency.sh
```

## Configuration

All tunablesi live in `internal/config/config.go` (`config.Default()`): listen
address, per-signal weights, challenge/reject thresholds, honeypot path, and the
rate-limit window and limit. Note the coupling called out there — keep
`2 × RateLimitLimit <= maxSamples` (128) or the `rate_limit` signal can never
reach 1.0.

## Evaluation

Evaluated along two independent axes — **concurrency** and
**detection accuracy** — with different tools for each. Numbers below are from an
Apple M3 (8 cores).

### Performance: single mutex vs. lock-striped store

Closed-loop concurrency sweep against the live endpoint with vegeta (`-rate=0`,
workers pinned to N), 1000 distinct clients, `maxSamples=128`. Reading the
achieved request rate and p99 latency, before and after sharding the store:

| Concurrency | Before (1 mutex) | After (256 shards) | Before p99 | After p99 |
|------------:|-----------------:|-------------------:|-----------:|----------:|
| 1           | 25,400 req/s     | 27,600 req/s       | 119µs      | 92µs      |
| 8           | 65,900           | 72,400             | 430µs      | 341µs     |
| 64          | 82,100           | 91,900             | 2.4ms      | 1.8ms     |
| 256         | 91,100           | **100,000**        | 6.9ms      | 5.8ms     |
| 1000        | 79,600           | **102,000**         | 28ms       | 21ms      |

### Accuracy: labeled human/bot simulation

A Python simulator (`loadtest/sim/`) drives labeled traffic — browser profiles
with irregular think time, scripting-tool and honeypot-scanning bots, and a
"looks human but fires on a metronome" bot — and scores the detector's verdict
(200 = human; 403/429 = bot) into a confusion matrix. 100 clients, 625 requests:

|                | predicted bot | predicted human |
|----------------|--------------:|----------------:|
| **actual bot**   | 425 (TP)    | 50 (FN)         |
| **actual human** | 34 (FP)     | 116 (TN)        |

**precision 0.93 · recall 0.90 · accuracy 0.87**

