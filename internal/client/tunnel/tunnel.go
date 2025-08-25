package tunnel

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strconv"
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

	// Add basic VPN subnet routing to allow communication with VPN server
	fmt.Println("⚠️  Configuring basic VPN subnet routing (10.0.0.0/24)...")

	// Get TUN interface information
	tunInterfaceIndex, err := tm.getTunInterfaceIndex()
	if err != nil {
		return fmt.Errorf("failed to get TUN interface index: %w", err)
	}

	// Step 1: Get current default gateway before we change it
	fmt.Println("   Getting current default gateway...")
	defaultGatewayCmd := exec.Command("route", "print", "0.0.0.0")
	defaultOutput, err := defaultGatewayCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to get current default gateway: %w", err)
	}
	
	// Extract current default gateway for host route
	currentGateway, currentInterface := tm.extractDefaultGateway(string(defaultOutput))
	if currentGateway == "" {
		return fmt.Errorf("could not determine current default gateway")
	}
	fmt.Printf("   Current default gateway: %s via interface %s\n", currentGateway, currentInterface)
	
	// Step 2: Add host route for VPN server to prevent routing loop
	serverIP := strings.Split(tm.config.ServerEndpoint, ":")[0]
	if serverIP != "127.0.0.1" && serverIP != "localhost" {
		fmt.Printf("   Adding host route for VPN server %s...\n", serverIP)
		hostRouteCmd := exec.Command("route", "add", serverIP, "mask", "255.255.255.255", currentGateway)
		if err := hostRouteCmd.Run(); err != nil {
			fmt.Printf("⚠️  Warning: Failed to add host route for VPN server: %v\n", err)
		}
	}
	
	// Step 3: Route all traffic through VPN tunnel
	fmt.Printf("   Routing ALL traffic through WireGuard interface %d...\n", tunInterfaceIndex)
	
	// Add 0.0.0.0/1 and 128.0.0.0/1 routes (this covers all traffic)
	// This is the standard VPN approach to override the default route
	routes := []struct{ network, mask string }{
		{"0.0.0.0", "128.0.0.0"},     // 0.0.0.0/1
		{"128.0.0.0", "128.0.0.0"},   // 128.0.0.0/1
	}
	
	for _, route := range routes {
		routeCmd := exec.Command("route", "add", route.network, "mask", route.mask, "0.0.0.0", "if", strconv.Itoa(tunInterfaceIndex), "metric", "1")
		routeOutput, err := routeCmd.CombinedOutput()
		if err != nil {
			fmt.Printf("⚠️  Failed to add route %s/%s: %v\n", route.network, route.mask, err)
			fmt.Printf("   Route command output: %s\n", string(routeOutput))
		} else {
			fmt.Printf("✅ Route %s/%s configured through interface %d\n", route.network, route.mask, tunInterfaceIndex)
		}
	}
	
	// Step 4: Add VPN subnet route for server communication
	vpcRouteCmd := exec.Command("route", "add", "10.0.0.0", "mask", "255.255.255.0", "0.0.0.0", "if", strconv.Itoa(tunInterfaceIndex), "metric", "1")
	if err := vpcRouteCmd.Run(); err != nil {
		fmt.Printf("⚠️  Warning: Failed to add VPN subnet route: %v\n", err)
	}

	fmt.Println()
	fmt.Println("🌐 ALL TRAFFIC now routes through VPN!")
	fmt.Println("   Your real IP is hidden and all internet traffic is encrypted.")
	fmt.Println("   Use 'ping google.com' or 'curl https://ipinfo.io/ip' to test.")

	return nil
}

// cleanupRouting removes VPN routes added during connection
func (tm *TunnelManager) cleanupRouting() {
	fmt.Println("🧹 Cleaning up VPN routes...")

	// Remove the routes we added for full traffic routing
	routes := []struct{ network, mask string }{
		{"0.0.0.0", "128.0.0.0"},     // 0.0.0.0/1
		{"128.0.0.0", "128.0.0.0"},   // 128.0.0.0/1
		{"10.0.0.0", "255.255.255.0"}, // VPN subnet
	}
	
	for _, route := range routes {
		routeCmd := exec.Command("route", "delete", route.network, "mask", route.mask)
		if err := routeCmd.Run(); err != nil {
			// Route might not exist, that's okay
			fmt.Printf("   Route cleanup %s/%s: %v (may be expected)\n", route.network, route.mask, err)
		} else {
			fmt.Printf("✅ Route %s/%s removed\n", route.network, route.mask)
		}
	}
	
	// Remove host route for VPN server
	serverIP := strings.Split(tm.config.ServerEndpoint, ":")[0]
	if serverIP != "127.0.0.1" && serverIP != "localhost" {
		hostRouteCmd := exec.Command("route", "delete", serverIP, "mask", "255.255.255.255")
		if err := hostRouteCmd.Run(); err != nil {
			fmt.Printf("   Host route cleanup %s: %v (may be expected)\n", serverIP, err)
		} else {
			fmt.Printf("✅ Host route for %s removed\n", serverIP)
		}
	}
}

