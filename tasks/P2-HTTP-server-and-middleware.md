# P2 HTTP Server and Middleware

## Summary
Implement net/http server with middleware: logging, request ID, recovery, auth, CORS(off), rate limit.

## Deliverables
- `internal/server/http/server.go` with graceful shutdown
- Middleware: `logging`, `requestid`, `recover`, `auth(apikey)`, `ratelimit` (token bucket in-memory)
- Handlers for `POST /v1/register`, `GET /v1/auth/ping`
- Configurable listen address and TLS cert/key

## Acceptance Criteria
- [ ] Unit tests for middleware chaining
- [ ] `go run cmd/server` serves HTTPS locally with self-signed cert

## Dependencies
P2-API-spec-and-authN, P2-2.5

## Estimate
4 days


