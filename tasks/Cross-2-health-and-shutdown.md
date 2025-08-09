# Cross-2 Health Endpoints and Graceful Shutdown

## Summary
Expose healthz/readyz and implement graceful shutdown for HTTP and WG device.

## Deliverables
- `GET /healthz` (no auth), `GET /readyz` (no auth)
- Graceful shutdown on SIGINT/SIGTERM with timeouts

## Acceptance Criteria
- [ ] `curl /healthz` OK; shutdown drains in-flight requests

## Dependencies
P2-HTTP-server-and-middleware, P1-1.4

## Estimate
1 day


