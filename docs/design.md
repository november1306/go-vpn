# Technical Design: GoWire VPN

## System Architecture

```
┌─────────────────┐         ┌─────────────────┐
│   vpn-cli.exe   │◄──HTTPS──►│   server.exe    │
│  (Windows CLI)  │           │ (Windows Server)│
└─────────────────┘         └─────────────────┘
         │                           │
         │ WireGuard UDP             │ WireGuard UDP
         │ (port 51820)              │ (port 51820)
         │                           │
    ┌─────────┐                 ┌─────────┐
    │ wg-tun0 │                 │ wg-tun0 │
    │Interface│                 │Interface│
    └─────────┘                 └─────────┘
```

## Component Design

### Server (`cmd/server`)

**Core Responsibilities**:
- HTTP API server with TLS
- WireGuard interface management
- User authentication and registration
- File-based storage with concurrency control

**Key Packages**:
- `net/http` + middleware stack
- `wireguard-go` for VPN tunneling
- `crypto/rand` + `bcrypt` for security
- `sync.RWMutex` for concurrent file access

### CLI Client (`cmd/vpn-cli`)

**Core Responsibilities**:
- User command interface
- Local WireGuard interface management
- API communication with server
- Windows routing table manipulation

**Key Packages**:
- `github.com/spf13/cobra` for CLI
- `wireguard-go` for local tunnel
- `os/exec` for Windows `netsh` commands

## Data Persistence

### Storage Format
```json
{
  "version": 1,
  "server": {
    "private_key": "...",
    "public_key": "...",
    "listen_port": 51820,
    "subnet": "10.0.0.0/24"
  },
  "users": [
    {
      "email": "user@example.com",
      "api_key_hash": "$2a$10$...",
      "client_public_key": "...",
      "assigned_ip": "10.0.0.2/32",
      "registered_at": "2023-08-09T10:30:00Z"
    }
  ]
}
```

### Concurrency Strategy
- Single `sync.RWMutex` protecting all file operations
- Atomic writes via temp file + rename
- In-memory cache for fast user lookups

## Windows Integration

### Network Configuration
- **Interface Creation**: `wireguard-go` handles TUN device
- **IP Assignment**: `netsh interface ip set address`
- **Routing**: `netsh interface ip add route 0.0.0.0/0`
- **DNS**: `netsh interface ip set dns` (optional)

### Privilege Requirements
- **Server**: Administrator for binding ports, creating interfaces
- **Client**: Administrator for route manipulation

### Error Handling
- Check Administrator privileges on startup
- Graceful cleanup on SIGINT/SIGTERM
- Restore original routes on disconnect failure

## Security Model

### API Authentication
1. Client generates WG keypair locally
2. Sends email + public key to `/register`
3. Server generates API key, returns server config
4. Client stores API key in local config file
5. All subsequent requests include `X-API-Key` header

### Key Management
- Server private key generated once, persisted
- API keys: 32 bytes from `crypto/rand`, base64url encoded
- Client private keys: never transmitted, stored locally only

### File Permissions
- Server config: `0600` (owner read/write only)
- Client config: `0600` (owner read/write only)
- Log files: `0644` (owner write, all read)

## IP Address Management (IPAM)

### Subnet Allocation
- Default: `10.0.0.0/24` (254 usable addresses)
- Server: `10.0.0.1/32` (gateway)
- Clients: `10.0.0.2` - `10.0.0.254`

### Allocation Strategy
```go
func (s *Server) allocateIP() (net.IP, error) {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    used := make(map[string]bool)
    for _, user := range s.users {
        used[user.AssignedIP] = true
    }
    
    for i := 2; i <= 254; i++ {
        ip := fmt.Sprintf("10.0.0.%d", i)
        if !used[ip] {
            return net.ParseIP(ip), nil
        }
    }
    return nil, errors.New("no free IPs")
}
```

## Configuration Management

### Server Configuration
```yaml
# server.yaml
listen_port: 51820
api_port: 8443
subnet: "10.0.0.0/24"
data_dir: "./data"
log_level: "info"
tls:
  cert_file: "server.crt"
  key_file: "server.key"
```

### Environment Override
```bash
GOVPN_LISTEN_PORT=51820
GOVPN_API_PORT=8443
GOVPN_DATA_DIR=C:\ProgramData\govpn
GOVPN_LOG_LEVEL=debug
```

## Error Handling Strategy

### Error Categories
- **Network**: interface creation, route manipulation
- **Auth**: invalid API keys, registration failures  
- **Storage**: file corruption, permission issues
- **Config**: missing files, invalid formats

### Client Error UX
```
$ vpn-cli connect
Error: Failed to create VPN interface
Cause: Administrator privileges required
Solution: Run as Administrator or contact IT support
```

## Logging Strategy

### Structured Logging (slog)
- Server: JSON format to stdout
- Client: Human-readable to stderr
- Log levels: DEBUG, INFO, WARN, ERROR

### Log Context
```go
logger.With(
    "request_id", reqID,
    "user_email", email,
    "client_ip", clientIP,
).Info("User registered successfully")
```

### Security Considerations
- Never log API keys or private keys
- Redact sensitive data in error messages
- Separate audit log for security events

## Testing Strategy

### Unit Tests
- `internal/auth`: API key generation/validation
- `internal/storage`: file operations, concurrency
- `internal/ipam`: IP allocation logic

### Integration Tests  
- HTTP API endpoints with test server
- WireGuard interface creation (requires Admin)
- File storage with temporary directories

### Windows-Specific Tests
- Route table manipulation
- Interface creation/deletion
- Privilege detection

## Performance Considerations

### Scalability Targets
- **Users**: 100-200 concurrent connections
- **Throughput**: Limited by CPU (userspace crypto)
- **Memory**: ~1MB per active connection

### Bottlenecks
- File I/O on user registration (add caching)
- bcrypt cost on authentication (consider lower cost)
- Single-threaded WG interface updates

## Deployment Architecture

### Directory Structure
```
C:\Program Files\GoWire\
├── server.exe
├── vpn-cli.exe
├── server.yaml
├── data\
│   ├── users.json
│   ├── wg-server.conf
│   └── logs\
└── certs\
    ├── server.crt
    └── server.key
```

### Service Installation (Future)
- Windows Service wrapper
- Auto-start on boot
- Service recovery policies