# Demo-02 Local VPN Tunnel with Server/Client

**Priority: HIGH** (Demo/proof of concept)
**Phase: Demo** (Working VPN tunnel prototype)

**⚠️ Updated Scope**: Railway deployment requires Linux root privileges for TUN device creation. Demo updated to run server and client locally for development/testing. Production deployment will target dedicated Linux VM.

## Summary
Demonstrate working VPN tunnel with local server and client:
1. VPN server runs locally with TUN interface support
2. CLI client connects and registers with local server
3. VPN tunnel established between server and client
4. Verify tunnel connectivity and configuration
5. Foundation for future remote Linux VM deployment

## Goal
Demonstrate working end-to-end VPN locally! Establish foundation for remote deployment.

## Deliverables

### Server Side (Local Development)
- **Local Server**: Server runs locally with TUN device support (Linux/macOS)
- **VPN Backend**: Full WireGuard tunnel using `internal/server/vpnserver`
- **API Endpoints**: 
  - `POST /api/register` - client registration with IP allocation
  - `GET /api/status` - server and peer status
  - `GET /health` - health check endpoint
  - `GET /api/vpn-test` - VPN tunnel functionality test endpoint
- **Configuration**: Local config file or environment variables
- **Logging**: Console logging for development

### Client Side
- **Registration**: `vpn-cli register --server=http://localhost:8443`
- **Connection**: `vpn-cli connect` - establishes VPN tunnel
- **Verification**: Ping server IP and check tunnel connectivity
- **Disconnect**: `vpn-cli disconnect` - clean tunnel teardown

## Test Scenario

```bash
# 1. Start local VPN server
go run ./cmd/server
# Server starts on localhost:8443 (HTTP API) and localhost:51820 (VPN)

# 2. Local client registration (in separate terminal)
vpn-cli register --server=http://localhost:8443
# Output:
# 🔐 Client Registration Demo
# Generating client key pair...
# ✅ Client Public Key: k7vHfX4Km9f8d3wq5N8F2g5z7m1P3Q6r9s2B5x8C4A=
# 📡 Registering with server: http://localhost:8443
# ✅ Registration successful - VPN tunnel established
# 📋 Server Details:
#   Public Key: m8wHgY5Lm0g9e4xr6O9G3h6z8n2Q4R7t0u3C6y9D5B=
#   Endpoint: localhost:51820
#   Your VPN IP: 10.0.0.2/32

# 3. Connect VPN tunnel
vpn-cli connect
# Output:
# 🔗 Connecting to VPN server...
# ✅ VPN tunnel established
# 📍 VPN tunnel established to local server

# 4. Verify tunnel connectivity
ping 10.0.0.1
# Output: PING 10.0.0.1 (10.0.0.1) 56(84) bytes of data.
# 64 bytes from 10.0.0.1: icmp_seq=1 ttl=64 time=0.123 ms

# 5. Disconnect
vpn-cli disconnect
# Output:
# 🔌 Disconnecting VPN tunnel...
# ✅ VPN tunnel closed
# 📍 Traffic restored to direct routing
```

## API Specification

**Registration Request:**
```bash
POST /api/register
Content-Type: application/json

{
  "clientPublicKey": "k7vHfX4Km9f8d3wq5N8F2g5z7m1P3Q6r9s2B5x8C4A="
}
```

**Registration Response:**
```json
{
  "serverPublicKey": "m8wHgY5Lm0g9e4xr6O9G3h6z8n2Q4R7t0u3C6y9D5B=",
  "serverEndpoint": "localhost:51820", 
  "clientIP": "10.0.0.2/32",
  "message": "Registration successful - VPN tunnel established",
  "timestamp": "2025-01-10T10:30:00Z"
}
```

**Status Endpoint:**
```bash
GET /api/status
```

