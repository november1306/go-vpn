# Windows Deployment Guide: GoWire VPN

## Prerequisites

### System Requirements
- **OS**: Windows 10/11 (64-bit)
- **RAM**: Minimum 4GB, recommended 8GB
- **Storage**: 100MB free space
- **Network**: Static IP recommended for server
- **Privileges**: Administrator access required

### Dependencies
- Windows 10 SDK (for TUN device support)
- Visual C++ Redistributable (if using CGO)
- PowerShell 5.1+ or PowerShell Core 7+

## Installation Steps

### 1. Download Binaries
```powershell
# Create installation directory
New-Item -Path "C:\Program Files\GoWire" -ItemType Directory -Force

# Download from releases (when available)
# For now, build from source:
cd C:\workspace\go-vpn
go build -o "C:\Program Files\GoWire\server.exe" ./cmd/server
go build -o "C:\Program Files\GoWire\vpn-cli.exe" ./cmd/vpn-cli
```

### 2. Generate TLS Certificates
```powershell
# Navigate to installation directory
cd "C:\Program Files\GoWire"

# Create certificates directory
New-Item -Path "certs" -ItemType Directory -Force

# Generate self-signed certificate (development)
# Install OpenSSL or use PowerShell alternative:
$cert = New-SelfSignedCertificate -DnsName "localhost","127.0.0.1" -CertStoreLocation "cert:\LocalMachine\My" -KeyAlgorithm RSA -KeyLength 2048 -NotAfter (Get-Date).AddYears(1)

# Export certificate and key
$certPath = "certs\server.crt"
$keyPath = "certs\server.key"
Export-Certificate -Cert $cert -FilePath $certPath -Type CERT
# Note: Key export requires additional steps in PowerShell
```

### 3. Create Server Configuration
```powershell
# Create server.yaml
@"
listen_port: 51820
api_port: 8443
subnet: "10.0.0.0/24"
data_dir: "C:\ProgramData\GoWire"
log_level: "info"
tls:
  cert_file: "certs\server.crt"  
  key_file: "certs\server.key"
dns:
  - "1.1.1.1"
  - "1.0.0.1"
"@ | Out-File -FilePath "server.yaml" -Encoding UTF8
```

### 4. Create Data Directory
```powershell
# Create application data directory
New-Item -Path "C:\ProgramData\GoWire" -ItemType Directory -Force
New-Item -Path "C:\ProgramData\GoWire\logs" -ItemType Directory -Force

# Set appropriate permissions (restrict access)
icacls "C:\ProgramData\GoWire" /grant "Administrators:(OI)(CI)F" /inheritance:r
icacls "C:\ProgramData\GoWire" /grant "SYSTEM:(OI)(CI)F"
```

### 5. Configure Windows Firewall
```powershell
# Open firewall ports
New-NetFirewallRule -DisplayName "GoWire VPN API" -Direction Inbound -Protocol TCP -LocalPort 8443 -Action Allow
New-NetFirewallRule -DisplayName "GoWire VPN WireGuard" -Direction Inbound -Protocol UDP -LocalPort 51820 -Action Allow

# Allow outbound for VPN traffic
New-NetFirewallRule -DisplayName "GoWire VPN Outbound" -Direction Outbound -Protocol Any -Action Allow
```

## Running the Server

### Manual Start (Development)
```powershell
# Start server as Administrator
cd "C:\Program Files\GoWire"
.\server.exe

# Server will start and display:
# 2023/08/09 10:30:00 INFO Server starting port=8443
# 2023/08/09 10:30:00 INFO WireGuard interface created interface=wg-tun0
```

### Windows Service Installation (Production)

#### Create Service Wrapper Script
```powershell
# Create service-wrapper.ps1
@"
`$ErrorActionPreference = "Stop"
Set-Location "C:\Program Files\GoWire"
& ".\server.exe" 2>&1 | Tee-Object -FilePath "C:\ProgramData\GoWire\logs\service.log"
"@ | Out-File -FilePath "service-wrapper.ps1" -Encoding UTF8
```

#### Install Service
```powershell
# Using NSSM (Non-Sucking Service Manager) - download from nssm.cc
# Or use built-in sc command:

