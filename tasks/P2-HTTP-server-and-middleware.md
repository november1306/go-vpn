# P2 HTTP Server and Middleware

## Summary
Implement HTTP API server for user registration and authentication with basic middleware.

## Deliverables
- `cmd/server` HTTP server with:
  - `POST /register` handler for user registration
  - Basic logging middleware using slog
  - API key authentication middleware
  - TLS configuration (self-signed cert generation)
- Integration with WireGuard peer management and file storage

## Acceptance Criteria
- [ ] Unit tests for middleware chaining
- [ ] `go run cmd/server` serves HTTPS locally with self-signed cert

## Dependencies
P2-API-spec-and-authN, P2-2.5

## Estimate
4 days