**Status Response:**
```json
{
  "status": "running",
  "connectedPeers": 1,
  "peers": [
    {
      "publicKey": "k7vHfX4Km9f8d3wq...",
      "allowedIPs": ["10.0.0.2/32"],
      "endpoint": "203.0.113.45:52847",
      "lastHandshake": "2025-01-10T10:32:15Z",
      "transferRx": 1024576,
      "transferTx": 2048192
    }
  ],
  "serverInfo": {
    "publicKey": "m8wHgY5Lm0g9e4xr...",
    "endpoint": "localhost:51820",
    "listenPort": 51820
  },
  "timestamp": "2025-01-10T10:33:00Z"
}
```

## Implementation Plan

1. **Local Development Setup**:
   - Server configuration for localhost deployment
   - TUN interface creation (requires sudo/admin privileges)
   - Local port bindings (51820 UDP, 8443 TCP)

2. **Server Updates** (`cmd/server/main.go`):
   - Full VPN server integration using `internal/server/vpnserver`
   - Client IP allocation from `10.0.0.0/24` subnet
   - Proper error handling for registration failures
   - Health endpoint for Railway health checks

3. **Client Updates** (`cmd/vpn-cli/main.go`):
   - `connect` command that configures local WireGuard interface
   - `disconnect` command with route cleanup
   - `status` command showing connection state
   - Cross-platform routing table management

4. **VPN Configuration**:
   - Server: `10.0.0.1/24` (Railway)
   - Client: `10.0.0.2/32` (hardcoded for demo)
   - Route all traffic (`0.0.0.0/0`) through tunnel

## Acceptance Criteria
- [ ] Server starts successfully locally with TUN interface
- [ ] `curl http://localhost:8443/health` returns 200 OK
- [ ] Client registration completes with valid WireGuard config
- [ ] `vpn-cli connect` establishes working tunnel
- [ ] `ping 10.0.0.1` succeeds through VPN tunnel
- [ ] `vpn-cli status` shows connected state with server details
- [ ] `vpn-cli disconnect` cleanly tears down tunnel
- [ ] Server logs show peer connection and registration
- [ ] Foundation ready for remote Linux VM deployment

## Technical Details
- **Deployment**: Local development environment
- **Protocol**: HTTP API (port 8443), WireGuard UDP (port 51820)
- **Network**: `10.0.0.0/24` subnet, client gets `10.0.0.2/32`
- **Routing**: VPN subnet routing for tunnel connectivity
- **Platform**: Local Linux/macOS for server, cross-platform client
- **Config**: Local configuration files

## Future Extensions
This demo establishes foundation for:
- Remote Linux VM deployment (Hetzner Cloud/DigitalOcean)
- Multi-client support with dynamic IP allocation
- Authentication and authorization
- Traffic analytics and monitoring
- Client configuration management
- Full internet traffic routing (0.0.0.0/0)

## Estimate
4-5 hours (Railway deployment + VPN tunnel configuration)

## Dependencies
- P1-1.4 (VPN server backend) - **COMPLETED** ✅
- P1-1.5 (Minimal server) - **COMPLETED** ✅
- P3-3.1 (CLI setup) - **COMPLETED** ✅
- P3-3.2 (Client config storage) - **COMPLETED** ✅
- Demo-01 (Client-server communication) - **COMPLETED** ✅
- Linux/macOS environment with sudo access - **REQUIRED**

## Task Sequence
1. **First**: Complete Railway deployment configuration
   - Update `railway.json` with proper port mappings
   - Configure environment variables for production
   
2. **Second**: Client VPN commands implementation
   - `connect` command with WireGuard interface setup
   - Platform-specific routing table management
   
3. **Third**: End-to-end testing
   - Test local server startup and API endpoints
   - Verify tunnel establishment and connectivity
   - Test clean connect/disconnect cycles

## Notes
- Requires sudo/admin privileges for TUN interface creation
- HTTP API for local development, HTTPS for remote deployment
- Focus on single hardcoded client for demo simplicity
- Document local setup process for team development
- Plan migration path to remote Linux VM deployment