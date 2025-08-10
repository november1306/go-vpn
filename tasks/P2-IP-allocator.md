# P2 IP Allocator

## Summary
Allocate/release client IPs from a configured CIDR (IPv4), concurrency-safe, persistent.

## Deliverables
- `internal/ipam/allocator.go` with simple functions:
  - `AllocateIP(existingUsers []User) (string, error)` - finds next free IP from 10.0.0.2-254
  - Logic: iterate through range, skip if IP already assigned to user
  - Returns IP in CIDR format (e.g., "10.0.0.5/32")
- Unit tests for allocation logic and exhaustion scenarios

## Acceptance Criteria
- [ ] Deterministic behavior with seed state; race-free

## Dependencies
P2-2.2

## Estimate
3 days








