# P2 Config Management

## Summary
Centralize configuration via env vars and optional YAML file.

## Deliverables
- `internal/config/config.go` reading from env (prefix `GOVPN_`) and file `server.yaml`
- Fields: listen addr, tls cert/key paths, wg interface name, wg listen port, server internal CIDR, data dir
- Validation helpers with clear errors

## Acceptance Criteria
- [ ] Unit tests for defaults and overrides

## Dependencies
P1-1.1

## Estimate
2 days


