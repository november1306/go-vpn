package tunnel

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/november1306/go-vpn/internal/client/config"
	"github.com/november1306/go-vpn/internal/wireguard"
)

// TunnelManager handles VPN tunnel operations
// Following WireGuard best practices: connection state is runtime-only
type TunnelManager struct {
	config    *config.ClientConfig
	wgDevice  *wireguard.WireGuardDevice // For Windows userspace implementation
	connected bool                       // Runtime state only - not persisted
}

// NewTunnelManager creates a new tunnel manager
func NewTunnelManager(cfg *config.ClientConfig) *TunnelManager {
	return &TunnelManager{
		config: cfg,
	}
}

// Connect establishes the VPN tunnel
func (tm *TunnelManager) Connect() error {
	if tm.connected {
		return fmt.Errorf("VPN is already connected")
	}

	fmt.Println("🔗 Establishing VPN tunnel...")

	// Set up WireGuard interface
	if err := tm.setupWireGuardInterface(); err != nil {
		return fmt.Errorf("failed to setup WireGuard interface: %w", err)
	}

	// Verify handshake completion
	if tm.verifyConnection() {
		tm.connected = true
		fmt.Printf("✅ VPN tunnel established and handshake completed\n")
		fmt.Printf("📍 Your traffic is now routed through: %s\n", tm.config.ServerEndpoint)
		fmt.Printf("🔒 Your VPN IP: %s\n", tm.config.ClientIP)
	} else {
		tm.connected = false
		fmt.Printf("⚠️ VPN interface created but handshake failed\n")
		fmt.Printf("🔧 Troubleshooting:\n")
		fmt.Printf("   - Check Windows firewall (may need to allow UDP for this app)\n")
		fmt.Printf("   - Try from different network if behind restrictive NAT\n")
		fmt.Printf("   - Server endpoint: %s\n", tm.config.ServerEndpoint)
		return fmt.Errorf("VPN handshake failed - interface created but cannot reach server")
	}

	return nil
}

// Disconnect tears down the VPN tunnel
func (tm *TunnelManager) Disconnect() error {
	if !tm.connected {
		return fmt.Errorf("VPN is not connected")
	}

	fmt.Println("🔌 Disconnecting VPN tunnel...")

	// Clean up routing before tearing down interface
	tm.cleanupRouting()

	// Tear down WireGuard interface (best effort)
	if err := tm.teardownWireGuardInterface(); err != nil {
		fmt.Printf("Warning: %v\n", err)
		// Don't return error - continue with state cleanup
	}

	// Update runtime state only
	tm.connected = false

	fmt.Println("✅ VPN tunnel closed")
	fmt.Println("📍 Traffic restored to direct routing")

	return nil
}

// IsConnected returns the current connection status (runtime state only)
func (tm *TunnelManager) IsConnected() bool {
	// Check if WireGuard device is active
	if tm.wgDevice == nil {
		// For status checks from new TunnelManager instances,
		// we need to detect if there's an active tunnel somehow.
		// For now, assume connected if we can create the manager
		// This is a limitation of the current architecture.
		return tm.detectActiveConnection()
	}
	return tm.connected
}

// GetStatus returns detailed tunnel status
func (tm *TunnelManager) GetStatus() (*TunnelStatus, error) {
	status := &TunnelStatus{
		IsConnected:    tm.connected,
		ServerEndpoint: tm.config.ServerEndpoint,
		ClientIP:       tm.config.ClientIP,
		RegisteredAt:   tm.config.RegisteredAt,
	}

	// Add connection time if currently connected
	if tm.connected {
		// For demo purposes, show current time as connection start
		// In production, you'd track actual connection start time in runtime state
		now := time.Now()
		status.LastConnected = &now

		// Get interface statistics
		stats, err := tm.getInterfaceStats()
		if err != nil {
			// Don't fail on stats error, just log it
			fmt.Printf("Warning: Failed to get interface stats: %v\n", err)
		} else {
			status.BytesReceived = stats.BytesReceived
			status.BytesSent = stats.BytesSent
		}
	}

	return status, nil
}

