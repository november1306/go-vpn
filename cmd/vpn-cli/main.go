package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/spf13/cobra"
	"github.com/november1306/go-vpn/internal/version"
	"github.com/november1306/go-vpn/internal/wireguard/keys"
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
		fmt.Println("Connect command - implementation coming soon")
	},
}

var disconnectCmd = &cobra.Command{
	Use:   "disconnect",
	Short: "Disconnect from VPN",
	Long:  `Disconnect from the currently active VPN connection.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Disconnect command - implementation coming soon")
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show VPN status",
	Long:  `Show the current status of VPN connections and configuration.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Status command - implementation coming soon")
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

	// Add flags for register command
	registerCmd.Flags().StringP("server", "s", "", "VPN server URL (required)")
	registerCmd.MarkFlagRequired("server")
}

type RegisterRequest struct {
	ClientPublicKey string `json:"clientPublicKey"`
}

type RegisterResponse struct {
	ServerPublicKey string `json:"serverPublicKey"`
	Message         string `json:"message"`
	Timestamp       string `json:"timestamp"`
}

func runRegister(serverURL string) error {
	fmt.Println("🔐 Client Registration Demo")

	// Generate client key pair
	fmt.Println("Generating client key pair...")
	_, clientPubKey, err := keys.GenerateKeyPair()
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

	// Display results
	fmt.Printf("✅ %s\n", registerResp.Message)
	fmt.Printf("📋 Server Public Key: %s\n", registerResp.ServerPublicKey)
	fmt.Printf("🕒 Timestamp: %s\n", registerResp.Timestamp)

	fmt.Println("\n🎉 Registration complete! Keys exchanged successfully.")

	return nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
