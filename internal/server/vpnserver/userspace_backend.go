package vpnserver

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log/slog"
	"sync"

	"github.com/november1306/go-vpn/internal/wireguard"
)

// UserspaceBackend implements WireGuardBackend using wireguard-go userspace implementation
// This provides cross-platform support and easy deployment, suitable for MVP and up to ~500 users
type UserspaceBackend struct {
	mu      sync.RWMutex
	device  *wireguard.WireGuardDevice
	config  ServerConfig
	running bool
	peers   map[string][]string // publicKey -> allowedIPs mapping for tracking
}

// NewUserspaceBackend creates a new userspace WireGuard backend
func NewUserspaceBackend() *UserspaceBackend {
	return &UserspaceBackend{
		peers: make(map[string][]string),
	}
}

// Start initializes and starts the userspace WireGuard device
func (ub *UserspaceBackend) Start(ctx context.Context, config ServerConfig) error {
	ub.mu.Lock()
	defer ub.mu.Unlock()

	if ub.running {
		return fmt.Errorf("backend already running")
	}

	slog.Info("Starting userspace WireGuard backend", "interface", config.InterfaceName, "port", config.ListenPort)

	// Create WireGuard device using existing foundation
	device, err := wireguard.NewWireGuardDevice(config.InterfaceName)
	if err != nil {
		return fmt.Errorf("failed to create WireGuard device: %w", err)
	}

	// Set device before configuring so IPC calls work
	ub.device = device

	// Configure the device with server settings
	if err := ub.configureDevice(config); err != nil {
		device.Stop()   // Clean up on error
		ub.device = nil // Reset on error
		return fmt.Errorf("failed to configure device: %w", err)
	}

	// Start the device
	if err := device.Start(); err != nil {
		device.Stop() // Clean up on error
		return fmt.Errorf("failed to start device: %w", err)
	}

	ub.device = device
	ub.config = config
	ub.running = true

	slog.Info("Userspace WireGuard backend started successfully", "interface", config.InterfaceName)
	return nil
}

// Stop gracefully shuts down the userspace WireGuard device
func (ub *UserspaceBackend) Stop(ctx context.Context) error {
	ub.mu.Lock()
	defer ub.mu.Unlock()

	if !ub.running {
		return nil // Already stopped
	}

	slog.Info("Stopping userspace WireGuard backend", "interface", ub.config.InterfaceName)

	if ub.device != nil {
		if err := ub.device.Stop(); err != nil {
			slog.Error("Error stopping WireGuard device", "error", err)
			// Continue with cleanup even if stop fails
		}
		ub.device = nil
	}

	ub.running = false
	ub.peers = make(map[string][]string) // Clear peer tracking

	slog.Info("Userspace WireGuard backend stopped")
	return nil
}

// AddPeer adds a new peer to the WireGuard device
func (ub *UserspaceBackend) AddPeer(publicKey string, allowedIPs []string) error {
	ub.mu.Lock()
	defer ub.mu.Unlock()

	if !ub.running {
		return fmt.Errorf("backend not running")
	}

	slog.Info("Adding peer to userspace backend", "allowedIPs", allowedIPs)

	// Convert base64 public key to hex for WireGuard IPC
	hexPublicKey, err := ub.base64ToHex(publicKey)
	if err != nil {
		return fmt.Errorf("invalid public key format: %w", err)
	}

	// Build IPC configuration string to add peer  
	// WireGuard UAPI format: public_key=<hex_key>\nallowed_ip=<ip>\n\n
	config := fmt.Sprintf("public_key=%s\n", hexPublicKey)

	for _, ip := range allowedIPs {
		config += fmt.Sprintf("allowed_ip=%s\n", ip)
	}
	config += "\n"

	// Apply configuration via IPC (this is how wireguard-go accepts peer config)
	if err := ub.applyIPCConfig(config); err != nil {
		return fmt.Errorf("failed to add peer via IPC: %w", err)
	}

	// Track peer for management
	ub.peers[publicKey] = allowedIPs

	slog.Info("Peer added successfully", "peerCount", len(ub.peers))
	return nil
}

