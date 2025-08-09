# Product Requirements Document: GoWire VPN MVP

**Target**: Windows development environment and Windows end users  
**Timeline**: MVP focused on core functionality, database deferred post-MVP

## MVP Scope

**Core Features**:
- Single Windows server hosting WireGuard VPN
- CLI client for Windows users (`register`, `connect`, `disconnect`, `status`)
- API key authentication with secure storage
- File-based persistence (JSON) with atomic writes and concurrency safety
- Basic logging and error handling

**Explicit Non-Goals**:
- Multi-server/load balancing
- Database integration (postponed)
- GUI interface
- Linux/macOS support (Windows-first)
- Advanced monitoring/metrics
- Payment/billing systems

## Architecture Overview

**Components**:
- `cmd/server`: VPN server with HTTPS API + WireGuard interface
- `cmd/vpn-cli`: Windows CLI client
- `internal/`: shared packages (auth, storage, config, logging)

**Communication**:
- Control plane: HTTPS REST API (port 8443)
- Data plane: WireGuard UDP (port 51820)
- Storage: JSON files with file locking

## Technical Decisions

**Platform**: Windows-first with Go cross-compilation support  
**WireGuard**: `wireguard-go` userspace implementation  
**Network**: IPv4 only, default subnet `10.0.0.0/24`  
**Authentication**: 32-byte random API keys, bcrypt hashed  
**TLS**: Required for API (self-signed certificates for dev)  
**Privileges**: Administrator rights required for network configuration

## User Workflow

1. **Setup**: Admin runs `server.exe`, generates TLS certificates
2. **Register**: User runs `vpn-cli register user@example.com`
3. **Connect**: User runs `vpn-cli connect` (requires Admin)
4. **Disconnect**: User runs `vpn-cli disconnect`
5. **Status**: User runs `vpn-cli status` (shows connection state)

## Success Criteria

- Windows users can register and establish VPN connection
- All internet traffic routes through VPN when connected
- Secure API key authentication prevents unauthorized access
- Server handles multiple concurrent users
- Clean connection/disconnection without network disruption
- Basic operational visibility through logs
