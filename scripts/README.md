# GoWire VPN Deployment Scripts

Automated deployment and maintenance scripts for GoWire VPN server.

## Quick Start

### 1. Initial Server Setup

```bash
# On fresh Ubuntu server (as root)
curl -sSL https://raw.githubusercontent.com/your-repo/go-vpn/main/scripts/setup-server.sh | bash
```

Or manually:
```bash
git clone https://github.com/your-repo/go-vpn.git
cd go-vpn
sudo ./scripts/setup-server.sh
```

### 2. Build and Deploy

```bash
# Build the server
make build-server

# Deploy to production
sudo ./scripts/update-server.sh update
```

### 3. Verify Installation

```bash
sudo ./scripts/verify-server.sh
```

## Scripts Overview

### `setup-server.sh`
- **Purpose**: Initial server environment setup
- **What it does**:
  - Installs Go 1.24.6 and build dependencies
  - Configures IP forwarding and iptables NAT rules
  - Sets up UFW firewall (ports 8443, 51820)
  - Installs systemd service
- **Usage**: `sudo ./setup-server.sh`
- **Requirements**: Fresh Ubuntu server with root access

### `verify-server.sh`
- **Purpose**: Environment validation and health checks
- **What it checks**:
  - Go installation and version
  - IP forwarding configuration
  - iptables NAT and forwarding rules
  - Firewall port configuration
  - Internet connectivity
- **Usage**: `sudo ./verify-server.sh`
- **Exit codes**: 0 = all checks pass, 1 = failures detected

### `backup-restore.sh`
- **Purpose**: Peer data backup and restore operations
- **Operations**:
  - `backup`: Create timestamped backup of peer data
  - `restore`: Restore from latest backup
  - `list`: Show available backups
- **Usage**: `sudo ./backup-restore.sh {backup|restore|list}`
- **Storage**: `/var/backups/go-vpn/`

### `update-server.sh`
- **Purpose**: Zero-downtime server updates and rollbacks
- **Operations**:
  - `update`: Deploy new server binary with automatic rollback on failure
  - `rollback`: Restore previous version
- **Usage**: `sudo ./update-server.sh {update|rollback}`
- **Safety**: Automatic backup before updates

### `go-vpn.service`
- **Purpose**: Systemd service configuration
- **Features**:
  - Automatic restart on failure
  - Security hardening (NoNewPrivileges, ProtectSystem)
  - Proper logging to systemd journal
  - Environment variable configuration

## Deployment Workflow

### Production Deployment

1. **Initial Setup** (one-time):
   ```bash
   # On DigitalOcean droplet or Linux VPS
   sudo ./scripts/setup-server.sh
   sudo ./scripts/verify-server.sh
   ```

2. **Build and Deploy**:
   ```bash
   # On development machine
   make build-server
   scp bin/server root@your-server:/tmp/
   
   # On server
   sudo cp /tmp/server /root/go-vpn/bin/
   sudo systemctl start go-vpn
   sudo systemctl status go-vpn
   ```

3. **Verify Deployment**:
   ```bash
   # Check service status
   sudo systemctl status go-vpn
   
   # Check logs
   sudo journalctl -u go-vpn -f
   
   # Test API
   curl http://localhost:8443/health
   ```

### Updates and Maintenance

1. **Regular Backups**:
   ```bash
   # Create backup before updates
   sudo ./scripts/backup-restore.sh backup
   ```

2. **Server Updates**:
   ```bash
   # Build new version
   make build-server
   
   # Deploy with automatic rollback on failure
   sudo ./scripts/update-server.sh update
   ```

3. **Rollback if Needed**:
   ```bash
   sudo ./scripts/update-server.sh rollback
   ```

## Security Considerations

- All scripts require root privileges for network configuration
- Service runs as root (required for WireGuard interface management)
- Firewall configured to allow only necessary ports
- Automatic backup before updates prevents data loss
- Service hardening through systemd security features

## Troubleshooting

### Common Issues

1. **Service won't start**:
   ```bash
   sudo journalctl -u go-vpn -n 50
   sudo ./scripts/verify-server.sh
   ```

2. **Network connectivity issues**:
   ```bash
   # Check iptables rules
   sudo iptables -t nat -L
   sudo iptables -L FORWARD
   
   # Verify IP forwarding
   cat /proc/sys/net/ipv4/ip_forward
   ```

3. **Firewall blocking connections**:
   ```bash
   sudo ufw status verbose
   sudo ufw allow 8443/tcp
   sudo ufw allow 51820/udp
   ```

### Log Analysis

```bash
# Real-time logs
sudo journalctl -u go-vpn -f

# Recent errors
sudo journalctl -u go-vpn -p err -n 20

# Service restart history
sudo journalctl -u go-vpn | grep "Started\|Stopped"
```

## Integration with Existing Infrastructure

These scripts are designed to work with:
- DigitalOcean droplets
- Generic Linux VPS providers
- Docker containers (with privileged mode)
- Existing Ubuntu/Debian servers

For other platforms, modify the package installation commands in `setup-server.sh`.