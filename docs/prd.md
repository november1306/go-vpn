# Product Requirements Document: GoWire VPN MVP

**Primary Target**: Cloud-agnostic Linux server deployment with cross-platform clients  
**Current Deployment**: DigitalOcean production server (134.122.48.36)  
**Timeline**: MVP focused on DigitalOcean deployment, Docker containerization for multi-cloud support

## MVP Scope

**Core Features**:
- **Cloud-agnostic VPN server** hosting WireGuard with native Linux deployment
- Cross-platform CLI client (`register`, `connect`, `disconnect`, `status`, `test-vpn`)
- WireGuard key exchange and peer management (✅ **IMPLEMENTED**)
- File-based persistence (JSON) with atomic writes and concurrency safety
- DigitalOcean production deployment (✅ **IMPLEMENTED**)
- Docker containerization for future cloud deployments (✅ **IMPLEMENTED**)
- Basic logging and error handling

**Explicit Non-Goals**:
- Database integration (using JSON file storage)
- GUI interface (CLI-focused)
- Advanced monitoring/metrics beyond basic connectivity
- Payment/billing systems
- Multi-server mesh networking
- Enterprise authentication (LDAP/SAML)

## Architecture Overview

**Components**:
- `cmd/server`: VPN server with HTTP API + WireGuard interface (runs on DigitalOcean)
- `cmd/vpn-cli`: Cross-platform CLI client (Windows/Linux/macOS) (✅ **IMPLEMENTED**)
- `internal/`: shared packages (config, wireguard, client, server, ipam, version)

**Communication**:
- Control plane: HTTP REST API (port 8443) (✅ **IMPLEMENTED**)
- Data plane: WireGuard UDP (port 51820) (✅ **IMPLEMENTED**)
- Storage: JSON files with atomic writes (✅ **IMPLEMENTED**)

**Deployment Platforms**:
- **DigitalOcean**: Production VM deployment with setup guide (✅ **LIVE SERVER**)
- **Docker**: Containerized deployment for any cloud provider (✅ **IMPLEMENTED**)
- **Manual Linux**: Direct deployment on any Linux VPS (✅ **IMPLEMENTED**)

## Technical Decisions

**Hosting**: Cloud-agnostic (DigitalOcean production, Docker-compatible platforms)  
**Platform**: Cross-platform clients with Go, Linux server deployment  
**WireGuard**: `wireguard-go` userspace implementation (✅ **IMPLEMENTED**)  
**Network**: IPv4 primary, IPv6 ready  
**Subnets**: `10.0.0.0/24` default with IPAM (✅ **IMPLEMENTED**)  
**Authentication**: WireGuard public key exchange (✅ **IMPLEMENTED**), API keys (planned)  
**TLS**: HTTP for development, HTTPS for production (manual TLS setup)  
**Firewall**: Manual configuration per cloud provider

## User Workflow

1. **Server Setup**: Deploy to DigitalOcean VM using comprehensive setup guide
2. **Register**: User runs `vpn-cli register --server http://134.122.48.36:8443` (✅ **WORKING**)
3. **Connect**: User runs `vpn-cli connect` (✅ **IMPLEMENTED** - handles routing automatically)
4. **Disconnect**: User runs `vpn-cli disconnect` (✅ **IMPLEMENTED**)
5. **Status**: User runs `vpn-cli status` (✅ **IMPLEMENTED** - shows connection state, server info)
6. **Test**: User runs `vpn-cli test-vpn` (✅ **IMPLEMENTED** - verifies tunnel functionality)

**Current Deployment Benefits**:
- **DigitalOcean**: Cost-effective ($6/month droplet), full root access, proven VPN tunnel
- **Docker**: Portable deployment across any cloud provider
- **Manual Setup**: Complete control over server configuration and networking

## Current Implementation Status

### ✅ Completed Features
- **WireGuard Key Exchange**: Full public key registration and exchange
- **Cross-Platform CLI**: Windows, Linux, macOS client with all commands
- **DigitalOcean Production**: Live server at 134.122.48.36 with working VPN tunnel
- **Complete Setup Guide**: Comprehensive DigitalOcean deployment documentation
- **Docker Support**: Containerized deployment for any cloud provider
- **IP Allocation**: IPAM system for automatic client IP assignment
- **Tunnel Management**: Connect, disconnect, status, and test commands
- **Userspace WireGuard**: Full tunnel establishment with routing
- **Persistent Server Keys**: Server maintains same keys across restarts, clients register once

### ⏳ In Progress
- **API Key Authentication**: Currently using WireGuard public keys only
- **Production TLS**: Using HTTP, HTTPS setup planned for production

### 📋 Roadmap
- Enhanced authentication system
- Server key persistence
- Multi-server support
- Advanced monitoring and metrics

## Success Criteria

- ✅ Cross-platform users can register and establish VPN connection (DigitalOcean production working)
- ✅ All internet traffic routes through VPN when connected (verified on DigitalOcean)
- ⏳ Secure API key authentication prevents unauthorized access (WireGuard keys implemented, API keys planned)
- ✅ Server handles multiple concurrent users (tested with userspace WireGuard)
- ✅ Clean connection/disconnection without network disruption (Windows/Linux support)
- ✅ Basic operational visibility through logs and status commands
- **Cost-effective**: DigitalOcean $6/month for production with full control
- **Cloud-agnostic**: Architecture proven on DigitalOcean, Docker-ready for other providers
