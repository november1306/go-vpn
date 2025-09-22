#!/bin/bash

# Go-VPN Server Environment Verification Script

set -uo pipefail

PASS=0
FAIL=0

pass() { echo "✓ $1"; ((PASS++)); }
fail() { echo "✗ $1"; ((FAIL++)); return 0; }

# Function to safely check commands with timeout
safe_check() {
    local cmd="$1"
    local description="$2"
    
    if timeout 5 bash -c "$cmd" >/dev/null 2>&1; then
        pass "$description"
    else
        fail "$description"
    fi
}

# Check Go installation
echo "Checking Go installation..."
if command -v go >/dev/null 2>&1; then
    GO_VERSION=$(timeout 10 go version 2>/dev/null || echo "timeout")
    if [[ "$GO_VERSION" != "timeout" ]]; then
        pass "Go installed: $GO_VERSION"
    else
        fail "Go command timeout"
    fi
else
    fail "Go not found"
fi

# Check IP forwarding
echo "Checking IP forwarding..."
if [[ -r /proc/sys/net/ipv4/ip_forward ]]; then
    IP_FORWARD=$(cat /proc/sys/net/ipv4/ip_forward 2>/dev/null || echo "0")
    if [[ "$IP_FORWARD" == "1" ]]; then
        pass "IP forwarding enabled"
    else
        fail "IP forwarding disabled (current: $IP_FORWARD)"
    fi
else
    fail "Cannot read IP forwarding status"
fi

# Check iptables rules
echo "Checking iptables rules..."
INTERFACE=$(ip route | grep default | awk '{print $5}' | head -1 2>/dev/null || echo "unknown")
if [[ "$INTERFACE" != "unknown" ]]; then
    echo "Using interface: $INTERFACE"
    
    # Check NAT rule with timeout
    if timeout 5 iptables -t nat -C POSTROUTING -s 10.0.0.0/24 -o "$INTERFACE" -j MASQUERADE 2>/dev/null; then
        pass "NAT rule configured"
    else
        fail "NAT rule missing for interface $INTERFACE"
    fi
    
    # Check forward rules with timeout
    if timeout 5 iptables -C FORWARD -i wg0 -j ACCEPT 2>/dev/null; then
        pass "Forward rules configured"
    else
        fail "Forward rules missing"
    fi
else
    fail "Cannot determine default network interface"
fi

# Check firewall ports
echo "Checking UFW firewall..."
UFW_STATUS=$(timeout 5 ufw status 2>/dev/null || echo "error")
if [[ "$UFW_STATUS" == "error" ]]; then
    fail "UFW status check failed"
elif echo "$UFW_STATUS" | grep -q "Status: inactive"; then
    fail "UFW firewall is inactive"
elif echo "$UFW_STATUS" | grep -q "51820/udp"; then
    pass "VPN port (51820/udp) allowed"
    
    if echo "$UFW_STATUS" | grep -q "8443/tcp"; then
        pass "API port (8443/tcp) allowed"
    else
        fail "API port (8443/tcp) blocked"
    fi
else
    fail "VPN port (51820/udp) blocked"
fi

# Check connectivity
echo "Checking internet connectivity..."
safe_check "ping -c 1 -W 2 8.8.8.8" "Internet connectivity"

echo ""
echo "=== Verification Summary ==="
if [[ $FAIL -eq 0 ]]; then
    echo "✓ All checks passed ($PASS/$((PASS+FAIL)))"
    echo "Ready to run: make build-server && make deploy"
else
    echo "✗ $FAIL checks failed, $PASS passed"
    echo ""
    echo "To fix issues, run: sudo ./scripts/setup-server.sh"
    exit 1
fi