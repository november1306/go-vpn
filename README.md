# GoWire VPN (MVP)

A simple, self-hosted WireGuard-based VPN written in Go. It consists of a server (control + data plane) and a CLI client for registration and connectivity.

Status: MVP in development. Expect breaking changes.

## Features (MVP)
- Single server location
- API-key authentication
- Basic peer management (add/remove)
- CLI: `register`, `connect`, `disconnect`, `status`
- File-based persistence (JSON)
- Structured logging

## Requirements
- Go 1.22+
- Admin privileges to configure network (Linux: `sudo`; Windows: Administrator)
- Open UDP port for WireGuard (default 51820)
- TLS certificate for API (self-signed allowed in dev)

## Install
```bash
# Clone
git clone https://github.com/<org>/go-vpn.git
cd go-vpn

# Build server and cli
make build
# or with Go
go build -o bin/server ./cmd/server
go build -o bin/vpn-cli ./cmd/vpn-cli
```

## Quickstart (Server)
1) Prepare network (Linux)
- Enable IP forwarding: `sudo sysctl -w net.ipv4.ip_forward=1`
- NAT on egress interface (e.g., `eth0`): `sudo iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE`
- Open WireGuard UDP port: `sudo ufw allow 51820/udp` (or equivalent)

2) TLS for API
- Dev: generate self-signed cert in `dev/certs/` (see docs)
- Prod: provide PEM cert and key

3) Configure and run
```bash
export GOVPN_LISTEN_ADDR=0.0.0.0:8443
export GOVPN_TLS_CERT=dev/certs/server.crt
export GOVPN_TLS_KEY=dev/certs/server.key
export GOVPN_WG_IFACE=wg0
export GOVPN_WG_LISTEN_PORT=51820
export GOVPN_VPN_CIDR=10.0.0.0/24
export GOVPN_DATA_DIR=./var

sudo ./bin/server
```

## Quickstart (Client)
```bash
# Register (generates local keys, requests API key, saves config)
./bin/vpn-cli register user@example.com --server https://<server>:8443

# Connect (requires elevated privileges to modify routes)
./bin/vpn-cli connect

# Status
./bin/vpn-cli status

# Disconnect
./bin/vpn-cli disconnect
```

Notes
- The client stores config at `~/.go-wire-vpn/config.json` with strict permissions.
- API key is included via `X-API-Key` header for authenticated requests.

## Configuration (Server)
Environment variables (prefix `GOVPN_`):
- `LISTEN_ADDR`: API listen address, e.g., `0.0.0.0:8443`
- `TLS_CERT`, `TLS_KEY`: PEM paths
- `WG_IFACE`: WireGuard interface name, e.g., `wg0`
- `WG_LISTEN_PORT`: UDP port, default `51820`
- `VPN_CIDR`: internal network, e.g., `10.0.0.0/24`
- `DATA_DIR`: directory for `users.json`, snapshots

Optional config file: `server.yaml` (env overrides file).

## Security
- API keys are high-entropy random bytes (base64url) stored hashed with bcrypt.
- TLS is mandatory; `--insecure` client mode is for dev only.
- Server and client may require admin privileges to configure networking.

## Troubleshooting
- Verify server UDP 51820 reachable from client
- Ensure NAT/masquerade configured on server egress
- Check `sysctl net.ipv4.ip_forward=1`
- Confirm TLS certificate trust or use `--insecure` in dev only

## Roadmap
- IPv6 support
- Key rotation/expiry
- Multi-server
- Database-backed storage

## License
TBD (see `tasks/Cross-7-license-and-readme.md`).

