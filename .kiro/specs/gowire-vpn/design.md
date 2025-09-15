# Design Document

## Overview

GoWire VPN is a self-hosted WireGuard-based VPN solution designed for cloud-agnostic deployment with cross-platform client support. The system follows a client-server architecture where a cloud-hosted server manages WireGuard peers and provides an HTTP API for registration, while cross-platform CLI clients handle local tunnel management and routing configuration.

The design emphasizes simplicity, security, and operational reliability through userspace WireGuard implementation, file-based persistence, and comprehensive error handling. The system is production-ready with a live DigitalOcean deployment and supports containerized deployment for multi-cloud portability.

## Architecture

### System Architecture

```
┌─────────────────────┐         ┌─────────────────────────┐
│   vpn-cli           │◄──HTTP───►│  Cloud VPS Server       │
│ (Cross-platform)    │           │  (DigitalOcean/Docker)  │
│ - Windows/Linux/Mac │           │  - Linux + Docker       │
│ - WireGuard Client  │           │  - HTTP API (8443)      │
│ - Route Management  │           │  - WireGuard (51820)    │
└─────────────────────┘           └─────────────────────────┘
         │                                     │
         │ WireGuard UDP (51820)               │
         │ Encrypted Tunnel                    │
         │                                     │
    ┌─────────┐                         ┌─────────┐
    │ wg-tun0 │                         │ wg0     │
    │Interface│                         │Interface│
    └─────────┘                         └─────────┘
                                              │
                                    ┌─────────────────┐
                                    │ Cloud Provider  │
                                    │ Firewall        │
                                    │ (Manual Config) │
                                    └─────────────────┘
```

### Component Interaction Flow

```
Registration Flow:
Client → Generate Keys → HTTP API → Server → Store Peer → Return Config

Connection Flow:
Client → Load Config → Create WireGuard Interface → Configure Routes → Establish Tunnel

Data Flow:
Client Traffic → WireGuard Interface → Encrypted UDP → Server → Internet
```

## Components and Interfaces

### Server Component (`cmd/server`)

**Primary Responsibilities:**
- HTTP API server for peer management and registration
- WireGuard interface management using userspace backend
- Persistent peer storage with atomic file operations
- IP address allocation and management (IPAM)
- Graceful shutdown and error handling

**Key Interfaces:**

```go
// HTTP API Endpoints
POST /api/register    - Client registration and key exchange
GET  /api/status      - Server status and peer information
GET  /health          - Health check endpoint
GET  /api/vpn-test    - VPN tunnel functionality test

// Core Server Interface
type VPNServer interface {
    Start(ctx context.Context, config ServerConfig) error
    Stop(ctx context.Context) error
    AddClient(publicKey string, clientIP string) error
    RemoveClient(publicKey string) error
    GetConnectedClients() ([]PeerInfo, error)
    IsRunning() bool
    GetServerInfo() (ServerInfo, error)
}
```

**WireGuard Backend Abstraction:**

```go
type WireGuardBackend interface {
    Start(ctx context.Context, config ServerConfig) error
    Stop(ctx context.Context) error
    AddPeer(publicKey string, allowedIPs []string) error
    RemovePeer(publicKey string) error
    GetPeers() ([]PeerInfo, error)
    IsRunning() bool
}
```

### CLI Client Component (`cmd/vpn-cli`)

**Primary Responsibilities:**
- Cross-platform command-line interface using Cobra
- Local WireGuard interface management (userspace implementation)
- Platform-specific routing table manipulation
- Configuration persistence and management
- Connection status monitoring and testing

**Command Interface:**

```go
// CLI Commands
vpn-cli register --server <url>  - Register with VPN server
vpn-cli connect                  - Establish VPN tunnel
vpn-cli disconnect               - Tear down VPN tunnel
vpn-cli status                   - Show connection status
vpn-cli test-vpn                 - Test tunnel functionality
```

**Tunnel Management Interface:**

```go
type TunnelManager interface {
    Connect() error
    Disconnect() error
    IsConnected() bool
    GetStatus() (*TunnelStatus, error)
}
```

