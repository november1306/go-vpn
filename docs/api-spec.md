# API Specification: GoWire VPN

**Base URL**: `https://localhost:8443`  
**Version**: v1  
**Authentication**: API Key via `X-API-Key` header

## Authentication

All endpoints except `/v1/register` require API key authentication.

**Header**: `X-API-Key: <base64url-encoded-key>`

**Error Response**: 401 Unauthorized
```json
{
  "error": "unauthorized",
  "message": "Invalid or missing API key",
  "request_id": "req_123456"
}
```

## Endpoints

### POST /v1/register

Register a new VPN user and receive API key.

**Authentication**: None required

**Request**:
```json
{
  "email": "user@example.com",
  "client_public_key": "base64-encoded-wireguard-public-key"
}
```

**Response** (200 OK):
```json
{
  "api_key": "base64url-encoded-api-key-32-bytes",
  "server_public_key": "base64-encoded-wireguard-public-key", 
  "endpoint": "vpn.example.com:51820",
  "assigned_ip": "10.0.0.2/32",
  "subnet": "10.0.0.0/24",
  "dns": ["1.1.1.1", "1.0.0.1"]
}
```

**Errors**:
- 400 Bad Request: Invalid email or public key format
- 409 Conflict: Email already registered
- 500 Internal Server Error: Server configuration error

**Example**:
```bash
curl -X POST https://localhost:8443/v1/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "alice@example.com",
    "client_public_key": "base64-key-here"
  }'
```

### GET /v1/auth/ping

Verify API key is valid and user is authorized.

**Authentication**: Required

**Request**: No body

**Response** (200 OK):
```json
{
  "ok": true,
  "user": {
    "email": "user@example.com",
    "assigned_ip": "10.0.0.2/32",
    "registered_at": "2023-08-09T10:30:00Z"
  }
}
```

**Errors**:
- 401 Unauthorized: Invalid API key
- 500 Internal Server Error: Server error

**Example**:
```bash
curl https://localhost:8443/v1/auth/ping \
  -H "X-API-Key: your-api-key-here"
```

### GET /v1/server/config

Get current server configuration for client setup.

**Authentication**: Required

**Request**: No body

**Response** (200 OK):
```json
{
  "server_public_key": "base64-encoded-wireguard-public-key",
  "endpoint": "vpn.example.com:51820", 
  "subnet": "10.0.0.0/24",
  "dns": ["1.1.1.1", "1.0.0.1"]
}
```

**Example**:
```bash
curl https://localhost:8443/v1/server/config \
  -H "X-API-Key: your-api-key-here"
```

### GET /v1/user/status

Get current user connection status and statistics.

**Authentication**: Required

**Request**: No body

**Response** (200 OK):
```json
{
  "user": {
    "email": "user@example.com", 
    "assigned_ip": "10.0.0.2/32",
    "registered_at": "2023-08-09T10:30:00Z"
  },
  "connection": {
    "connected": true,
    "last_handshake": "2023-08-09T15:45:30Z",
    "bytes_sent": 1048576,
    "bytes_received": 2097152
  }
}
```

**Example**:
```bash
curl https://localhost:8443/v1/user/status \
  -H "X-API-Key: your-api-key-here"
```

## Error Response Format

All API errors follow this structure:

```json
{
  "error": "error_code",
  "message": "Human readable error description",
  "request_id": "unique_request_identifier",
  "details": {
    "field": "Additional context if applicable"
  }
}
```

### Standard Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `bad_request` | 400 | Invalid request format or parameters |
| `unauthorized` | 401 | Missing or invalid API key |
| `forbidden` | 403 | Valid API key but access denied |
| `not_found` | 404 | Resource not found |
| `conflict` | 409 | Resource already exists |
| `rate_limited` | 429 | Too many requests |
| `internal_error` | 500 | Server-side error |
| `service_unavailable` | 503 | Server temporarily unavailable |

## Rate Limiting

- **Rate**: 100 requests per minute per IP address
- **Header Response**: `X-RateLimit-Remaining: 95`
- **Error Response**: 429 Too Many Requests

## Request/Response Headers

### Standard Request Headers
```
Content-Type: application/json
X-API-Key: <api-key>
User-Agent: vpn-cli/1.0.0
```

### Standard Response Headers  
```
Content-Type: application/json; charset=utf-8
X-Request-ID: req_123456789
X-RateLimit-Remaining: 95
```

## Data Validation

### Email Format
- Must be valid RFC 5322 email address
- Maximum 254 characters
- Normalized to lowercase

### WireGuard Public Key Format
- Base64-encoded 32-byte key
- Must be valid Curve25519 point
- Example: `base64-encoded-key-here=`

### API Key Format
- Base64url-encoded 32-byte random value
- No padding characters
- Example: `abcd1234efgh5678ijkl9012mnop3456`

## TLS Configuration

- **Minimum TLS Version**: 1.2
- **Cipher Suites**: Modern/secure only
- **Certificate**: Self-signed for development
- **Client Verification**: Optional (not required for MVP)

## Development vs Production

### Development Mode
- Self-signed certificates accepted
- Detailed error messages in responses
- Debug logging enabled

### Production Mode  
- Valid TLS certificates required
- Generic error messages (security)
- Structured JSON logging only