# Product Requirements Document: GoWire VPN MVP

**Primary Target**: Hetzner Cloud Linux server with cross-platform clients  
**Timeline**: MVP focused on Hetzner deployment, other cloud providers deferred post-MVP

## MVP Scope

**Core Features**:
- **Hetzner Cloud VPN server** hosting WireGuard with automated setup
- Cross-platform CLI client (`register`, `connect`, `disconnect`, `status`)
- API key authentication with secure storage
- File-based persistence (JSON) with atomic writes and concurrency safety
- Hetzner-optimized networking and firewall configuration
- Basic logging and error handling

**Explicit Non-Goals**:
- Multi-cloud/multi-server deployment (AWS, GCP, Azure - postponed)
- Database integration (postponed)
- GUI interface
- Advanced monitoring/metrics beyond basic connectivity
- Payment/billing systems
- Manual server setup (focus on Hetzner automation)

## Architecture Overview

**Components**:
- `cmd/server`: VPN server with HTTPS API + WireGuard interface (runs on Hetzner Cloud)
- `cmd/vpn-cli`: Cross-platform CLI client (Windows/Linux/macOS)
- `internal/`: shared packages (auth, storage, config, logging, hetzner-integration)

**Communication**:
- Control plane: HTTPS REST API (port 8443)
- Data plane: WireGuard UDP (port 51820)
- Storage: JSON files with file locking

**Hetzner Integration**:
- Automated server provisioning via Hetzner Cloud API
- Optimized for Hetzner's network topology and firewall rules
- Cost-effective scaling on Hetzner's €3.79/month CX22 servers

## Technical Decisions

**Hosting**: Hetzner Cloud Linux servers (primary), extensible to other providers  
**Platform**: Cross-platform clients with Go, Linux server deployment  
**WireGuard**: `wireguard-go` userspace implementation (scales to kernel when needed)  
**Network**: IPv4 primary, IPv6 ready (Hetzner provides free IPv6)  
**Subnets**: `10.0.0.0/24` default, configurable for larger deployments  
**Authentication**: 32-byte random API keys, bcrypt hashed  
**TLS**: Required for API (Let's Encrypt for production, self-signed for dev)  
**Firewall**: Hetzner Cloud Firewall integration for automatic security rules

## User Workflow

1. **Server Setup**: Deploy to Hetzner Cloud via automated script or manual installation
2. **Register**: User runs `vpn-cli register user@example.com --server hetzner-vpn.example.com`
3. **Connect**: User runs `vpn-cli connect` (handles routing automatically)
4. **Disconnect**: User runs `vpn-cli disconnect`
5. **Status**: User runs `vpn-cli status` (shows connection state, server location)

**Hetzner-Specific Benefits**:
- Fast deployment (< 2 minutes from API call to running VPN)
- Global locations (Germany, Finland, USA, Singapore)
- Consistent 1Gbps performance even on basic servers

## Success Criteria

- Cross-platform users can register and establish VPN connection to Hetzner servers
- All internet traffic routes through VPN when connected
- Secure API key authentication prevents unauthorized access
- Server handles multiple concurrent users (target: 100+ on basic Hetzner instance)
- Clean connection/disconnection without network disruption
- Basic operational visibility through logs
- **Hetzner-specific**: Sub-€5/month operating cost for small user base
- **Future-ready**: Architecture supports migration to other cloud providers
