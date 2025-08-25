package vpnserver

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/november1306/go-vpn/internal/wireguard/keys"
)

const (
	// MaxTCPUDPPort is the maximum valid TCP/UDP port number
	MaxTCPUDPPort = 65535
)

// VPNServer manages the WireGuard VPN server with pluggable backends
// This allows scaling from userspace (MVP) to kernel implementations (high-scale)
type VPNServer struct {
	mu        sync.RWMutex
	backend   WireGuardBackend
	config    ServerConfig
	running   bool
	peerStore *PeerStore // Persistent peer storage for restart resilience
}

// NewVPNServer creates a new VPN server with the specified backend
// For MVP, use NewUserspaceBackend(). For scale, implement KernelBackend later.
func NewVPNServer(backend WireGuardBackend, dataDir string) (*VPNServer, error) {
	peerStore, err := NewPeerStore(dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create peer store: %w", err)
	}

	return &VPNServer{
		backend:   backend,
		peerStore: peerStore,
	}, nil
}

// NewUserspaceVPNServer creates a VPN server with userspace backend (convenience constructor)
func NewUserspaceVPNServer(dataDir string) (*VPNServer, error) {
	return NewVPNServer(NewUserspaceBackend(), dataDir)
}

// Start initializes and starts the VPN server
func (s *VPNServer) Start(ctx context.Context, config ServerConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("VPN server already running")
	}

	slog.Info("Starting VPN server", "interface", config.InterfaceName, "serverIP", config.ServerIP, "port", config.ListenPort)

	// Validate configuration
	if err := s.validateConfig(config); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Start the backend
	if err := s.backend.Start(ctx, config); err != nil {
		return fmt.Errorf("backend start failed: %w", err)
	}

	// Restore persisted peers (WireGuard best practice: survive restarts)
	if err := s.restorePersistedPeers(); err != nil {
		slog.Warn("Failed to restore persisted peers", "error", err)
		// Don't fail startup, just log warning
	}

	s.config = config
	s.running = true

	slog.Info("VPN server started successfully",
		"interface", config.InterfaceName,
		"serverIP", config.ServerIP,
		"port", config.ListenPort)
	return nil
}

// Stop gracefully shuts down the VPN server
func (s *VPNServer) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil // Already stopped
	}

	slog.Info("Stopping VPN server", "interface", s.config.InterfaceName)

	if err := s.backend.Stop(ctx); err != nil {
		slog.Error("Backend stop failed", "error", err)
		// Continue with cleanup even if backend stop fails
	}

	s.running = false

	slog.Info("VPN server stopped")
	return nil
}

// AddClient adds a new VPN client as a peer
// This is the core functionality that gets called when a client registers
func (s *VPNServer) AddClient(publicKey string, clientIP string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.running {
		return fmt.Errorf("VPN server not running")
	}

	slog.Info("Adding VPN client", "clientIP", clientIP)

	// Client gets their assigned IP as their allowed IP range
	// This means they can only send traffic from this specific IP
	allowedIPs := []string{clientIP + "/32"}

	if err := s.backend.AddPeer(publicKey, allowedIPs); err != nil {
		return fmt.Errorf("failed to add client peer: %w", err)
	}

	// Persist peer configuration (survive server restarts)
	if err := s.peerStore.AddPeer(publicKey, clientIP+"/32"); err != nil {
		slog.Warn("Failed to persist peer configuration", "error", err)
		// Don't fail the registration, just log warning
	}

	slog.Info("VPN client added successfully", "clientIP", clientIP)
	return nil
}

// RemoveClient removes a VPN client peer
func (s *VPNServer) RemoveClient(publicKey string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.running {
		return fmt.Errorf("VPN server not running")
	}

	slog.Info("Removing VPN client")

	if err := s.backend.RemovePeer(publicKey); err != nil {
		return fmt.Errorf("failed to remove client peer: %w", err)
	}

	// Remove from persistent storage
	if err := s.peerStore.RemovePeer(publicKey); err != nil {
		slog.Warn("Failed to remove peer from persistent storage", "error", err)
		// Don't fail the removal, just log warning
	}

	slog.Info("VPN client removed successfully")
	return nil
}

// GetConnectedClients returns information about all connected clients
func (s *VPNServer) GetConnectedClients() ([]PeerInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.running {
		return nil, fmt.Errorf("VPN server not running")
	}

	return s.backend.GetPeers()
}

// IsRunning returns whether the VPN server is currently running
func (s *VPNServer) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.running && s.backend.IsRunning()
}

// GetConfig returns the current server configuration (read-only copy)
func (s *VPNServer) GetConfig() ServerConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.config
}

// GetServerInfo returns basic server information for clients
type ServerInfo struct {
	PublicKey string
	Endpoint  string // IP:Port where clients should connect
	ServerIP  string // Server IP within VPN network
}

// GetServerInfo returns connection information that clients need
func (s *VPNServer) GetServerInfo() (ServerInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.running {
		return ServerInfo{}, fmt.Errorf("VPN server not running")
	}

	// Extract public key from private key for client connection info
	// This would typically be computed once at startup and cached
	publicKey, err := s.derivePublicKey(s.config.PrivateKey)
	if err != nil {
		return ServerInfo{}, fmt.Errorf("failed to derive public key: %w", err)
	}

	return ServerInfo{
		PublicKey: publicKey,
		Endpoint:  fmt.Sprintf(":%d", s.config.ListenPort), // Client needs to know port
		ServerIP:  s.config.ServerIP,
	}, nil
}

// validateConfig validates the server configuration
func (s *VPNServer) validateConfig(config ServerConfig) error {
	if config.InterfaceName == "" {
		return fmt.Errorf("interface name is required")
	}

	if config.PrivateKey == "" {
		return fmt.Errorf("private key is required")
	}

	// Validate private key format for security
	if err := keys.ValidatePrivateKey(config.PrivateKey); err != nil {
		return fmt.Errorf("invalid private key: %w", err)
	}

	if config.ListenPort <= 0 || config.ListenPort > MaxTCPUDPPort {
		return fmt.Errorf("invalid listen port: %d", config.ListenPort)
	}

	if config.ServerIP == "" {
		return fmt.Errorf("server IP is required")
	}

	return nil
}

// derivePublicKey derives the public key from the private key
func (s *VPNServer) derivePublicKey(privateKey string) (string, error) {
	return keys.PublicKeyFromPrivate(privateKey)
}

// restorePersistedPeers restores peer configurations after server restart
// This ensures WireGuard best practice: registered peers survive restarts
func (s *VPNServer) restorePersistedPeers() error {
	peers := s.peerStore.ListPeers()
	if len(peers) == 0 {
		slog.Info("No persisted peers to restore")
		return nil
	}

	slog.Info("Restoring persisted peers", "count", len(peers))
	restored := 0
	
	for publicKey, peerConfig := range peers {
		allowedIPs := []string{peerConfig.AllowedIPs}
		if err := s.backend.AddPeer(publicKey, allowedIPs); err != nil {
			slog.Warn("Failed to restore peer", "publicKey", publicKey, "error", err)
			continue
		}
		restored++
		slog.Debug("Restored peer", "publicKey", publicKey, "allowedIPs", peerConfig.AllowedIPs)
	}
	
	slog.Info("Peer restoration complete", "restored", restored, "total", len(peers))
	return nil
}