### Internal Packages

**Configuration Management (`internal/config`):**
- Environment-based configuration with sensible defaults
- Validation and type safety for all configuration parameters
- Support for development and production environments

**WireGuard Integration (`internal/wireguard`):**
- Key generation and validation using crypto/rand
- Userspace WireGuard device management
- Cross-platform TUN interface handling

**Client Configuration (`internal/client/config`):**
- Secure local configuration storage (0600 permissions)
- JSON-based persistence with atomic writes
- Registration state management

**Server Persistence (`internal/server/vpnserver`):**
- File-based peer storage with concurrent access protection
- Atomic write operations using temp files and rename
- Peer restoration on server restart

## Data Models

### Server Configuration

```go
type ServerConfig struct {
    InterfaceName string // WireGuard interface name (e.g., "wg0")
    PrivateKey    string // Server's WireGuard private key
    ListenPort    int    // UDP port for WireGuard (default: 51820)
    ServerIP      string // Server IP within VPN network (e.g., "10.0.0.1/24")
}

type Config struct {
    Server   ServerConfig  // HTTP and WireGuard server settings
    Network  NetworkConfig // VPN network configuration
    Timeouts TimeoutConfig // HTTP and operation timeouts
    Test     TestConfig    // Test-specific settings
}
```

### Client Configuration

```go
type ClientConfig struct {
    ClientPrivateKey string    // Client's WireGuard private key (never transmitted)
    ClientPublicKey  string    // Client's WireGuard public key
    ServerPublicKey  string    // Server's WireGuard public key
    ServerEndpoint   string    // Server endpoint (IP:port)
    ClientIP         string    // Assigned VPN IP address with CIDR
    RegisteredAt     time.Time // Registration timestamp
}
```

### Peer Storage

```json
{
  "version": 1,
  "server": {
    "private_key": "base64-encoded-private-key",
    "public_key": "base64-encoded-public-key",
    "listen_port": 51820,
    "subnet": "10.0.0.0/24"
  },
  "peers": [
    {
      "public_key": "base64-encoded-client-public-key",
      "allowed_ips": "10.0.0.100/32",
      "registered_at": "2023-08-09T10:30:00Z"
    }
  ]
}
```

### API Data Models

```go
// Registration Request/Response
type RegisterRequest struct {
    ClientPublicKey string `json:"clientPublicKey"`
}

type RegisterResponse struct {
    ServerPublicKey string `json:"serverPublicKey"`
    ServerEndpoint  string `json:"serverEndpoint"`
    ClientIP        string `json:"clientIP"`
    Message         string `json:"message"`
    Timestamp       string `json:"timestamp"`
}

// Status Response
type StatusResponse struct {
    Status         string     `json:"status"`
    ConnectedPeers int        `json:"connectedPeers"`
    Peers          []PeerInfo `json:"peers"`
    ServerInfo     ServerInfo `json:"serverInfo"`
    Timestamp      string     `json:"timestamp"`
}
```

## Error Handling

### Error Categories and Strategies

**Network Errors:**
- Interface creation failures (privilege issues, driver problems)
- Route manipulation failures (platform-specific handling)
- Connection timeouts and network unreachability
- Strategy: Graceful degradation with specific error messages and resolution guidance

**Authentication Errors:**
- Invalid public key formats (cryptographic validation)
- Registration failures (server-side validation)
- API communication errors (HTTP status handling)
- Strategy: Clear error messages with security considerations (no information disclosure)

**Storage Errors:**
- File permission issues (0600 enforcement)
- Disk space and I/O errors (atomic write protection)
- Configuration corruption (validation and recovery)
- Strategy: Atomic operations with rollback capability

**Platform-Specific Errors:**
- Windows: Administrator privilege requirements, WinTun driver issues
- Linux: Root access for network configuration, kernel module dependencies
- macOS: System integrity protection, network extension requirements
- Strategy: Platform-specific error detection with targeted resolution steps