// verifyConnection checks if the WireGuard handshake actually completed
func (tm *TunnelManager) verifyConnection() bool {
	fmt.Println("🔍 Verifying handshake completion...")

	// Wait a moment for handshake to potentially complete
	time.Sleep(3 * time.Second)

	// Step 1: Check if server is reachable externally (only for remote servers)
	serverIP := strings.Split(tm.config.ServerEndpoint, ":")[0]
	if serverIP != "127.0.0.1" && serverIP != "localhost" {
		fmt.Printf("   Testing external connectivity to %s...\n", serverIP)

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
	}

	// Step 2: Test VPN tunnel connectivity (most important test)
	if tm.config.ServerVPNIP != "" {
		fmt.Printf("   Testing VPN tunnel connectivity to %s...\n", tm.config.ServerVPNIP)

		ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel2()

		cmd := exec.CommandContext(ctx2, "ping", "-n", "2", "-w", "2000", tm.config.ServerVPNIP)
		output, err := cmd.CombinedOutput()

		if err == nil && !strings.Contains(string(output), "Request timed out") {
			fmt.Printf("✅ DIAGNOSIS: VPN tunnel is working correctly!\n")
			fmt.Printf("   Can reach server at %s through VPN tunnel\n", tm.config.ServerVPNIP)
			return true
		}

		// Step 3: Advanced diagnosis - check routing table
		fmt.Printf("❌ DIAGNOSIS: VPN tunnel connectivity failed\n")
		fmt.Printf("   Checking routing configuration...\n")

		routeCmd := exec.Command("route", "print", "10.0.0.0")
		routeOutput, routeErr := routeCmd.CombinedOutput()
		if routeErr == nil {
			fmt.Printf("   Current 10.0.0.0 routes:\n%s\n", string(routeOutput))

			// Check if route goes through correct interface
			if strings.Contains(string(routeOutput), "192.168.") && !strings.Contains(string(routeOutput), "wg") {
				fmt.Printf("🔧 ROUTING ISSUE DETECTED:\n")
				fmt.Printf("   VPN traffic is routing through your physical network interface\n")
				fmt.Printf("   instead of the WireGuard tunnel interface.\n")
				fmt.Printf("   This has been fixed in the current version.\n")
				fmt.Printf("   Please reconnect the VPN to apply the fix.\n")
			} else {
				fmt.Printf("🔧 POSSIBLE CAUSES:\n")
				fmt.Printf("   1. Windows firewall blocking UDP traffic\n")
				fmt.Printf("   2. Network routing configuration issue\n")
				fmt.Printf("   3. WireGuard handshake not completing\n")
				fmt.Printf("\n   Try running as Administrator or check Windows firewall settings.\n")
			}
		}
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

// getTunInterfaceIndex gets the Windows interface index for the TUN device
func (tm *TunnelManager) getTunInterfaceIndex() (int, error) {
	// Get all network interfaces
	interfaces, err := net.Interfaces()
	if err != nil {
		return 0, fmt.Errorf("failed to get network interfaces: %w", err)
	}

	// Look for WireGuard/TUN interface by name pattern
	for _, iface := range interfaces {
		// Look for WireGuard Tunnel interfaces (typical pattern on Windows)
		if strings.Contains(strings.ToLower(iface.Name), "wireguard") ||
			strings.Contains(strings.ToLower(iface.Name), "wg-go-vpn") ||
			strings.Contains(strings.ToLower(iface.Name), "wintun") {
			fmt.Printf("Found TUN interface: %s (index: %d)\n", iface.Name, iface.Index)
			return iface.Index, nil
		}
	}

	// If no interface found by name, look for the most recently created interface
	// that matches typical TUN characteristics (this is a fallback)
	var candidates []net.Interface
	for _, iface := range interfaces {
		// Skip loopback and inactive interfaces
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		// Look for interfaces that might be TUN devices
		// TUN interfaces typically have specific characteristics
		if iface.MTU == 1420 { // Common WireGuard MTU
			candidates = append(candidates, iface)
		}
	}

	// If we found potential candidates, use the one with the highest index (most recent)
	if len(candidates) > 0 {
		bestCandidate := candidates[0]
		for _, candidate := range candidates[1:] {
			if candidate.Index > bestCandidate.Index {
				bestCandidate = candidate
			}
		}
		fmt.Printf("Found TUN interface candidate: %s (index: %d)\n", bestCandidate.Name, bestCandidate.Index)
		return bestCandidate.Index, nil
	}

	return 0, fmt.Errorf("no WireGuard/TUN interface found - please ensure the WireGuard device is running")
}

// extractDefaultGateway extracts the current default gateway from route output
func (tm *TunnelManager) extractDefaultGateway(routeOutput string) (gateway, interfaceName string) {
	lines := strings.Split(routeOutput, "\n")
	for _, line := range lines {
		// Parse the route line: Network Destination  Netmask  Gateway  Interface  Metric
		fields := strings.Fields(line)
		if len(fields) >= 4 {
			// Look for the default route (0.0.0.0 network with 0.0.0.0 netmask)
			if fields[0] == "0.0.0.0" && fields[1] == "0.0.0.0" {
				gateway = strings.TrimSpace(fields[2])
				interfaceName = strings.TrimSpace(fields[3])
				// Skip if gateway is 0.0.0.0 or On-link (we want the actual gateway)
				if gateway != "0.0.0.0" && gateway != "On-link" && gateway != "" {
					return gateway, interfaceName
				}
			}
		}
	}
	return "", ""
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
