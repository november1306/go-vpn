# Docker Deployment Guide: GoWire VPN

## Quick Start

### Development Environment
```bash
# Clone and start
git clone <repository-url>
cd go-vpn
docker-compose up --build
```

### Production Deployment
```bash
# Pull latest image
docker pull ghcr.io/november1306/go-vpn:latest

# Run with configuration
docker run -d \
  --name go-vpn-server \
  --cap-add NET_ADMIN \
  --cap-add SYS_MODULE \
  -p 8443:8443 \
  -p 51820:51820/udp \
  -v ./config:/etc/vpn:ro \
  -v vpn-data:/var/lib/vpn \
  ghcr.io/november1306/go-vpn:latest
```

## Configuration

### Environment Variables
| Variable | Default | Description |
|----------|---------|-------------|
| `VPN_LISTEN_PORT` | `51820` | WireGuard UDP port |
| `VPN_API_PORT` | `8443` | HTTP API port |
| `VPN_SUBNET` | `10.0.0.0/24` | VPN client subnet |
| `VPN_DATA_DIR` | `/var/lib/vpn` | Data storage directory |
| `VPN_LOG_LEVEL` | `info` | Log level (debug, info, warn, error) |

### Volume Mounts
- `/etc/vpn` - Configuration files (read-only)
- `/var/lib/vpn` - Persistent data (keys, client configs)
- `/lib/modules` - Kernel modules (read-only, for WireGuard)

## Docker Compose Configuration

### Basic Setup
```yaml
version: '3.8'
services:
  vpn-server:
    image: ghcr.io/november1306/go-vpn:latest
    ports:
      - "8443:8443"
      - "51820:51820/udp"
    cap_add:
      - NET_ADMIN
      - SYS_MODULE
    volumes:
      - ./config:/etc/vpn:ro
      - vpn-data:/var/lib/vpn
    environment:
      - VPN_SUBNET=10.0.0.0/24
volumes:
  vpn-data:
```

### Production Setup with Monitoring
```yaml
version: '3.8'
services:
  vpn-server:
    image: ghcr.io/november1306/go-vpn:latest
    restart: unless-stopped
    ports:
      - "8443:8443"
      - "51820:51820/udp"
    cap_add:
      - NET_ADMIN
      - SYS_MODULE
    sysctls:
      - net.ipv4.ip_forward=1
    volumes:
      - ./config:/etc/vpn:ro
      - vpn-data:/var/lib/vpn
      - /lib/modules:/lib/modules:ro
    environment:
      - VPN_SUBNET=10.0.0.0/24
      - VPN_LOG_LEVEL=info
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:8443/health"]
      interval: 30s
      timeout: 5s
      retries: 3

volumes:
  vpn-data:
    driver: local
```

## Cloud Platform Deployment

### AWS ECS
```json
{
  "family": "go-vpn",
  "networkMode": "host",
  "requiresCompatibilities": ["EC2"],
  "containerDefinitions": [
    {
      "name": "vpn-server",
      "image": "ghcr.io/november1306/go-vpn:latest",
      "essential": true,
      "portMappings": [
        {"containerPort": 8443, "protocol": "tcp"},
        {"containerPort": 51820, "protocol": "udp"}
      ],
      "linuxParameters": {
        "capabilities": {
          "add": ["NET_ADMIN", "SYS_MODULE"]
        }
      }
    }
  ]
}
```

### Google Cloud Run
```yaml
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: go-vpn
  annotations:
    run.googleapis.com/execution-environment: gen2
spec:
  template:
    metadata:
      annotations:
        run.googleapis.com/container-dependencies: '{"vpn-server":[]}'
    spec:
      containers:
      - name: vpn-server
        image: ghcr.io/november1306/go-vpn:latest
        ports:
        - containerPort: 8443
        - containerPort: 51820
          protocol: UDP
        env:
        - name: VPN_SUBNET
          value: "10.0.0.0/24"
```

## Security Considerations

### Container Security
- Runs as non-root user (uid: 1001)
- Minimal Alpine base image
- No sensitive data in image layers
- Health checks enabled

### Network Capabilities
```bash
# Required capabilities for WireGuard
--cap-add NET_ADMIN    # Network interface management
--cap-add SYS_MODULE   # Kernel module loading

# Optional: More restrictive approach
--cap-drop ALL
--cap-add NET_ADMIN
--cap-add SYS_MODULE
```

### Firewall Rules
```bash
# Host firewall (iptables)
iptables -A INPUT -p tcp --dport 8443 -j ACCEPT
iptables -A INPUT -p udp --dport 51820 -j ACCEPT

# Docker published ports are automatically handled
```

## Troubleshooting

### Common Issues

#### Permission Denied
**Symptom**: `permission denied` when creating WireGuard interface
**Solution**: Ensure NET_ADMIN capability and privileged mode if needed
```bash
docker run --cap-add NET_ADMIN --cap-add SYS_MODULE ...
# or
docker run --privileged ...
```

#### Port Conflicts
**Symptom**: `port already in use`
**Solution**: Check for conflicting services
```bash
netstat -tulpn | grep :8443
netstat -tulpn | grep :51820
```

#### Module Not Found
**Symptom**: `wireguard module not found`
**Solution**: Ensure kernel modules are accessible
```bash
docker run -v /lib/modules:/lib/modules:ro ...
```

### Diagnostic Commands
```bash
# Check container status
docker ps
docker logs go-vpn-server

# Check network interfaces
docker exec go-vpn-server ip link show

# Test API endpoint
curl http://localhost:8443/health

# Monitor container resources
docker stats go-vpn-server
```

### Log Analysis
```bash
# Follow logs
docker-compose logs -f vpn-server

# Container health
docker inspect go-vpn-server | grep -A 10 "Health"

# System logs
journalctl -u docker -f | grep go-vpn
```

## Performance Tuning

### Resource Limits
```yaml
services:
  vpn-server:
    deploy:
      resources:
        limits:
          cpus: '1.0'
          memory: 512M
        reservations:
          cpus: '0.25'
          memory: 256M
```

### Kernel Parameters
```yaml
services:
  vpn-server:
    sysctls:
      - net.core.rmem_max=134217728
      - net.core.wmem_max=134217728
      - net.ipv4.ip_forward=1
      - net.ipv6.conf.all.forwarding=1
```

## Backup and Recovery

### Data Backup
```bash
# Backup persistent data
docker run --rm \
  -v go-vpn_vpn-data:/data \
  -v $(pwd):/backup \
  alpine tar czf /backup/vpn-backup.tar.gz /data
```

### Restore Data
```bash
# Restore from backup
docker run --rm \
  -v go-vpn_vpn-data:/data \
  -v $(pwd):/backup \
  alpine tar xzf /backup/vpn-backup.tar.gz -C /
```

## Monitoring

### Health Check Endpoint
```bash
curl http://localhost:8443/health
# Expected: {"status": "ok", "timestamp": "..."}
```

### Metrics Collection
```bash
# Basic stats
curl http://localhost:8443/metrics

# Prometheus format (if implemented)
curl http://localhost:8443/prometheus
```