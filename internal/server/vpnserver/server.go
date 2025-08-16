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
	mu      sync.RWMutex
	backend WireGuardBackend
	config  ServerConfig
	running bool
}

// NewVPNServer creates a new VPN server with the specified backend
// For MVP, use NewUserspaceBackend(). For scale, implement KernelBackend later.
func NewVPNServer(backend WireGuardBackend) *VPNServer {
	return &VPNServer{
		backend: backend,
	}
}

// NewUserspaceVPNServer creates a VPN server with userspace backend (convenience constructor)
func NewUserspaceVPNServer() *VPNServer {
	return NewVPNServer(NewUserspaceBackend())
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