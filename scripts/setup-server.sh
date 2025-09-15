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
    apt update && apt upgrade -y
    apt install -y curl wget git build-essential iptables-persistent ufw
}

install_go() {
    echo "Installing Go..."
    echo "Architecture: $(uname -m)"
    rm -rf /usr/local/go
    cd /tmp
    wget -q "https://golang.org/dl/go1.24.6.linux-amd64.tar.gz"
    tar -C /usr/local -xzf "go1.24.6.linux-amd64.tar.gz"
    echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
    export PATH="/usr/local/go/bin:$PATH"
    /usr/local/go/bin/go version
}

configure_network() {
    echo "Configuring network..."
    
    # 1. Enable IP forwarding
    echo 1 > /proc/sys/net/ipv4/ip_forward
    echo 'net.ipv4.ip_forward=1' >> /etc/sysctl.conf
    
    # 2. Find public interface
    INTERFACE=$(ip route | grep default | awk '{print $5}' | head -1)
    echo "Using interface: $INTERFACE"
    
    # 3. Add NAT masquerading for VPN subnet
    iptables -t nat -A POSTROUTING -s 10.0.0.0/24 -o $INTERFACE -j MASQUERADE
    
    # 4. Allow VPN traffic forwarding
    iptables -A FORWARD -i wg0 -j ACCEPT
    iptables -A FORWARD -o wg0 -j ACCEPT
    
    # 5. Allow VPN ports
    ufw allow 51820/udp
    ufw allow 8443/tcp
    ufw --force enable
    
    # 6. Save rules
    iptables-save > /etc/iptables/rules.v4
}

setup_service() {
    echo "Setting up systemd service..."
    
    # Create service directories
    mkdir -p /root/go-vpn/{bin,data}
    
    # Install service file if it exists
    if [[ -f "scripts/go-vpn.service" ]]; then
        cp scripts/go-vpn.service /etc/systemd/system/
        systemctl daemon-reload
        systemctl enable go-vpn
        echo "✓ Systemd service installed and enabled"
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