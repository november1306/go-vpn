#!/bin/bash

# Go-VPN Server Environment Verification Script

set -euo pipefail

PASS=0
FAIL=0

pass() { echo "✓ $1"; ((PASS++)); }
fail() { echo "✗ $1"; ((FAIL++)); }

# Check Go installation
if command -v go >/dev/null; then
    pass "Go installed: $(go version)"
else
    fail "Go not found"
fi

# Check IP forwarding
if [[ $(cat /proc/sys/net/ipv4/ip_forward) == "1" ]]; then
    pass "IP forwarding enabled"
else
    fail "IP forwarding disabled"
fi

# Check iptables rules
INTERFACE=$(ip route | grep default | awk '{print $5}' | head -1)
if iptables -t nat -C POSTROUTING -s 10.0.0.0/24 -o "$INTERFACE" -j MASQUERADE 2>/dev/null; then
    pass "NAT rule configured"
else
    fail "NAT rule missing"
fi

if iptables -C FORWARD -i wg0 -j ACCEPT 2>/dev/null; then
    pass "Forward rules configured"
else
    fail "Forward rules missing"
fi

# Check firewall ports
if ufw status | grep -q "51820/udp"; then
    pass "VPN port allowed"
else
    fail "VPN port blocked"
fi

if ufw status | grep -q "8443/tcp"; then
    pass "API port allowed"
else
    fail "API port blocked"
fi

# Check connectivity
if ping -c 1 -W 2 8.8.8.8 >/dev/null 2>&1; then
    pass "Internet connectivity"
else
    fail "No internet"
fi

echo ""
if [[ $FAIL -eq 0 ]]; then
    echo "✓ All checks passed ($PASS/$((PASS+FAIL)))"
    echo "Ready to run: make run-server"
else
    echo "✗ $FAIL checks failed"
    exit 1
fi