# P2 IP Allocator

## Summary
Allocate/release client IPs from a configured CIDR (IPv4), concurrency-safe, persistent.

## Deliverables
- `internal/server/ipam` with:
  - `Allocator` supporting `Allocate() (net.IP, error)` and `Release(ip net.IP) error`
  - Persistence in users file; no separate DB
  - Avoids server IP and broadcast; skips in-use
- Tests for exhaustion, concurrent allocate, release/reuse

## Acceptance Criteria
- [ ] Deterministic behavior with seed state; race-free

## Dependencies
P2-2.2

## Estimate
3 days


