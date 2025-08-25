package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/november1306/go-vpn/internal/client/config"
	"github.com/november1306/go-vpn/internal/client/tunnel"
	"github.com/november1306/go-vpn/internal/version"
	"github.com/november1306/go-vpn/internal/wireguard/keys"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "vpn-cli",
	Short: "GoWire VPN client",
	Long:  `GoWire VPN client for managing VPN connections and registrations.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("go-vpn cli %s\n", version.Version)
		fmt.Println("Use --help for available commands")
	},
}

var registerCmd = &cobra.Command{
	Use:   "register",
	Short: "Register with VPN server",
	Long:  `Register this client with a VPN server by exchanging public keys.`,
	Run: func(cmd *cobra.Command, args []string) {
		serverURL, _ := cmd.Flags().GetString("server")
		if err := runRegister(serverURL); err != nil {
			fmt.Fprintf(os.Stderr, "Registration failed: %v\n", err)
			os.Exit(1)
		}
	},
}

var connectCmd = &cobra.Command{
	Use:   "connect",
	Short: "Connect to VPN",
	Long:  `Connect to the VPN using stored configuration.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runConnect(); err != nil {
			fmt.Fprintf(os.Stderr, "Connection failed: %v\n", err)
			os.Exit(1)
		}
	},
}

var disconnectCmd = &cobra.Command{
	Use:   "disconnect",
	Short: "Disconnect from VPN",
	Long:  `Disconnect from the currently active VPN connection.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runDisconnect(); err != nil {
			fmt.Fprintf(os.Stderr, "Disconnect failed: %v\n", err)
			os.Exit(1)
		}
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show VPN status",
	Long:  `Show the current status of VPN connections and configuration.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runStatus(); err != nil {
			fmt.Fprintf(os.Stderr, "Status check failed: %v\n", err)
			os.Exit(1)
		}
	},
}