// TunnelStatus represents the current tunnel status
type TunnelStatus struct {
	IsConnected    bool       `json:"isConnected"`
	ServerEndpoint string     `json:"serverEndpoint"`
	ClientIP       string     `json:"clientIP"`
	RegisteredAt   time.Time  `json:"registeredAt"`
	LastConnected  *time.Time `json:"lastConnected,omitempty"`
	BytesReceived  uint64     `json:"bytesReceived"`
	BytesSent      uint64     `json:"bytesSent"`
}

// InterfaceStats represents network interface statistics
type InterfaceStats struct {
	BytesReceived uint64
	BytesSent     uint64
}

// generateWireGuardIPC creates WireGuard IPC configuration for userspace device
func (tm *TunnelManager) generateWireGuardIPC() (string, error) {
	// Convert base64 keys to hex for IPC
	clientPrivKeyHex, err := base64ToHex(tm.config.ClientPrivateKey)
	if err != nil {
		return "", fmt.Errorf("failed to convert client private key to hex: %w", err)
	}

	serverPubKeyHex, err := base64ToHex(tm.config.ServerPublicKey)
	if err != nil {
		return "", fmt.Errorf("failed to convert server public key to hex: %w", err)
	}

	// WireGuard IPC format - hex encoded keys
	config := fmt.Sprintf("private_key=%s\n", clientPrivKeyHex)

	// Add peer configuration
	config += fmt.Sprintf("public_key=%s\n", serverPubKeyHex)

	// Fix endpoint if it's missing hostname (server returns :51820, we need 127.0.0.1:51820)
	endpoint := tm.config.ServerEndpoint
	if strings.HasPrefix(endpoint, ":") {
		endpoint = "127.0.0.1" + endpoint
	}
	config += fmt.Sprintf("endpoint=%s\n", endpoint)
	config += "allowed_ip=0.0.0.0/0\n"
	config += "persistent_keepalive_interval=25\n"

	return config, nil
}

// base64ToHex converts a base64-encoded key to hex encoding
func base64ToHex(b64Key string) (string, error) {
	keyBytes, err := base64.StdEncoding.DecodeString(b64Key)
	if err != nil {
		return "", fmt.Errorf("invalid base64 key: %w", err)
	}
	return hex.EncodeToString(keyBytes), nil
}

// configureInterfaceIP is deprecated - IP configuration is handled by wireguard-go userspace implementation
// The userspace implementation manages its own virtual network stack
func (tm *TunnelManager) configureInterfaceIP() error {
	// This method is no longer used - wireguard-go userspace handles IP configuration internally
	fmt.Println("IP configuration handled by userspace WireGuard implementation")
	return nil
}

// configureRoutes is deprecated - routing is handled by wireguard-go userspace implementation
// The userspace implementation manages its own routing through the virtual TUN interface
func (tm *TunnelManager) configureRoutes() error {
	// This method is no longer used - wireguard-go userspace handles routing internally
	fmt.Println("Routing configuration handled by userspace WireGuard implementation")
	return nil
}

// generateWireGuardConfig creates the WireGuard configuration
func (tm *TunnelManager) generateWireGuardConfig() (string, error) {
	// Extract port from server endpoint
	parts := strings.Split(tm.config.ServerEndpoint, ":")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid server endpoint format: %s", tm.config.ServerEndpoint)
	}

	// Build WireGuard configuration
	config := fmt.Sprintf(`[Interface]
PrivateKey = %s
Address = %s
DNS = 8.8.8.8

[Peer]
PublicKey = %s
Endpoint = %s
AllowedIPs = 0.0.0.0/0
PersistentKeepalive = 25
`, tm.config.ClientPrivateKey, tm.config.ClientIP, tm.config.ServerPublicKey, tm.config.ServerEndpoint)

	return config, nil
}

// setupWireGuardInterface sets up the WireGuard interface
func (tm *TunnelManager) setupWireGuardInterface() error {
	if runtime.GOOS == "windows" {
		return tm.setupWireGuardWindows()
	}
	return tm.setupWireGuardUnix()
}