sc.exe create "GoWire VPN" binPath="powershell.exe -ExecutionPolicy Bypass -File \"C:\Program Files\GoWire\service-wrapper.ps1\"" start=auto
sc.exe description "GoWire VPN" "GoWire VPN Server Service"

# Start service  
Start-Service "GoWire VPN"
```

## Client Setup

### 1. Install CLI Tool
```powershell
# Add to PATH or create alias
$env:PATH += ";C:\Program Files\GoWire"

# Or copy to user directory
Copy-Item "C:\Program Files\GoWire\vpn-cli.exe" "$env:USERPROFILE\AppData\Local\Microsoft\WindowsApps\"
```

### 2. Client Registration
```powershell
# Register new user (requires server to be running)
vpn-cli register user@example.com

# Output:
# Registration successful!
# API key saved to: C:\Users\username\.go-wire-vpn\config.json
# You can now connect using: vpn-cli connect
```

### 3. VPN Connection
```powershell
# Connect to VPN (requires Administrator)
# Right-click PowerShell -> "Run as Administrator"
vpn-cli connect

# Check status
vpn-cli status

# Disconnect when done
vpn-cli disconnect
```

## Network Configuration Details

### Required Windows Capabilities
- **TUN Device Creation**: Handled by `wireguard-go`
- **Route Manipulation**: Uses `netsh` commands
- **DNS Configuration**: Optional, via `netsh` or registry

### Network Commands Used
```powershell
# Interface creation (handled by wireguard-go internally)
# Route addition
netsh interface ip add route 0.0.0.0/0 "wg-tun0" 10.0.0.1

# DNS configuration (optional)
netsh interface ip set dns "wg-tun0" static 1.1.1.1
```

## Troubleshooting

### Common Issues

#### "Access Denied" Errors
- **Cause**: Not running as Administrator
- **Solution**: Right-click terminal -> "Run as Administrator"

#### "Port Already in Use"
- **Cause**: Another service using port 8443 or 51820
- **Solution**: Check with `netstat -an | findstr :8443` and stop conflicting services

#### "Certificate Not Found"
- **Cause**: Missing or invalid TLS certificate
- **Solution**: Regenerate certificates following step 2

#### "WireGuard Interface Creation Failed"  
- **Cause**: TUN adapter driver issues
- **Solution**: Install/update Windows TUN driver

### Diagnostic Commands
```powershell
# Check server status
Get-Process -Name "server" -ErrorAction SilentlyContinue

# Check service status
Get-Service "GoWire VPN"

# Check firewall rules
Get-NetFirewallRule -DisplayName "*GoWire*"

# Check network interfaces
Get-NetAdapter | Where-Object {$_.Name -like "*wg*"}

# View logs
Get-Content "C:\ProgramData\GoWire\logs\service.log" -Tail 50
```

### Log Locations
- **Server Logs**: `C:\ProgramData\GoWire\logs\`
- **Client Config**: `C:\Users\{username}\.go-wire-vpn\`
- **Event Log**: Windows Event Viewer -> Application

## Security Considerations

### File Permissions
- Server config: Readable by Administrators only
- User configs: Readable by user only (`0600` equivalent)
- Private keys: Never world-readable

### Network Security
- Server should be behind firewall with only required ports open
- Consider VPN server on dedicated network segment
- Regular security updates for Windows and Go runtime

### Certificate Management
- Replace self-signed certificates with CA-issued for production
- Implement certificate rotation
- Monitor certificate expiration

## Uninstallation

### Remove Service
```powershell
Stop-Service "GoWire VPN"
sc.exe delete "GoWire VPN"
```

### Remove Files
```powershell
Remove-Item "C:\Program Files\GoWire" -Recurse -Force
Remove-Item "C:\ProgramData\GoWire" -Recurse -Force
Remove-Item "$env:USERPROFILE\.go-wire-vpn" -Recurse -Force
```

### Remove Firewall Rules
```powershell
Remove-NetFirewallRule -DisplayName "*GoWire*"
```