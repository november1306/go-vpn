# Cross-2 Health Endpoints and Graceful Shutdown

**Priority: MEDIUM** (Operational readiness)
**Phase: 2** (After server integration)

## Summary
Expose health endpoints and implement graceful shutdown for HTTP server and WireGuard device.

## Deliverables
- `GET /health` endpoint (no auth) - returns server status and WireGuard interface state
- Graceful shutdown on SIGINT/SIGTERM with configurable timeout
- WireGuard interface cleanup on shutdown

## Acceptance Criteria
- [ ] `curl /health` returns 200 OK with status info
- [ ] Shutdown drains in-flight requests before terminating
- [ ] WireGuard interface cleaned up properly on exit

## Dependencies
P2-2.7 (server integration), P1-1.4

## Estimate
1 day