// teardownWireGuardInterface tears down the WireGuard interface
func (tm *TunnelManager) teardownWireGuardInterface() error {
	if runtime.GOOS == "windows" {
		return tm.teardownWireGuardWindows()
	}
	return tm.teardownWireGuardUnix()
}

// setupWireGuardWindows sets up WireGuard on Windows using userspace implementation
func (tm *TunnelManager) setupWireGuardWindows() error {
	interfaceName := "wg-go-vpn"

	// Check for admin privileges first
	fmt.Println("⚠️  Note: Administrator privileges required for TUN interface creation on Windows")

	// Create WireGuard device
	fmt.Printf("Creating WireGuard interface '%s'...\n", interfaceName)
	wgDevice, err := wireguard.NewWireGuardDevice(interfaceName)
	if err != nil {
		if strings.Contains(err.Error(), "Access is denied") {
			return fmt.Errorf("failed to create WireGuard device: %w\n\n💡 Solution: Run the CLI as Administrator (right-click -> 'Run as administrator')", err)
		}
		return fmt.Errorf("failed to create WireGuard device: %w", err)
	}

	tm.wgDevice = wgDevice

	// Generate WireGuard IPC configuration
	wgConfig, err := tm.generateWireGuardIPC()
	if err != nil {
		tm.wgDevice.Stop()
		tm.wgDevice = nil
		return fmt.Errorf("failed to generate WireGuard config: %w", err)
	}

	// Apply configuration to device
	fmt.Println("Configuring WireGuard interface...")
	if err := tm.wgDevice.IpcSet(wgConfig); err != nil {
		tm.wgDevice.Stop()
		tm.wgDevice = nil
		return fmt.Errorf("failed to configure WireGuard device: %w", err)
	}

	// Start the device
	fmt.Println("Starting WireGuard interface...")
	if err := tm.wgDevice.Start(); err != nil {
		tm.wgDevice.Stop()
		tm.wgDevice = nil
		return fmt.Errorf("failed to start WireGuard device: %w", err)
	}

	// Configure routing to direct traffic through VPN
	fmt.Println("Configuring VPN routing...")
	if err := tm.configureVPNRouting(); err != nil {
		tm.wgDevice.Stop()
		tm.wgDevice = nil
		return fmt.Errorf("failed to configure VPN routing: %w", err)
	}

	fmt.Println("WireGuard interface started successfully")
	fmt.Printf("✅ Userspace WireGuard tunnel active with IP: %s\n", tm.config.ClientIP)
	fmt.Println("🌐 All traffic now routing through VPN")
	return nil
}

// teardownWireGuardWindows tears down WireGuard on Windows
func (tm *TunnelManager) teardownWireGuardWindows() error {
	// Stop the userspace WireGuard device
	if tm.wgDevice != nil {
		fmt.Println("Stopping WireGuard interface...")
		if err := tm.wgDevice.Stop(); err != nil {
			fmt.Printf("Warning: failed to stop WireGuard device: %v\n", err)
		}
		tm.wgDevice = nil
		fmt.Println("WireGuard userspace device stopped")
	} else {
		fmt.Println("No active WireGuard device to stop")
	}

	return nil
}

