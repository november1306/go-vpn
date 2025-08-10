package vpnserver

import (
	"context"
)

// PeerInfo contains information about a connected peer
type PeerInfo struct {
	PublicKey  string
	AllowedIPs []string
	Endpoint   string
	LastSeen   int64 // Unix timestamp
	RxBytes    int64
	TxBytes    int64
}

// ServerConfig contains configuration for the VPN server
type ServerConfig struct {
	// Interface name (e.g., "wg0")
	InterfaceName string
	
	// Server private key (base64 encoded)
	PrivateKey string
	
	// Listen port for WireGuard UDP traffic
	ListenPort int
	
	// Server IP within the VPN network (e.g., "10.0.0.1/24")
	ServerIP string
}

// WireGuardBackend defines the interface for different WireGuard implementations
// This abstraction allows switching between userspace, kernel, and other backends
// for scalability without changing the VPN server logic
type WireGuardBackend interface {
	// Start initializes and starts the WireGuard device with the given configuration
	Start(ctx context.Context, config ServerConfig) error
	
	// Stop gracefully shuts down the WireGuard device
	Stop(ctx context.Context) error
	
	// AddPeer adds a new peer to the WireGuard device
	// publicKey: base64-encoded peer public key
	// allowedIPs: CIDR blocks that the peer is allowed to send traffic for
	AddPeer(publicKey string, allowedIPs []string) error
	
	// RemovePeer removes a peer from the WireGuard device
	RemovePeer(publicKey string) error
	
	// GetPeers returns information about all connected peers
	GetPeers() ([]PeerInfo, error)
	
	// IsRunning returns whether the backend is currently running
	IsRunning() bool
}