var testVPNCmd = &cobra.Command{
	Use:   "test-vpn",
	Short: "Test VPN tunnel functionality",
	Long:  `Test if the VPN tunnel is working by connecting to server test endpoint.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runTestVPN(); err != nil {
			fmt.Fprintf(os.Stderr, "VPN test failed: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	// Add version flag to root command
	rootCmd.Version = version.Version

	// Add subcommands
	rootCmd.AddCommand(registerCmd)
	rootCmd.AddCommand(connectCmd)
	rootCmd.AddCommand(disconnectCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(testVPNCmd)

	// Add flags for register command
	registerCmd.Flags().StringP("server", "s", "", "VPN server URL (required)")
	registerCmd.MarkFlagRequired("server")
}

type RegisterRequest struct {
	ClientPublicKey string `json:"clientPublicKey"`
}

type RegisterResponse struct {
	ServerPublicKey string `json:"serverPublicKey"`
	ServerEndpoint  string `json:"serverEndpoint"`
	ClientIP        string `json:"clientIP"`
	Message         string `json:"message"`
	Timestamp       string `json:"timestamp"`
}

func runRegister(serverURL string) error {
	fmt.Println("🔐 Client Registration Demo")

	// Check if already registered
	if config.Exists() {
		fmt.Println("⚠️ Already registered. Use 'vpn-cli connect' to establish VPN tunnel.")
		fmt.Println("   To re-register, first run: rm ~/.go-wire-vpn/config.json")
		return nil
	}

	// Generate client key pair
	fmt.Println("Generating client key pair...")
	clientPrivKey, clientPubKey, err := keys.GenerateKeyPair()
	if err != nil {
		return fmt.Errorf("failed to generate client keys: %w", err)
	}

	fmt.Printf("✅ Client Public Key: %s\n", clientPubKey)

	// Prepare request
	reqBody := RegisterRequest{
		ClientPublicKey: clientPubKey,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make HTTP request
	fmt.Printf("📡 Registering with server: %s\n", serverURL)
	resp, err := http.Post(serverURL+"/api/register", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	// Parse response
	var registerResp RegisterResponse
	if err := json.NewDecoder(resp.Body).Decode(&registerResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Save client configuration (WireGuard best practice: persistent config only)
	clientConfig := &config.ClientConfig{
		ClientPrivateKey: clientPrivKey,
		ClientPublicKey:  clientPubKey,
		ServerPublicKey:  registerResp.ServerPublicKey,
		ServerEndpoint:   registerResp.ServerEndpoint,
		ClientIP:         registerResp.ClientIP,
		RegisteredAt:     time.Now(),
	}

	if err := config.Save(clientConfig); err != nil {
		return fmt.Errorf("failed to save client configuration: %w", err)
	}

	// Display results
	fmt.Printf("✅ %s\n", registerResp.Message)
	fmt.Printf("📋 Server Details:\n")
	fmt.Printf("   Public Key: %s\n", registerResp.ServerPublicKey)
	fmt.Printf("   Endpoint: %s\n", registerResp.ServerEndpoint)
	fmt.Printf("   Your VPN IP: %s\n", registerResp.ClientIP)
	fmt.Printf("🕒 Timestamp: %s\n", registerResp.Timestamp)

	fmt.Println("\n🎉 Registration complete! Configuration saved securely.")
	fmt.Println("💡 Next step: Run 'vpn-cli connect' to establish VPN tunnel")

	return nil
}

func runConnect() error {
	// Load client configuration
	clientConfig, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w\nHint: Run 'vpn-cli register --server=<url>' first", err)
	}

	// Create tunnel manager
	tm := tunnel.NewTunnelManager(clientConfig)

	// Connect to VPN
	return tm.Connect()
}

func runDisconnect() error {
	// Load client configuration
	clientConfig, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create tunnel manager
	tm := tunnel.NewTunnelManager(clientConfig)

	// Disconnect from VPN
	return tm.Disconnect()
}

func runStatus() error {
	// Load client configuration
	clientConfig, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w\nHint: Run 'vpn-cli register --server=<url>' first", err)
	}

	// Create tunnel manager
	tm := tunnel.NewTunnelManager(clientConfig)

	// Get tunnel status
	status, err := tm.GetStatus()
	if err != nil {
		return fmt.Errorf("failed to get status: %w", err)
	}

	// Display status
	fmt.Println("📊 VPN Status")
	fmt.Println("==============")

	if status.IsConnected {
		fmt.Printf("Status: 🟢 Connected\n")
		fmt.Printf("Server: %s\n", status.ServerEndpoint)
		fmt.Printf("VPN IP: %s\n", status.ClientIP)
		if status.LastConnected != nil {
			fmt.Printf("Connected since: %s\n", status.LastConnected.Format("2006-01-02 15:04:05"))
		}
		if status.BytesReceived > 0 || status.BytesSent > 0 {
			fmt.Printf("Data transferred: ⬇️ %d bytes, ⬆️ %d bytes\n", status.BytesReceived, status.BytesSent)
		}
	} else {
		fmt.Printf("Status: 🔴 Disconnected\n")
		fmt.Printf("Server: %s (available)\n", status.ServerEndpoint)
		fmt.Printf("Your IP: %s (assigned)\n", status.ClientIP)
	}

	fmt.Printf("Registered: %s\n", status.RegisteredAt.Format("2006-01-02 15:04:05"))

	if status.IsConnected {
		fmt.Println("\n💡 Use 'vpn-cli disconnect' to close the VPN tunnel")
	} else {
		fmt.Println("\n💡 Use 'vpn-cli connect' to establish VPN tunnel")
	}

	return nil
}

func runTestVPN() error {
	// Load client configuration
	clientConfig, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w\nHint: Run 'vpn-cli register --server=<url>' first", err)
	}

	fmt.Println("🔍 Note: Attempting VPN test regardless of detected connection state")
	fmt.Println("   (Connection state detection has limitations with userspace WireGuard)")
	fmt.Println()

	fmt.Println("🧪 Testing VPN tunnel functionality...")
	
	// Extract server endpoint info
	serverEndpoint := clientConfig.ServerEndpoint
	if strings.HasPrefix(serverEndpoint, ":") {
		serverEndpoint = "localhost" + serverEndpoint
	}
	
	// Try to access the VPN test endpoint
	testURL := "http://localhost:8443/api/vpn-test"
	fmt.Printf("Testing VPN endpoint: %s\n", testURL)
	
	resp, err := http.Get(testURL)
	if err != nil {
		return fmt.Errorf("VPN test failed - could not reach test endpoint: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("VPN test failed - server returned status %d", resp.StatusCode)
	}
	
	// Parse response
	var testResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&testResp); err != nil {
		return fmt.Errorf("failed to parse test response: %w", err)
	}
	
	// Display results
	fmt.Println("\n✅ VPN Test Results:")
	fmt.Printf("   Message: %v\n", testResp["message"])
	fmt.Printf("   Client IP seen by server: %v\n", testResp["clientIP"])  
	fmt.Printf("   Server time: %v\n", testResp["serverTime"])
	fmt.Printf("   Via: %v\n", testResp["via"])
	fmt.Println()
	
	// Additional diagnostics
	fmt.Println("📊 VPN Tunnel Diagnostics:")
	fmt.Printf("   Local VPN IP: %s\n", clientConfig.ClientIP)
	fmt.Printf("   Server endpoint: %s\n", clientConfig.ServerEndpoint)
	fmt.Printf("   Connection method: Userspace WireGuard\n")
	
	return nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