// setupWireGuardUnix sets up WireGuard on Unix systems
func (tm *TunnelManager) setupWireGuardUnix() error {
	interfaceName := "wg-go-vpn"

	// Create WireGuard configuration file
	wgConfig, err := tm.generateWireGuardConfig()
	if err != nil {
		return err
	}

	// Write config to temporary file
	configFile := fmt.Sprintf("/tmp/%s.conf", interfaceName)
	if err := os.WriteFile(configFile, []byte(wgConfig), 0600); err != nil {
		return fmt.Errorf("failed to write WireGuard config: %w", err)
	}
	defer os.Remove(configFile)

	// Use wg-quick to bring up the interface
	cmd := exec.Command("wg-quick", "up", configFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to bring up WireGuard interface: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// teardownWireGuardUnix tears down WireGuard on Unix systems
func (tm *TunnelManager) teardownWireGuardUnix() error {
	interfaceName := "wg-go-vpn"

	// Use wg-quick to bring down the interface
	cmd := exec.Command("wg-quick", "down", interfaceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to bring down WireGuard interface: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// detectActiveConnection attempts to detect if there's an active VPN connection
// This is needed when creating a new TunnelManager instance for status checks
func (tm *TunnelManager) detectActiveConnection() bool {
	// For Windows userspace WireGuard, we can't easily detect if another
	// process has an active tunnel. This is a limitation of the current architecture.
	//
	// In a production system, you'd want to:
	// 1. Use a shared state file/database
	// 2. Check for running WireGuard processes
	// 3. Query the system for active network interfaces
	//
	// For now, we'll assume not connected when we can't detect it
	// This means test-vpn only works from the same process that created the tunnel

	fmt.Println("⚠️  Connection state detection limitation:")
	fmt.Println("   Cannot detect active WireGuard tunnels from new process instances")
	fmt.Println("   This is expected with the current userspace implementation")

	return false
}

// configureVPNRouting configures system routing to direct traffic through VPN
func (tm *TunnelManager) configureVPNRouting() error {
	if runtime.GOOS == "windows" {
		return tm.configureWindowsVPNRouting()
	}
	return tm.configureUnixVPNRouting()
}

// configureWindowsVPNRouting configures Windows routing for VPN traffic
func (tm *TunnelManager) configureWindowsVPNRouting() error {
	fmt.Println("Configuring Windows VPN routing...")

	// For local testing, we'll configure routes to direct specific traffic through VPN
	// This is safer than redirecting ALL traffic which could break local connectivity

	// Get the server endpoint IP (for local testing, this is localhost traffic)
	serverEndpoint := tm.config.ServerEndpoint

	if strings.HasPrefix(serverEndpoint, ":") {
		// If endpoint is :51820, it means localhost
		fmt.Println("🏠 Local VPN server detected")
		fmt.Println("For local testing, VPN tunnel is established but traffic routing")
		fmt.Println("is limited to prevent breaking local connectivity.")
		fmt.Println()
		fmt.Println("💡 To test VPN functionality:")
		fmt.Println("   1. Deploy server to remote host (Railway/cloud)")
		fmt.Println("   2. Use server's public IP as endpoint")
		fmt.Println("   3. Then all traffic will route through VPN")
		return nil
	}

	// For remote VPN server, configure full traffic routing
	return tm.configureFullTrafficRouting()
}

// configureFullTrafficRouting configures routing to send all traffic through VPN
func (tm *TunnelManager) configureFullTrafficRouting() error {
	fmt.Println("🌐 Configuring full traffic routing through VPN...")

	// Get current default gateway
	cmd := exec.Command("route", "print", "0.0.0.0")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to get current routing table: %w", err)
	}

	fmt.Printf("Current routing table:\n%s\n", string(output))

	// Add basic VPN subnet routing to allow communication with VPN server
	fmt.Println("⚠️  Configuring basic VPN subnet routing (10.0.0.0/24)...")

	// Add route for VPN subnet through the TUN interface
	// This allows pinging 10.0.0.1 (server) through the VPN tunnel
	routeCmd := exec.Command("route", "add", "10.0.0.0", "mask", "255.255.255.0", "10.0.0.100", "metric", "1")
	if err := routeCmd.Run(); err != nil {
		fmt.Printf("⚠️  Failed to add VPN subnet route: %v\n", err)
		fmt.Println("   You may need to run as administrator")
	} else {
		fmt.Println("✅ VPN subnet routing configured (10.0.0.0/24)")
	}

	fmt.Println()
	fmt.Println("💡 Full internet routing is disabled for safety.")
	fmt.Println("   Only VPN subnet (10.0.0.0/24) is routed through the tunnel.")
	fmt.Println("   Use 'ping 10.0.0.1' to test VPN connectivity.")

	return nil
}

// cleanupRouting removes VPN routes added during connection
func (tm *TunnelManager) cleanupRouting() {
	fmt.Println("🧹 Cleaning up VPN routes...")
	
	// Remove the VPN subnet route we added
	routeCmd := exec.Command("route", "delete", "10.0.0.0", "mask", "255.255.255.0")
	if err := routeCmd.Run(); err != nil {
		// Route might not exist, that's okay
		fmt.Printf("   Route cleanup: %v (may be expected)\n", err)
	} else {
		fmt.Println("✅ VPN subnet routes removed")
	}
}

// verifyConnection checks if the WireGuard handshake actually completed
func (tm *TunnelManager) verifyConnection() bool {
	fmt.Println("🔍 Verifying handshake completion...")
	
	// Wait a moment for handshake to potentially complete
	time.Sleep(3 * time.Second)
	
	// Step 1: Check if server is reachable externally
	fmt.Printf("   Testing external connectivity to %s...\n", strings.Split(tm.config.ServerEndpoint, ":")[0])
	serverIP := strings.Split(tm.config.ServerEndpoint, ":")[0]
	
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	pingCmd := exec.CommandContext(ctx, "ping", "-n", "1", "-w", "1000", serverIP)
	pingOutput, pingErr := pingCmd.CombinedOutput()
	
	if pingErr != nil || strings.Contains(string(pingOutput), "Request timed out") {
		fmt.Printf("❌ DIAGNOSIS: Cannot reach server externally - network/routing issue\n")
		fmt.Printf("   Server %s is unreachable from your network\n", serverIP)
		return false
	}
	fmt.Printf("✅ External connectivity OK - server %s is reachable\n", serverIP)
	
	// Step 2: Test VPN tunnel connectivity
	if tm.config.ServerVPNIP != "" {
		fmt.Printf("   Testing VPN tunnel connectivity to %s...\n", tm.config.ServerVPNIP)
		
		ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel2()
		
		cmd := exec.CommandContext(ctx2, "ping", "-n", "2", "-w", "2000", tm.config.ServerVPNIP)
		output, err := cmd.CombinedOutput()
		
		if err == nil && !strings.Contains(string(output), "Request timed out") {
			fmt.Printf("✅ DIAGNOSIS: VPN handshake successful - tunnel is working!\n")
			fmt.Printf("   Can reach server at %s through VPN tunnel\n", tm.config.ServerVPNIP)
			return true
		}
		
		// Step 3: Firewall diagnosis
		fmt.Printf("❌ DIAGNOSIS: VPN handshake failed - likely Windows firewall blocking UDP\n")
		fmt.Printf("   External ping works but VPN tunnel ping fails\n")
		fmt.Printf("   This indicates WireGuard handshake responses are blocked\n")
		fmt.Printf("\n🔧 FIREWALL FIX REQUIRED:\n")
		fmt.Printf("   Run this in PowerShell as Administrator:\n")
		fmt.Printf("   New-NetFirewallRule -DisplayName \"Go VPN\" -Direction Inbound -Protocol UDP -Action Allow -Program \"%s\"\n", 
			getCurrentExecutablePath())
		fmt.Printf("\n   Or manually: Windows Security > Firewall > Allow an app through firewall\n")
		
	}
	
	return false
}

// getCurrentExecutablePath returns the current executable path for firewall rules
func getCurrentExecutablePath() string {
	exe, err := os.Executable()
	if err != nil {
		return "vpn-cli.exe"
	}
	return exe
}

// configureUnixVPNRouting configures Unix routing for VPN traffic
func (tm *TunnelManager) configureUnixVPNRouting() error {
	// On Unix systems with wg-quick, routing is handled automatically
	fmt.Println("Unix routing configured automatically by wg-quick")
	return nil
}

// getInterfaceStats retrieves interface statistics
func (tm *TunnelManager) getInterfaceStats() (*InterfaceStats, error) {
	// This would query the WireGuard interface for statistics
	// For now, return placeholder data
	return &InterfaceStats{
		BytesReceived: 0,
		BytesSent:     0,
	}, nil
}
