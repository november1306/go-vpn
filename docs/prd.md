# Internal PRD: GoWire VPN (MVP)

Audience: engineers. Purpose: align on architecture, interfaces, data, and OS concerns for implementation.

## Goals and Non-Goals
- Goals: single-server WireGuard VPN, API-key auth, CLI UX, JSON persistence, Linux-first.
- Non-goals (MVP): billing, multi-region, SSO, IPv6, GUI.

## High-level Architecture
- Control plane: HTTPS REST API for register and auth ping.
- Data plane: WireGuard UDP (default 51820) on interface `wg0`.
- Storage: `users.json` (atomic write, RWMutex), optional `wg0.conf` snapshot.
- Components: server (`cmd/server`), CLI (`cmd/vpn-cli`), shared `internal/*` packages.

## Key Decisions
- OS: Linux-first; Windows support is best-effort behind build tags.
- WG mode: Userspace `wireguard-go` for MVP; revisit kernel `wgctrl` if host allows.
- IPv4 only (10.0.0.0/24). IPv6 deferred.
- API keys: random 32 bytes base64url; bcrypt hashed at rest.
- TLS required. Dev self-signed. Client `--insecure` only for dev.

## API (REST, v1)
- Header: `X-API-Key: <base64url>` where required.
- Errors: `{ code: string, message: string, request_id: string }`.

Endpoints
1. `POST /v1/register`
   - Request: `{ email: string, client_public_key: string(base64) }`
   - Response: `{ api_key: string, server_public_key: string, endpoint: string, assigned_ip: string }`
   - Auth: none
   - Side effects: persist user {email, bcrypt(api_key), client_pub, assigned_ip}
2. `GET /v1/auth/ping`
   - Auth: required
   - Response: `{ ok: true }`

## Data Model
- User
  - `email` (string)
  - `api_key_hash` (string, bcrypt)
  - `client_public_key` (string)
  - `assigned_ip` (string, CIDR `/32`)
  - `registered_at` (RFC3339)
- Users file: `{ users: [User], version: 1 }`

## IPAM
- Configured CIDR (default `10.0.0.0/24`)
- Exclude gateway (`10.0.0.1/32`) and broadcast
- Allocate lowest free IP; persist in users file; concurrency-safe

## Server Runtime
- Config: env `GOVPN_*` or `server.yaml`
- HTTP: `net/http` with middleware: request ID, logging, recovery, rate-limit
- TLS: cert/key files; `http2` default; HSTS optional
- WG setup (Linux): create interface, set private key, listen port, set IP, IP forwarding on, NAT (`iptables`), firewall open
- Peer lifecycle: add on register; remove when deregistered (future)
- Shutdown: context cancel, drain HTTP, tear down interface

## CLI
- Cobra-based; commands: `register`, `connect`, `disconnect`, `status`
- Stores client private key, server info, api key at `~/.go-wire-vpn/config.json` (0600)
- Connect: auth ping, create TUN, set routes (default via WG), bring up
- Disconnect: restore routes, down & delete TUN

## Security
- Keys: API keys never logged; redact secrets
- Rate limit: token bucket per IP
- Files: atomic writes (`.tmp` + fsync + rename), strict perms
- Admin rights: required for networking; document clearly

## Telemetry & Logging
- slog JSON on server; text on CLI
- Include `request_id`, `remote_ip`, `email` where applicable

## Testing Strategy
- Unit: auth, storage, IPAM, config
- Integration: HTTP server + middleware; WG device creation under Linux
- E2E: register, connect, status, disconnect; assert egress IP
- API contract tests: Playwright APIRequestContext specs

## Risks & Mitigations
- Userspace WG performance: acceptable for MVP; document trade-offs
- Windows routing/NAT complexity: scope as experimental; detailed docs
- File-based store corruption: locks + atomic writes; backups in `var/backup/`

## Open Questions
- Adopt `wgctrl` on Linux sooner?
- Support dual-stack later with `/64` IPv6 plan?

## Milestones
- P1: server WG setup (Linux), minimal peer
- P2: auth, storage, IPAM, API server
- P3: CLI commands, connect/disconnect
- P4: logging/error handling polish
- Cross: TLS, CI, Windows, E2E, security doc
