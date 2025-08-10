# Demo-01 Simple Client-Server Key Exchange

**Priority: HIGH** (Demo/proof of concept)
**Phase: Demo** (Quick working prototype)

## Summary
Create a minimal working demo of client-server communication where:
1. Server exposes HTTP API endpoint to generate WireGuard keys
2. Client CLI requests key from server
3. Server generates key pair and returns it via JSON
4. Client displays key in terminal

## Goal
See something working! Basic client-server communication with key generation.

## Deliverables

### Server Side
- **HTTP Server**: Simple HTTP server listening on port 8443
- **Endpoint**: `GET /api/generate-key` - generates and returns key pair
- **Response Format**: JSON with privateKey and publicKey fields
- **Integration**: Uses our `internal/wireguard/keys.GenerateKeyPair()`

### Client Side  
- **CLI Command**: `vpn-cli register --server=<url>` command (partial implementation)
- **HTTP Client**: Makes POST request to server with client public key
- **Output**: Pretty-prints the server's public key to terminal

## API Specification

**Request:**
```bash
POST /api/register
Content-Type: application/json

{
  "clientPublicKey": "k7vHfX4Km9f8d3wq5N8F2g5z7m1P3Q6r9s2B5x8C4A="
}
```

**Response:**
```json
{
  "serverPublicKey": "m8wHgY5Lm0g9e4xr6O9G3h6z8n2Q4R7t0u3C6y9D5B=",
  "message": "Registration successful - demo mode",
  "timestamp": "2025-01-10T10:30:00Z"
}
```

## Implementation Plan

1. **Update Server** (`cmd/server/main.go`):
   - Add HTTP server with single endpoint
   - Use standard library `net/http`
   - Integrate with `keys.GenerateKeyPair()`

2. **Update Client** (`cmd/vpn-cli/main.go`):
   - Add basic `get-key` command  
   - HTTP client to make request
   - Pretty terminal output

3. **Test Scenario**:
   ```bash
   # Terminal 1: Start server
   go run cmd/server
   # Output: Server listening on :8443, Server public key: m8wHgY5L...
   
   # Terminal 2: Register client
   go run cmd/vpn-cli register --server=http://localhost:8443
   # Output: 
   # Client Registration Demo
   # Generated client key pair
   # Client Public Key: k7vHfX4Km9f8d3wq5N8F2g5z7m1P3Q6r9s2B5x8C4A=
   # Registering with server...
   # ✅ Registration successful!
   # Server Public Key: m8wHgY5Lm0g9e4xr6O9G3h6z8n2Q4R7t0u3C6y9D5B=
   ```

## Acceptance Criteria
- [ ] Server starts and serves on port 8443 with server key generation
- [ ] `curl -X POST http://localhost:8443/api/register -d '{"clientPublicKey":"..."}' -H "Content-Type: application/json"` returns server public key
- [ ] CLI client can generate keys and register with server
- [ ] Server displays received client public keys 
- [ ] Client displays received server public key
- [ ] Keys are valid WireGuard format (using our keys package)
- [ ] Both processes can run simultaneously

## Technical Details
- **Protocol**: HTTP/1.1 (no TLS for demo simplicity)
- **Format**: JSON REST API
- **Libraries**: Standard library only (no external dependencies)
- **Error Handling**: Basic HTTP status codes
- **Logging**: Simple log output for debugging

## Future Extensions
This demo sets foundation for:
- Authentication (API keys)
- Client registration
- Peer management
- TLS/HTTPS security
- Full VPN configuration

## Estimate
2-3 hours (simple implementation)

## Dependencies
- P1-1.3 (Key generation) - **COMPLETED** ✅
- P3-3.1 (CLI setup with Cobra) - **REQUIRED**
- P2-HTTP-server-and-middleware (partial) - **REQUIRED**

## Task Sequence
1. **First**: Complete P3-3.1 (CLI setup with Cobra)
   - Sets up proper CLI framework
   - Provides foundation for `get-key` command
   
2. **Second**: Minimal HTTP server implementation
   - Extract from P2-HTTP-server-and-middleware scope
   - Just the basic server + single endpoint
   
3. **Third**: Demo integration
   - Add `get-key` command to CLI
   - Test end-to-end communication