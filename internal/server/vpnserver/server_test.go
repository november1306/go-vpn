package vpnserver

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/november1306/go-vpn/internal/wireguard/keys"
)

func TestVPNServerLifecycle(t *testing.T) {
	// Test basic server lifecycle: start, configure, stop
	server := NewUserspaceVPNServer()

	// Generate test server key
	serverPrivKey, _, err := keys.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate server key: %v", err)
	}

	config := ServerConfig{
		InterfaceName: "wg-test",
		PrivateKey:    serverPrivKey,
		ListenPort:    51822, // Use different port to avoid conflicts
		ServerIP:      "10.99.0.1/24",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test server start
	if server.IsRunning() {
		t.Error("Server should not be running initially")
	}

	err = server.Start(ctx, config)
	if err != nil {
		// On Windows/systems without TUN interface support, skip all tests gracefully
		if isTUNError(err) {
			t.Skipf("Skipping all VPN server tests - requires system TUN support: %v", err)
		}
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop(ctx)

	if !server.IsRunning() {
		t.Error("Server should be running after start")
	}

	// Test getting server info
	t.Run("GetServerInfo", func(t *testing.T) {
		info, err := server.GetServerInfo()
		if err != nil {
			t.Fatalf("Failed to get server info: %v", err)
		}

		if info.PublicKey == "" {
			t.Error("Server info should include public key")
		}

		if info.ServerIP != config.ServerIP {
			t.Errorf("Expected server IP %s, got %s", config.ServerIP, info.ServerIP)
		}
	})

	// Test initial peer count
	t.Run("InitialPeerCount", func(t *testing.T) {
		peers, err := server.GetConnectedClients()
		if err != nil {
			t.Fatalf("Failed to get connected clients: %v", err)
		}

		if len(peers) != 0 {
			t.Errorf("Expected 0 peers initially, got %d", len(peers))
		}
	})

	// Test server stop
	t.Run("Stop", func(t *testing.T) {
		err := server.Stop(ctx)
		if err != nil {
			t.Fatalf("Failed to stop server: %v", err)
		}

		if server.IsRunning() {
			t.Error("Server should not be running after stop")
		}
	})
}

func TestVPNServerPeerManagement(t *testing.T) {
	// Test adding and removing peers
	server := NewUserspaceVPNServer()

	// Generate server and client keys
	serverPrivKey, _, err := keys.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate server key: %v", err)
	}

	_, clientPubKey1, err := keys.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate client1 key: %v", err)
	}

	_, clientPubKey2, err := keys.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate client2 key: %v", err)
	}

	config := ServerConfig{
		InterfaceName: "wg-test",
		PrivateKey:    serverPrivKey,
		ListenPort:    51823,
		ServerIP:      "10.99.0.1/24",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Start server
	if err := server.Start(ctx, config); err != nil {
		if isTUNError(err) {
			t.Skipf("Skipping TUN interface test - requires system TUN support: %v", err)
		}
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop(ctx)

	// Test adding clients
	t.Run("AddClients", func(t *testing.T) {
		// Add first client
		err := server.AddClient(clientPubKey1, "10.99.0.2")
		if err != nil {
			t.Fatalf("Failed to add client1: %v", err)
		}

		// Add second client
		err = server.AddClient(clientPubKey2, "10.99.0.3")
		if err != nil {
			t.Fatalf("Failed to add client2: %v", err)
		}

		// Verify peer count
		peers, err := server.GetConnectedClients()
		if err != nil {
			t.Fatalf("Failed to get connected clients: %v", err)
		}

		if len(peers) != 2 {
			t.Errorf("Expected 2 peers, got %d", len(peers))
		}

		// Verify peer public keys
		foundClient1, foundClient2 := false, false
		for _, peer := range peers {
			if peer.PublicKey == clientPubKey1 {
				foundClient1 = true
				if len(peer.AllowedIPs) != 1 || peer.AllowedIPs[0] != "10.99.0.2/32" {
					t.Errorf("Client1 allowed IPs incorrect: %v", peer.AllowedIPs)
				}
			}
			if peer.PublicKey == clientPubKey2 {
				foundClient2 = true
				if len(peer.AllowedIPs) != 1 || peer.AllowedIPs[0] != "10.99.0.3/32" {
					t.Errorf("Client2 allowed IPs incorrect: %v", peer.AllowedIPs)
				}
			}
		}

		if !foundClient1 {
			t.Error("Client1 not found in peer list")
		}
		if !foundClient2 {
			t.Error("Client2 not found in peer list")
		}
	})

	// Test removing clients
	t.Run("RemoveClients", func(t *testing.T) {
		// Remove first client
		err := server.RemoveClient(clientPubKey1)
		if err != nil {
			t.Fatalf("Failed to remove client1: %v", err)
		}

		// Verify peer count
		peers, err := server.GetConnectedClients()
		if err != nil {
			t.Fatalf("Failed to get connected clients: %v", err)
		}

		if len(peers) != 1 {
			t.Errorf("Expected 1 peer after removal, got %d", len(peers))
		}

		// Verify remaining peer is client2
		if len(peers) > 0 && peers[0].PublicKey != clientPubKey2 {
			t.Errorf("Expected remaining peer to be client2, got %s", peers[0].PublicKey)
		}

		// Remove second client
		err = server.RemoveClient(clientPubKey2)
		if err != nil {
			t.Fatalf("Failed to remove client2: %v", err)
		}

		// Verify no peers remain
		peers, err = server.GetConnectedClients()
		if err != nil {
			t.Fatalf("Failed to get connected clients: %v", err)
		}

		if len(peers) != 0 {
			t.Errorf("Expected 0 peers after removing all, got %d", len(peers))
		}
	})
}

func TestVPNServerErrorCases(t *testing.T) {
	// Test error conditions
	server := NewUserspaceVPNServer()
	ctx := context.Background()

	t.Run("InvalidConfiguration", func(t *testing.T) {
		// Test with empty config
		err := server.Start(ctx, ServerConfig{})
		if err == nil {
			t.Error("Expected error with empty config")
		}

		// Test with invalid port
		err = server.Start(ctx, ServerConfig{
			InterfaceName: "wg0",
			PrivateKey:    "test-key",
			ListenPort:    -1,
			ServerIP:      "10.0.0.1/24",
		})
		if err == nil {
			t.Error("Expected error with invalid port")
		}
	})

	t.Run("OperationsOnStoppedServer", func(t *testing.T) {
		// Try operations on stopped server
		err := server.AddClient("test-key", "10.0.0.2")
		if err == nil {
			t.Error("Expected error adding client to stopped server")
		}

		err = server.RemoveClient("test-key")
		if err == nil {
			t.Error("Expected error removing client from stopped server")
		}

		_, err = server.GetConnectedClients()
		if err == nil {
			t.Error("Expected error getting clients from stopped server")
		}

		_, err = server.GetServerInfo()
		if err == nil {
			t.Error("Expected error getting server info from stopped server")
		}
	})

	t.Run("DoubleStart", func(t *testing.T) {
		// Generate valid config
		serverPrivKey, _, err := keys.GenerateKeyPair()
		if err != nil {
			t.Fatalf("Failed to generate server key: %v", err)
		}

		config := ServerConfig{
			InterfaceName: "wg-test",
			PrivateKey:    serverPrivKey,
			ListenPort:    51824,
			ServerIP:      "10.99.0.1/24",
		}

		// Start server
		err = server.Start(ctx, config)
		if err != nil {
			if isTUNError(err) {
				t.Skipf("Skipping TUN interface test - requires system TUN support: %v", err)
			}
			t.Fatalf("Failed to start server: %v", err)
		}
		defer server.Stop(ctx)

		// Try to start again
		err = server.Start(ctx, config)
		if err == nil {
			t.Error("Expected error starting already running server")
		}
	})
}

// isTUNError checks if the error is related to TUN interface creation
func isTUNError(err error) bool {
	errStr := err.Error()
	return strings.Contains(errStr, "wintun.dll") ||
		strings.Contains(errStr, "TUN interface") ||
		strings.Contains(errStr, "tun") ||
		strings.Contains(errStr, "Unable to load library") ||
		strings.Contains(errStr, "failed to create TUN interface")
}
