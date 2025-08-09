# Cross-1 TLS and Certificate Management

## Summary
Provide HTTPS for control-plane API with self-signed dev mode and configurable certs.

## Deliverables
- Dev helper to generate self-signed cert in `dev/certs/` (Go or mkcert)
- Server config loads PEM cert/key; disable HTTP; HSTS optional
- CLI trusts system roots; allow `--insecure` only in dev

## Acceptance Criteria
- [ ] Register and ping work over TLS locally

## Dependencies
P2-HTTP-server-and-middleware

## Estimate
2 days


