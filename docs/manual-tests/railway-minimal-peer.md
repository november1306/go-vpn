# Manual Test: Railway Minimal Server with Hardcoded Peer

## Overview
This test validates the minimal VPN server deployment on Railway with a hardcoded test peer.

**Goal**: Verify that the server deploys to Railway and can accept VPN clients with full traffic routing.

## Prerequisites

1. **Railway Account**: Sign up at https://railway.app
2. **GitHub Repository**: Project connected to Railway for auto-deploy
3. **Git Access**: Ability to push to main branch
4. **Client WireGuard**: Install WireGuard client on test device

**Note**: Railway CLI is optional since we use auto-deployment

## Part 1: Railway Auto-Deployment

### Step 1: Trigger Auto-Deploy

Railway is configured for automatic deployment on GitHub changes. Simply:

```bash
# Push changes to main branch
git add .
git commit -m "Deploy minimal VPN server to Railway"
git push origin main
```

Railway will automatically detect the push and start deployment using:
- `railway.json` for build configuration
- `Dockerfile` for container setup
- Auto-detected Go environment

### Step 2: Configure Environment Variables

In Railway dashboard (https://railway.app):

1. **Navigate to your project**
2. **Go to Variables tab** 
3. **Add environment variables**:
   ```
   VPN_API_PORT=8443
   VPN_LISTEN_PORT=51820
   VPN_SERVER_IP=10.0.0.1/24
   VPN_INTERFACE=wg0
   ```
4. **Save** (triggers automatic redeploy)

### Step 3: Verify Auto-Deployment

**In Railway Dashboard:**
- ✅ Deployment status shows "SUCCESS"
- ✅ Build logs show successful Go build
- ✅ Runtime logs show server startup

**Get your Railway URL:**
- Copy from Railway dashboard "Domains" section
- Format: `https://your-project-name.up.railway.app`

**Expected Results**:
- ✅ Deployment completes in < 3 minutes
- ✅ Server logs show: "HTTP API server starting port=8443"
- ✅ Server logs show: "VPN server started successfully" (on Linux)
- ✅ Health check passes

## Part 2: API Testing

### Step 3: Test Health Endpoint

```bash
# Test the actual deployed server
curl https://go-vpn-production.up.railway.app/health
```

**Expected Response**:
```
OK - Server running
```

### Step 4: Test Status Endpoint

```bash
# Test the actual deployed server
curl https://go-vpn-production.up.railway.app/api/status
```

**Expected Response** (when VPN running):
```json
{
  "status": "running",
  "connectedPeers": 0,
  "peers": [],
  "serverInfo": {
    "publicKey": "...",
    "endpoint": ":51820",
    "serverIP": "10.0.0.1/24"
  },
  "timestamp": "2025-08-16T12:00:00Z"
}
```

## Part 3: Client Registration

### Step 5: Generate Client Keys

```bash
# On local machine with WireGuard tools
wg genkey | tee client-private.key | wg pubkey > client-public.key

# Or use our CLI (when implemented)
# go run ./cmd/vpn-cli keygen
```

### Step 6: Register Client

```bash
# Get client public key
CLIENT_PUBKEY=$(cat client-public.key)

# Register with the actual deployed server
curl -X POST https://go-vpn-production.up.railway.app/api/register \
  -H "Content-Type: application/json" \
  -d "{\"clientPublicKey\":\"$CLIENT_PUBKEY\"}"

# Alternative: Use the existing CLI (from Demo 1)
# vpn-cli register --server=https://go-vpn-production.up.railway.app
```

**Expected Response**:
```json
{
  "serverPublicKey": "...",
  "serverEndpoint": ":51820",
  "clientIP": "10.0.0.100/32",
  "message": "Registration successful - VPN tunnel established",
  "timestamp": "2025-08-16T12:00:00Z"
}
```

## Part 4: WireGuard Client Configuration

### Step 7: Create Client Config

Create `wg0-client.conf`:
```ini
[Interface]
PrivateKey = <contents of client-private.key>
Address = 10.0.0.100/32
DNS = 8.8.8.8

[Peer]
PublicKey = <serverPublicKey from registration response>
Endpoint = go-vpn-production.up.railway.app:51820
AllowedIPs = 0.0.0.0/0
PersistentKeepalive = 25
```

### Step 8: Connect Client

```bash
# Linux/macOS
sudo wg-quick up wg0-client

# Windows: Import wg0-client.conf in WireGuard app and activate
```

## Part 5: End-to-End Validation

### Step 9: Verify VPN Connection

```bash
# Check WireGuard status
wg show

# Expected output:
# interface: wg0-client
#   public key: ...
#   private key: (hidden)
#   listening port: xxxxx
#   
#   peer: <server-public-key>
#     endpoint: your-project-name.up.railway.app:51820
#     allowed ips: 0.0.0.0/0
#     latest handshake: X seconds ago
#     transfer: X B received, Y B sent
```

### Step 10: Test Traffic Routing

```bash
# Before VPN: Check current IP
curl ifconfig.me
# Should show your real public IP

# After VPN: Check routed IP  
curl ifconfig.me
# Should show Railway server's public IP

# Test specific site access
curl -s https://httpbin.org/ip
# Should show Railway server IP in origin field
```

### Step 11: Verify Server Logs

**In Railway Dashboard:**
1. Go to your project
2. Click "Deployments" tab
3. Click on latest deployment
4. View "Logs" section

**Expected log entries:**
- ✅ "Client registered successfully"
- ✅ VPN server startup messages
- ✅ HTTP API requests
- ✅ Peer connection activity (when VPN active)

## Part 6: Performance Testing

### Step 12: Basic Speed Test

```bash
# Test download speed through VPN
curl -o /dev/null -s -w "%{speed_download}\n" https://speed.cloudflare.com/__down?bytes=10000000

# Test latency
ping -c 10 8.8.8.8
```

**Expected Results**:
- ✅ Download speed > 10 MB/s (Railway bandwidth dependent)
- ✅ Latency increase < 100ms compared to direct connection
- ✅ No packet loss

## Part 7: Cleanup

### Step 13: Disconnect Client

```bash
# Linux/macOS
sudo wg-quick down wg0-client

# Windows: Deactivate tunnel in WireGuard app
```

### Step 14: Verify Disconnection

```bash
# Check IP returns to original
curl ifconfig.me

# Check WireGuard status shows no interfaces
wg show
```

## Success Criteria Checklist

- [ ] ✅ Server deploys successfully to Railway
- [ ] ✅ From client (any platform), traffic routes via Railway server  
- [ ] ✅ Confirmed with `curl ifconfig.me` showing Railway server's public IP
- [ ] ✅ Server logs show peer connection and data transfer statistics
- [ ] ✅ No root privileges required for server operation
- [ ] ✅ Railway deployment completes in < 3 minutes

## Troubleshooting

### Common Issues

1. **Server won't start on Railway**
   - Check Railway logs for port binding errors
   - Verify environment variables are set correctly
   - Ensure Dockerfile is properly configured

2. **Client can't connect**
   - Verify Railway firewall allows UDP port 51820
   - Check client config has correct server endpoint
   - Ensure client keys are properly generated

3. **VPN connects but no internet**
   - Check AllowedIPs in client config (should be 0.0.0.0/0)
   - Verify DNS settings in client config
   - Check Railway server has internet access

4. **Slow performance**
   - Railway free tier has bandwidth limits
   - Check Railway region matches client location
   - Verify userspace WireGuard performance (expected for MVP)

## Notes

- This test uses userspace WireGuard implementation
- Performance ceiling ~500 concurrent users
- Railway provides free hosting tier for testing
- Full production deployment will migrate to Hetzner Cloud