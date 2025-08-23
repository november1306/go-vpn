# Windows Client Setup Guide

## Prerequisites for Windows VPN Client

### 1. WinTUN Driver Installation

**Option A: Install WireGuard Client (Recommended)**
1. Download WireGuard for Windows: https://www.wireguard.com/install/
2. Run installer as Administrator
3. This automatically installs WinTUN driver system-wide

**Option B: Manual WinTUN Installation**
1. Download `wintun.dll` from https://www.wintun.net/
2. Place in same directory as `vpn-cli.exe`
3. Requires Administrator privileges for TUN interface creation

### 2. Administrator Privileges

VPN client requires Administrator privileges for:
- Creating TUN network interfaces
- Modifying routing tables
- Managing network adapters

**Setup:**
1. Right-click Command Prompt → "Run as administrator"
2. Navigate to vpn-cli directory
3. Run VPN commands from elevated prompt

## Quick Start Guide

### Step 1: Build Client
```cmd
# From project root (as Administrator)
go build -o vpn-cli.exe cmd/vpn-cli/main.go
```

### Step 2: Register with Server
```cmd
# Replace with your Railway server URL
vpn-cli.exe register --server https://go-vpn-production.up.railway.app

# Expected output:
# 🔐 Client Registration Demo
# Generating client key pair...
# ✅ Client Public Key: k7vHfX4Km9f8d3wq5N8F2g5z7m1P3Q6r9s2B5x8C4A=
# 📡 Registering with server: https://go-vpn-production.up.railway.app
# ✅ Registration successful - VPN tunnel established
# 📋 Server Public Key: m8wHgY5Lm0g9e4xr6O9G3h6z8n2Q4R7t0u3C6y9D5B=
```

### Step 3: Connect VPN (Coming Soon)
```cmd
# This will establish the VPN tunnel
vpn-cli.exe connect

# Expected behavior:
# - Creates WireGuard interface
# - Routes all traffic through VPN
# - Masks your real IP address
```

### Step 4: Verify IP Masking
```cmd
# Check current public IP
curl ifconfig.me

# Should show Railway server IP, not your local IP
```

### Step 5: Disconnect VPN
```cmd
# Cleanly shutdown VPN tunnel
vpn-cli.exe disconnect

# Expected behavior:
# - Removes WireGuard interface
# - Restores original routing
# - Returns to direct internet access
```

## Windows-Specific Configuration

### Network Interface Management
Windows VPN client will:
- Create WireGuard TUN adapter (requires WinTUN driver)
- Assign VPN IP address (e.g., `10.0.0.100`)
- Add routes through VPN interface
- Manage DNS settings (optional)

### Firewall Configuration
Windows Firewall may block VPN traffic:
1. Open Windows Defender Firewall
2. Allow `vpn-cli.exe` through firewall
3. Or temporarily disable for testing: `netsh advfirewall set allprofiles state off`
4. Re-enable after testing: `netsh advfirewall set allprofiles state on`

### Network Commands (Advanced)
Manual network setup (if needed):
```cmd
# View network interfaces
netsh interface show interface

# Check routing table
route print

# View WireGuard interface (after connect)
netsh interface show interface name="wg-client"
```

## Troubleshooting

### Common Windows Issues

**1. "Access Denied" / Permission Errors**
- **Solution**: Run Command Prompt as Administrator
- **Why**: TUN interface creation requires elevated privileges

**2. "wintun.dll not found" Error**
- **Solution**: Install WireGuard for Windows or place `wintun.dll` in app directory
- **Why**: Windows requires WinTUN driver for TUN interfaces

**3. "Unable to create network adapter"**
- **Solution**: 
  1. Check Administrator privileges
  2. Verify WinTUN driver installed
  3. Disable antivirus temporarily
  4. Check Windows version (Windows 7+ required)

**4. VPN Connects But No Internet**
- **Solution**:
  1. Check DNS settings: `nslookup google.com`
  2. Verify routing: `route print`
  3. Test direct connection: `ping 8.8.8.8`
  4. Check Windows Firewall settings

**5. Slow Performance**
- **Expected**: Userspace WireGuard has performance overhead
- **Typical**: 100-500 Mbps on modern hardware
- **Improvement**: Use kernel WireGuard (future enhancement)

### Debug Commands
```cmd
# Check WireGuard status (after implementing connect)
# This will show interface status, peer info, transfer stats

# View client logs
vpn-cli.exe status --verbose

# Test connectivity through VPN
ping 8.8.8.8
nslookup google.com
curl -v https://httpbin.org/ip
```

## Windows Version Compatibility

**Supported Versions:**
- ✅ Windows 11 (all editions)
- ✅ Windows 10 (version 1803+)
- ✅ Windows Server 2019/2022
- ⚠️ Windows 8.1 (limited testing)
- ❌ Windows 7 (WinTUN not supported)

## Security Considerations

### Privilege Requirements
- Administrator required for network interface management
- Consider UAC elevation prompts
- Alternative: Use Windows service for persistent VPN

### Data Protection
- Client private key stored in `%USERPROFILE%\.go-vpn\config.json`
- File permissions set to user-only access
- No sensitive data in logs or console output

### Network Security
- All traffic encrypted with WireGuard protocol
- Perfect forward secrecy with key rotation
- No traffic leaks during connect/disconnect transitions

## Configuration File Location

Windows client stores configuration at:
```
C:\Users\{username}\.go-vpn\config.json
```

**Contains:**
- Client private key (encrypted storage planned)
- Server public key and endpoint
- Assigned VPN IP address
- Connection settings

**Security:**
- Hidden file attribute set
- NTFS ACL restricts access to user only
- Future: Windows DPAPI encryption planned

## Deployment Options

### 1. Standalone Executable
- Single `vpn-cli.exe` file
- Include `wintun.dll` in same directory
- Distribute as ZIP archive

### 2. MSI Installer (Future)
- Windows Installer package
- Automatic WinTUN driver installation
- Start Menu shortcuts
- Uninstaller included

### 3. Windows Service (Future)
- Background VPN service
- Auto-connect on startup
- System tray management GUI
- No Administrator prompt per connection

## Next Steps for Implementation

**Phase 1: Core VPN Client (Current)**
1. ✅ Registration command working
2. 🔄 Implement `connect` command
3. 🔄 Implement `disconnect` command  
4. 🔄 Windows networking integration

**Phase 2: Windows Polish**
1. MSI installer creation
2. Windows service implementation
3. GUI management interface
4. Auto-update mechanism

**Phase 3: Enterprise Features**
1. Group Policy support
2. Certificate-based authentication
3. Centralized configuration management
4. Audit logging