### Error Response Format

```go
type ErrorResponse struct {
    Error     string `json:"error"`
    Timestamp string `json:"timestamp"`
}

// Client-side error handling
func handleError(err error) {
    if strings.Contains(err.Error(), "Access is denied") {
        fmt.Println("💡 Solution: Run as Administrator")
    } else if strings.Contains(err.Error(), "wintun.dll") {
        fmt.Println("💡 Solution: Download WinTun drivers")
    }
    // ... platform-specific guidance
}
```

## Testing Strategy

### Unit Testing

**Server Components:**
- HTTP API endpoint testing with test servers
- WireGuard backend interface testing with mock implementations
- Configuration validation and error handling
- Peer storage operations with temporary directories

**Client Components:**
- Command-line interface testing with Cobra test framework
- Configuration persistence testing with temporary files
- Key generation and validation testing
- Platform detection and error message testing

**Internal Packages:**
- Cryptographic key operations (generation, validation, conversion)
- IPAM allocation logic with edge cases
- File operations with concurrent access simulation
- Network interface detection and manipulation

### Integration Testing

**End-to-End Workflows:**
- Complete registration flow (client → server → storage)
- Connection establishment with actual WireGuard interfaces
- Graceful disconnection and cleanup verification
- Server restart with peer restoration

**Cross-Platform Testing:**
- Windows: Administrator privilege detection, WinTun driver integration
- Linux: Root access validation, ip command integration
- macOS: System permission handling, route command integration

**Docker Integration:**
- Container deployment with volume persistence
- Network configuration within containerized environment
- Multi-stage build verification and optimization

### Performance Testing

**Scalability Targets:**
- 100-200 concurrent client connections
- Sub-second registration response times
- Minimal memory footprint (< 50MB base usage)
- Efficient file I/O with concurrent peer operations

**Bottleneck Analysis:**
- bcrypt cost optimization for API key hashing
- File I/O optimization with in-memory caching
- WireGuard userspace performance vs kernel implementation
- HTTP server performance under load

## Security Considerations

### Cryptographic Security

**Key Management:**
- WireGuard keys generated using crypto/rand (cryptographically secure)
- Private keys never transmitted over network
- Server private key persistence with 0600 file permissions
- Public key validation to prevent malformed key attacks

**API Security:**
- HTTPS enforcement for production deployments
- API key authentication with secure hashing (bcrypt)
- Request validation and sanitization
- Rate limiting considerations for registration endpoints

### Network Security

**Firewall Configuration:**
- Minimal port exposure (8443 for API, 51820 for WireGuard)
- Cloud provider security group integration
- Host-based firewall configuration guidance
- DDoS protection considerations

**Traffic Isolation:**
- Client IP allocation with /32 allowed IPs (strict isolation)
- Server-side routing validation
- Network namespace isolation (future enhancement)
- DNS leak prevention

### Operational Security

**File System Security:**
- Configuration files with restrictive permissions (0600)
- Atomic write operations to prevent corruption
- Secure temporary file handling
- Log sanitization (no private key logging)

**Process Security:**
- Privilege escalation detection and guidance
- Graceful handling of permission errors
- Secure cleanup on process termination
- Memory protection for sensitive data

## Deployment Architecture

### Cloud Deployment Options

**DigitalOcean Production (Current):**
```
DigitalOcean Droplet (134.122.48.36):
├── /root/go-vpn/
│   ├── bin/server (Go binary)
│   ├── data/peers.json (persistent storage)
│   └── systemd service configuration
├── Network Configuration:
│   ├── IP forwarding enabled (/proc/sys/net/ipv4/ip_forward)
│   ├── iptables NAT rules for traffic forwarding
│   └── UFW firewall (ports 8443, 51820)
└── WireGuard Interface:
    ├── wg0 (10.0.0.1/24)
    └── Userspace implementation (wireguard-go)
```

