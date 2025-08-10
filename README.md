# GoWire VPN

A simple, self-hosted WireGuard-based VPN server and CLI client written in Go.

**⚠️ Status: MVP in development - expect breaking changes**

## What is GoWire VPN?

GoWire VPN consists of:
- **Server**: HTTP API for peer management + WireGuard data plane
- **CLI Client**: Register, connect, disconnect, and check status

## Current Features

✅ **Working Demo:**
- WireGuard key generation and exchange
- Client-server registration with key exchange
- CLI `register` command with live server
- Deployed demo server on Railway


## Requirements
- Go 1.21+ (for building the CLI client)

## Quick Start

```bash
# Clone and build
git clone https://github.com/<org>/go-vpn.git
cd go-vpn
go build -o bin/vpn-cli ./cmd/vpn-cli
```

## Try the Demo

**Live Demo Server:** Test client registration with our deployed server:

```bash
# Build and test registration
go build -o bin/vpn-cli ./cmd/vpn-cli
./bin/vpn-cli register --server=https://go-vpn-production.up.railway.app
```

This will:
- Generate a new WireGuard key pair
- Exchange public keys with the server
- Demonstrate the working client-server communication

## Development Status

This is an early-stage project demonstrating WireGuard key exchange. The full VPN functionality (connect/disconnect) is not yet implemented.

## Roadmap to MVP
- API key authentication
- IP allocation for clients
- Client config storage
- CLI commands: `connect`, `disconnect`, `status`
- Full WireGuard tunnel establishment

## License
AGPL-3



