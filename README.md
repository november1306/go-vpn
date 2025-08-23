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
- Go 1.21+ (for building)
- Docker (for integration testing)
- Make (for build automation)
- curl or wget (for downloading WinTun DLLs on Windows)

## Quick Start

```bash
# Clone the repository
git clone https://github.com/<org>/go-vpn.git
cd go-vpn

# Download WinTun DLLs (required for Windows VPN functionality)
make download-wintun

# Build both server and CLI
make build

# Or build individual components
make build-server  # builds bin/server
make build-cli     # builds bin/vpn-cli
```

## Try the Demo

**Live Demo Server:** Test client registration with our deployed server:

```bash
# Build CLI and test registration
make build-cli
./bin/vpn-cli register --server=https://go-vpn-production.up.railway.app
```

This will:
- Generate a new WireGuard key pair
- Exchange public keys with the server
- Demonstrate the working client-server communication

## Development

### Build Commands

```bash
# Download WinTun DLLs (Windows only, first time setup)
make download-wintun

# Build everything
make build

# Cross-platform builds for releases
make build-all

# View all available targets
make help
```

### Testing

The project uses a staged testing approach aligned with CI pipelines:

```bash
# Stage 1: Fast unit tests (no external dependencies)
make test-unit

# Stage 2: Integration tests (requires Docker)
make test-integration

# Stage 3: Docker container tests
make test-docker

# Run all test stages
make test-all
```

**Test Organization:**
- **Unit Tests**: Fast tests for individual components, run in any environment
- **Integration Tests**: End-to-end server startup and API testing via Docker
- **Docker Tests**: Container deployment validation

### Code Quality

```bash
# Format code
make fmt

# Run linter (uses golangci-lint if available, falls back to go vet)
make lint

# Clean build artifacts
make clean

# Clean everything including downloaded libraries
make clean-all
```

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



