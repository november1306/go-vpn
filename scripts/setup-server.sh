#!/bin/bash

# Go-VPN Server Environment Setup Script
# Sets up clean Ubuntu environment for running go-vpn server

set -euo pipefail

check_root() {
    if [[ $EUID -ne 0 ]]; then
        echo "Error: This script must be run as root"
        exit 1
    fi
}

install_dependencies() {
    echo "Installing dependencies..."
    
    # Set multiple non-interactive modes to avoid prompts
    export DEBIAN_FRONTEND=noninteractive
    export NEEDRESTART_MODE=a
    export NEEDRESTART_SUSPEND=1
    
    # Additional apt options for silent operation
    APT_OPTS="-y -qq -o Dpkg::Options::=--force-confold -o Dpkg::Options::=--force-confdef -o Dpkg::Use-Pty=0"
    
    apt update $APT_OPTS
    
    # Remove conflicting packages if they exist
    apt remove $APT_OPTS iptables-persistent netfilter-persistent 2>/dev/null || true
    
    # Upgrade with comprehensive silent options
    apt upgrade $APT_OPTS
    
    # Install dependencies with silent options
    # Use yes pipe as fallback for any remaining prompts
    yes '' | apt install $APT_OPTS curl wget git build-essential ufw make || \
    apt install $APT_OPTS curl wget git build-essential ufw make
    
    # Reset environment
    unset DEBIAN_FRONTEND NEEDRESTART_MODE NEEDRESTART_SUSPEND
}

install_go() {
    echo "Installing Go..."
    
    # Check if Go is already installed
    if command -v go >/dev/null 2>&1; then
        echo "✓ Go already installed: $(go version)"
    else
        # Install Go using apt (much simpler)
        apt install -y golang-go
        echo "✓ Go installed: $(go version)"
    fi
}

configure_network() {
    echo "Configuring network..."
    
    # 1. Enable IP forwarding (idempotent)
    echo 1 > /proc/sys/net/ipv4/ip_forward
    if ! grep -q "net.ipv4.ip_forward=1" /etc/sysctl.conf; then
        echo 'net.ipv4.ip_forward=1' >> /etc/sysctl.conf
    fi
    
    # 2. Find public interface
    INTERFACE=$(ip route | grep default | awk '{print $5}' | head -1)
    echo "Using interface: $INTERFACE"
    
    # 3. Add NAT masquerading for VPN subnet (check if exists first)
    if ! iptables -t nat -C POSTROUTING -s 10.0.0.0/24 -o "$INTERFACE" -j MASQUERADE 2>/dev/null; then
        iptables -t nat -A POSTROUTING -s 10.0.0.0/24 -o "$INTERFACE" -j MASQUERADE
        echo "✓ NAT rule added"
    else
        echo "✓ NAT rule already exists"
    fi
    
    # 4. Allow VPN traffic forwarding (check if exists first)
    if ! iptables -C FORWARD -i wg0 -j ACCEPT 2>/dev/null; then
        iptables -A FORWARD -i wg0 -j ACCEPT
        echo "✓ Forward rule (wg0 in) added"
    else
        echo "✓ Forward rule (wg0 in) already exists"
    fi
    
    if ! iptables -C FORWARD -o wg0 -j ACCEPT 2>/dev/null; then
        iptables -A FORWARD -o wg0 -j ACCEPT
        echo "✓ Forward rule (wg0 out) added"
    else
        echo "✓ Forward rule (wg0 out) already exists"
    fi
    
    # 5. Allow VPN ports (UFW handles duplicates gracefully)
    ufw allow 51820/udp
    ufw allow 8443/tcp
    ufw --force enable
    
    # 6. UFW will persist rules automatically
}

setup_service() {
    echo "Setting up systemd service..."
    
    # Create service directories (idempotent)
    mkdir -p /root/go-vpn/{bin,data}
    
    # Install service file if it exists
    if [[ -f "scripts/go-vpn.service" ]]; then
        # Only copy if different or doesn't exist
        if ! cmp -s "scripts/go-vpn.service" "/etc/systemd/system/go-vpn.service" 2>/dev/null; then
            cp scripts/go-vpn.service /etc/systemd/system/
            systemctl daemon-reload
            echo "✓ Systemd service file updated"
        else
            echo "✓ Systemd service file already up to date"
        fi
        
        # Enable service (idempotent)
        if ! systemctl is-enabled go-vpn >/dev/null 2>&1; then
            systemctl enable go-vpn
            echo "✓ Systemd service enabled"
        else
            echo "✓ Systemd service already enabled"
        fi
    else
        echo "⚠ Service file not found, manual service setup required"
    fi
}

main() {
    echo "Setting up Go-VPN server environment..."
    check_root
    install_dependencies
    install_go
    configure_network
    setup_service
    
    echo ""
    echo "✓ Setup complete!"
    echo ""
    echo "Next steps:"
    echo "1. Build server: make build-server"
    echo "2. Copy binary: cp bin/server /root/go-vpn/bin/"
    echo "3. Start service: systemctl start go-vpn"
    echo "4. Check status: systemctl status go-vpn"
}

main "$@"