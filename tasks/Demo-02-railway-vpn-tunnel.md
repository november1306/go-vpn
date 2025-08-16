# Demo-02 Railway VPN Tunnel with IP Masking

**Priority: HIGH** (Demo/proof of concept)
**Phase: Demo** (Working VPN tunnel prototype)

## Summary
Deploy the VPN server to Railway and establish a working VPN tunnel where:
1. Server deploys to Railway Linux with public IP
2. Local CLI client connects and registers with Railway server
3. VPN tunnel routes all traffic through Railway server
4. Local IP is masked - `curl ifconfig.me` shows Railway server IP
5. One hardcoded client for simplicity

## Goal
Demonstrate working end-to-end VPN! Traffic routing through Railway with IP masking.

## Deliverables

### Server Side (Railway Deployment)
- **Railway Deploy**: Server runs on Railway Linux with TUN device support
- **VPN Backend**: Full WireGuard tunnel using `internal/server/vpnserver`
- **API Endpoints**: 
  - `POST /api/register` - client registration with IP allocation
  - `GET /api/status` - server and peer status
  - `GET /health` - health check for Railway
- **Configuration**: Environment-based config for Railway deployment
- **Logging**: Structured logging for Railway logs dashboard

### Client Side
- **Registration**: `vpn-cli register --server=https://railway-server.railway.app`
- **Connection**: `vpn-cli connect` - establishes VPN tunnel
- **Verification**: `curl ifconfig.me` returns Railway server IP
- **Disconnect**: `vpn-cli disconnect` - clean tunnel teardown

## Test Scenario

```bash
# 1. Deploy to Railway (automated)
git push railway main
# Railway deploys server with public endpoint

# 2. Local client registration
vpn-cli register --server=https://go-vpn-production.up.railway.app
# Output:
# 🔐 Client Registration Demo
# Generating client key pair...
# ✅ Client Public Key: k7vHfX4Km9f8d3wq5N8F2g5z7m1P3Q6r9s2B5x8C4A=
# 📡 Registering with server: https://go-vpn-production.up.railway.app
# ✅ Registration successful - VPN tunnel established
# 📋 Server Details:
#   Public Key: m8wHgY5Lm0g9e4xr6O9G3h6z8n2Q4R7t0u3C6y9D5B=
#   Endpoint: go-vpn-production.up.railway.app:51820
#   Your VPN IP: 10.0.0.2/32

# 3. Connect VPN tunnel
vpn-cli connect
# Output:
# 🔗 Connecting to VPN server...
# ✅ VPN tunnel established
# 📍 Your traffic is now routed through: Railway US-West

# 4. Verify IP masking
curl ifconfig.me
# Output: [Railway server public IP, not local IP]

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
  "serverEndpoint": "go-vpn-production.up.railway.app:51820", 
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
    "endpoint": "go-vpn-production.up.railway.app:51820",
    "listenPort": 51820
  },
  "timestamp": "2025-01-10T10:33:00Z"
}
```

## Implementation Plan

1. **Railway Configuration**:
   - `railway.json` with port mappings (51820 UDP, 8443 TCP)
   - Environment variables for server config
   - Dockerfile optimized for Railway Linux deployment

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
- [ ] Server deploys successfully to Railway with public endpoint
- [ ] `curl https://go-vpn-production.up.railway.app/health` returns 200 OK
- [ ] Client registration completes with valid WireGuard config
- [ ] `vpn-cli connect` establishes working tunnel
- [ ] `curl ifconfig.me` shows Railway server IP (not local IP)
- [ ] `vpn-cli status` shows connected state with transfer stats
- [ ] `vpn-cli disconnect` cleanly tears down tunnel
- [ ] Local IP restored after disconnect
- [ ] Railway logs show peer connection and data transfer

## Technical Details
- **Deployment**: Railway with Dockerfile
- **Protocol**: HTTPS API (port 8443), WireGuard UDP (port 51820)
- **Network**: `10.0.0.0/24` subnet, client gets `10.0.0.2/32`
- **Routing**: All traffic (`0.0.0.0/0`) through VPN tunnel
- **Platform**: Linux server (Railway), cross-platform client
- **Config**: Environment variables for Railway deployment

## Future Extensions
This demo establishes foundation for:
- Multi-client support with dynamic IP allocation
- Authentication and authorization
- Hetzner Cloud migration
- Traffic analytics and monitoring
- Client configuration management

## Estimate
4-5 hours (Railway deployment + VPN tunnel configuration)

## Dependencies
- P1-1.4 (VPN server backend) - **COMPLETED** ✅
- P1-1.5 (Minimal server) - **IN PROGRESS** 
- Demo-01 (Client-server communication) - **COMPLETED** ✅
- Railway account and CLI setup - **REQUIRED**

## Task Sequence
1. **First**: Complete Railway deployment configuration
   - Update `railway.json` with proper port mappings
   - Configure environment variables for production
   
2. **Second**: Client VPN commands implementation
   - `connect` command with WireGuard interface setup
   - Platform-specific routing table management
   
3. **Third**: End-to-end testing
   - Deploy to Railway and test registration
   - Verify IP masking with `curl ifconfig.me`
   - Test clean connect/disconnect cycles

## Notes
- Use Railway's included TUN device support (Linux containers)
- Start with HTTP for API, upgrade to HTTPS for production
- Focus on single hardcoded client for demo simplicity
- Document Railway deployment process for future automation