// RemovePeer removes a peer from the WireGuard device
func (ub *UserspaceBackend) RemovePeer(publicKey string) error {
	ub.mu.Lock()
	defer ub.mu.Unlock()

	if !ub.running {
		return fmt.Errorf("backend not running")
	}

	slog.Info("Removing peer from userspace backend")

	// Convert base64 public key to hex for WireGuard IPC
	hexPublicKey, err := ub.base64ToHex(publicKey)
	if err != nil {
		return fmt.Errorf("invalid public key format: %w", err)
	}

	// Build IPC configuration string to remove peer
	config := fmt.Sprintf("public_key=%s\n", hexPublicKey)
	config += "remove=true\n\n"

	// Apply configuration via IPC
	if err := ub.applyIPCConfig(config); err != nil {
		return fmt.Errorf("failed to remove peer via IPC: %w", err)
	}

	// Remove from tracking
	delete(ub.peers, publicKey)

	slog.Info("Peer removed successfully", "peerCount", len(ub.peers))
	return nil
}

// GetPeers returns information about all connected peers
func (ub *UserspaceBackend) GetPeers() ([]PeerInfo, error) {
	ub.mu.RLock()
	defer ub.mu.RUnlock()

	if !ub.running {
		return nil, fmt.Errorf("backend not running")
	}

	// For userspace implementation, we'll return basic info from our tracking
	// More detailed stats would require parsing IPC get responses
	peers := make([]PeerInfo, 0, len(ub.peers))

	for publicKey, allowedIPs := range ub.peers {
		peers = append(peers, PeerInfo{
			PublicKey:  publicKey,
			AllowedIPs: allowedIPs,
			Endpoint:   "", // Would need IPC query for endpoint
			LastSeen:   0,  // Would need IPC query for handshake time
			RxBytes:    0,  // Would need IPC query for transfer stats
			TxBytes:    0,  // Would need IPC query for transfer stats
		})
	}

	return peers, nil
}

// IsRunning returns whether the backend is currently running
func (ub *UserspaceBackend) IsRunning() bool {
	ub.mu.RLock()
	defer ub.mu.RUnlock()

	return ub.running
}

// configureDevice configures the WireGuard device with server settings
func (ub *UserspaceBackend) configureDevice(config ServerConfig) error {
	// Convert base64 private key to hex for WireGuard IPC
	hexPrivateKey, err := ub.base64ToHex(config.PrivateKey)
	if err != nil {
		return fmt.Errorf("invalid private key format: %w", err)
	}

	// Build IPC configuration for server setup
	// UAPI format: private_key=<hex_key>\nlisten_port=<port>\n\n
	// Note: Private key is passed directly to WireGuard IPC, not logged
	ipcConfig := fmt.Sprintf("private_key=%s\nlisten_port=%d\n\n", hexPrivateKey, config.ListenPort)

	return ub.applyIPCConfig(ipcConfig)
}

// applyIPCConfig applies configuration to the device via IPC
func (ub *UserspaceBackend) applyIPCConfig(config string) error {
	if ub.device == nil {
		return fmt.Errorf("device not initialized")
	}

	// SECURITY: Do not log IPC config as it contains private key material
	// Use the exposed IPC method from our WireGuardDevice wrapper
	return ub.device.IpcSet(config)
}

// base64ToHex converts a base64-encoded key to hex format for WireGuard IPC
func (ub *UserspaceBackend) base64ToHex(base64Key string) (string, error) {
	keyBytes, err := base64.StdEncoding.DecodeString(base64Key)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64 key: %w", err)
	}

	if len(keyBytes) != 32 {
		return "", fmt.Errorf("key must be 32 bytes, got %d", len(keyBytes))
	}

	return hex.EncodeToString(keyBytes), nil
}