**Docker Deployment (Multi-Cloud):**
```dockerfile
FROM golang:1.21-alpine AS builder
# Multi-stage build for minimal production image
COPY . /app
RUN go build -o server cmd/server/main.go

FROM alpine:latest
RUN apk add --no-cache iptables
COPY --from=builder /app/server /app/server
EXPOSE 8443 51820/udp
VOLUME ["/data"]
CMD ["/app/server"]
```

**Manual Linux Deployment:**
```
/opt/go-vpn/ (or /root/go-vpn/):
├── bin/server (Linux binary)
├── data/peers.json (persistent storage)
├── systemd service configuration
└── Manual network configuration scripts
```

### Service Management

**Systemd Service Configuration:**
```ini
[Unit]
Description=GoWire VPN Server
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/root/go-vpn
ExecStart=/root/go-vpn/bin/server
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

**Health Monitoring:**
- HTTP health check endpoint (/health)
- Systemd service status monitoring
- Log aggregation and rotation
- Basic connectivity testing

### Network Configuration

**Server Network Setup:**
```bash
# Enable IP forwarding
echo 'net.ipv4.ip_forward=1' >> /etc/sysctl.conf
sysctl -p

# Configure iptables NAT
iptables -t nat -A POSTROUTING -s 10.0.0.0/24 -o eth0 -j MASQUERADE
iptables -A FORWARD -i wg0 -j ACCEPT
iptables -A FORWARD -o wg0 -j ACCEPT

# Configure firewall
ufw allow 8443/tcp
ufw allow 51820/udp
```

**Client Network Requirements:**
- Administrator/root privileges for route manipulation
- TUN/TAP driver support (WinTun on Windows)
- Outbound UDP connectivity on port 51820
- DNS resolution for server endpoint

## Performance and Scalability

### Current Performance Characteristics

**Throughput:**
- Limited by userspace WireGuard implementation
- CPU-bound encryption/decryption operations
- Typical throughput: 100-500 Mbps depending on hardware

**Latency:**
- Userspace processing overhead: ~1-2ms additional latency
- Network latency depends on client-server geographic distance
- Connection establishment: <1 second for registration

**Resource Usage:**
- Memory: ~1MB per active peer connection
- CPU: Moderate usage during active data transfer
- Disk: Minimal (JSON configuration files only)

### Scalability Considerations

**Horizontal Scaling:**
- Current design: Single server instance
- Future enhancement: Multi-server mesh networking
- Load balancing considerations for API endpoints
- Geographic distribution for latency optimization

**Vertical Scaling:**
- CPU scaling: More cores improve concurrent connection handling
- Memory scaling: Linear growth with peer count
- Network scaling: Bandwidth limitations at cloud provider level
- Storage scaling: Minimal requirements (peer data is small)

### Optimization Opportunities

**Performance Improvements:**
- Kernel WireGuard implementation for higher throughput
- In-memory peer caching to reduce file I/O
- Connection pooling for HTTP API clients
- Batch operations for peer management

**Resource Optimization:**
- Memory pool management for frequent allocations
- Efficient JSON parsing and serialization
- Log level optimization for production
- Garbage collection tuning for long-running processes

## Future Enhancements

### Authentication System Evolution

**Current State:** WireGuard public key exchange with planned API key authentication
**Planned Enhancements:**
- JWT-based authentication with expiration
- Multi-factor authentication support
- Integration with external identity providers (LDAP, OAuth)
- Role-based access control for administrative functions

### Multi-Server Architecture

**Current State:** Single server deployment
**Planned Enhancements:**
- Server discovery and selection based on geographic proximity
- Load balancing across multiple server instances
- Failover and redundancy mechanisms
- Centralized peer management across server fleet

### Advanced Monitoring

**Current State:** Basic logging and status endpoints
**Planned Enhancements:**
- Prometheus metrics integration
- Real-time connection monitoring dashboard
- Alerting for server health and performance issues
- Detailed bandwidth and usage analytics

### Enterprise Features

**Current State:** Individual user focus
**Planned Enhancements:**
- Team and organization management
- Centralized policy enforcement
- Audit logging and compliance reporting
- Integration with enterprise security tools