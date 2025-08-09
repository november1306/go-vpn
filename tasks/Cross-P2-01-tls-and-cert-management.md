# Cross-1 TLS and Certificate Management

**Priority: HIGH** (Required for MVP security)
**Phase: 2** (After basic HTTP server)

## Summary
Provide HTTPS for control-plane API with self-signed dev mode and configurable certs.

## Deliverables
- Dev helper to generate self-signed cert in `dev/certs/` (Go or mkcert)
- Server config loads PEM cert/key; disable HTTP; HSTS optional
- CLI trusts system roots; allow `--insecure` flag only in dev mode

## Acceptance Criteria
- [ ] Register and ping work over TLS locally
- [ ] Self-signed cert generation works out of the box
- [ ] Production-ready cert configuration documented

## Dependencies
P2-HTTP-server-and-middleware

## Estimate
2 days






