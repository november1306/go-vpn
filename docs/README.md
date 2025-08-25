# GoWire VPN Documentation

## API Overview

The GoWire VPN server provides a simple HTTP API for client registration and status monitoring.

**Base URL**: `http://localhost:8443` (development)

**Endpoints**:
- `POST /api/register` - Register VPN client with WireGuard public key
- `GET /api/status` - Get server status and connected peers  
- `GET /health` - Health check endpoint
- `GET /api/vpn-test` - Test VPN tunnel functionality

**Key Features**:
- Simple key-based registration (no authentication required for Demo-02)
- JSON-based request/response format
- WireGuard configuration exchange
- Real-time server and peer status monitoring

### Example Usage

**Register a Client:**
```bash
curl -X POST http://localhost:8443/api/register \
  -H "Content-Type: application/json" \
  -d '{"clientPublicKey": "k7vHfX4Km9f8d3wq5N8F2g5z7m1P3Q6r9s2B5x8C4A="}'
```

**Check Server Status:**
```bash
curl http://localhost:8443/api/status
```

**Health Check:**
```bash
curl http://localhost:8443/health
```

### Testing

#### Development Testing
```bash
# Start server
go run ./cmd/server

# Test endpoints
curl http://localhost:8443/health
curl -X POST http://localhost:8443/api/register \
  -H "Content-Type: application/json" \
  -d '{"clientPublicKey": "k7vHfX4Km9f8d3wq5N8F2g5z7m1P3Q6r9s2B5x8C4A="}'
curl http://localhost:8443/api/status
```

#### Integration Testing
```bash
# Run unit tests
make test-unit

# Run integration tests
make test-integration

# Run all tests
make test-all
```

### Error Handling

All endpoints return structured JSON error responses:

```json
{
  "error": "Error message describing what went wrong",
  "timestamp": "2025-01-10T15:30:00Z"
}
```

Common HTTP status codes:
- **200**: Success
- **400**: Bad Request (invalid JSON, missing fields, etc.)
- **404**: Not Found
- **405**: Method Not Allowed  
- **500**: Internal Server Error