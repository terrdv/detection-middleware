# detection-middleware

A middleware-based bot detection system in Go. It wraps existing HTTP handlers,
scores each incoming request across multiple signals, and takes a tiered action
(allow, challenge, or reject) based on the aggregate suspicion score.

## Detection signals (MVP)

- **User-Agent** — missing/empty UA or known bot patterns (curl, python-requests, Scrapy, headless browsers).
- **Rate limiting** — request frequency per client over a time window.
- **Missing headers** — absence of headers real browsers send (Accept-Language, Accept-Encoding, Referer).
- **Honeypot** — a hidden trap endpoint only a bot would hit (high confidence).
- **Timing pattern** — unnaturally regular request intervals.

## Structure

```
cmd/server/            demo server + load-test target
internal/config/       thresholds, weights, listen address
internal/store/        per-client state (in-memory, IP-keyed)
internal/detector/     signal interface + concrete signals + scoring engine
internal/middleware/   net/http middleware adapter
```


