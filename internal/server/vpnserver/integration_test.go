package vpnserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/november1306/go-vpn/internal/config"
	"github.com/november1306/go-vpn/internal/wireguard/keys"
)

// Mock HTTP handlers to simulate the actual server endpoints
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

type StatusResponse struct {
	Status         string     `json:"status"`
	ConnectedPeers int        `json:"connectedPeers"`
	Peers          []PeerInfo `json:"peers"`
	ServerInfo     ServerInfo `json:"serverInfo"`
	Timestamp      string     `json:"timestamp"`
}

// TestVPNServerClientIntegration tests the complete client-server communication workflow
func TestVPNServerClientIntegration(t *testing.T) {
	// Skip on systems without TUN support
	server, _ := NewUserspaceVPNServer("test_data")

	// Generate server keys
	serverPrivKey, serverPubKey, err := keys.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate server keys: %v", err)
	}

	config := ServerConfig{
		InterfaceName: "wg-test-integration",
		PrivateKey:    serverPrivKey,
		ListenPort:    51825,
		ServerIP:      "10.98.0.1/24",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Start server
	if err := server.Start(ctx, config); err != nil {
		if isTUNError(err) {
			t.Skipf("Skipping integration test - requires TUN support: %v", err)
		}
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop(ctx)

	t.Run("ClientRegistrationWorkflow", func(t *testing.T) {
		// Generate client keys (simulating client)
		_, clientPubKey, err := keys.GenerateKeyPair()
		if err != nil {
			t.Fatalf("Failed to generate client keys: %v", err)
		}

		// Test 1: Register client
		err = server.AddClient(clientPubKey, "10.98.0.2")
		if err != nil {
			t.Fatalf("Failed to register client: %v", err)
		}

		// Test 2: Verify client is connected
		peers, err := server.GetConnectedClients()
		if err != nil {
			t.Fatalf("Failed to get connected clients: %v", err)
		}

		if len(peers) != 1 {
			t.Errorf("Expected 1 connected peer, got %d", len(peers))
		}

		if peers[0].PublicKey != clientPubKey {
			t.Errorf("Expected client public key %s, got %s", clientPubKey, peers[0].PublicKey)
		}

		if len(peers[0].AllowedIPs) != 1 || peers[0].AllowedIPs[0] != "10.98.0.2/32" {
			t.Errorf("Expected allowed IPs [10.98.0.2/32], got %v", peers[0].AllowedIPs)
		}

		// Test 3: Verify server info
		serverInfo, err := server.GetServerInfo()
		if err != nil {
			t.Fatalf("Failed to get server info: %v", err)
		}

		if serverInfo.PublicKey != serverPubKey {
			t.Errorf("Expected server public key %s, got %s", serverPubKey, serverInfo.PublicKey)
		}

		if serverInfo.ServerIP != config.ServerIP {
			t.Errorf("Expected server IP %s, got %s", config.ServerIP, serverInfo.ServerIP)
		}

		// Test 4: Remove client
		err = server.RemoveClient(clientPubKey)
		if err != nil {
			t.Fatalf("Failed to remove client: %v", err)
		}

		// Test 5: Verify client is removed
		peers, err = server.GetConnectedClients()
		if err != nil {
			t.Fatalf("Failed to get connected clients after removal: %v", err)
		}

		if len(peers) != 0 {
			t.Errorf("Expected 0 connected peers after removal, got %d", len(peers))
		}
	})

	t.Run("MultipleClientsWorkflow", func(t *testing.T) {
		// Generate multiple client keys
		_, client1PubKey, err := keys.GenerateKeyPair()
		if err != nil {
			t.Fatalf("Failed to generate client1 keys: %v", err)
		}

		_, client2PubKey, err := keys.GenerateKeyPair()
		if err != nil {
			t.Fatalf("Failed to generate client2 keys: %v", err)
		}

		_, client3PubKey, err := keys.GenerateKeyPair()
		if err != nil {
			t.Fatalf("Failed to generate client3 keys: %v", err)
		}

		// Register multiple clients
		err = server.AddClient(client1PubKey, "10.98.0.10")
		if err != nil {
			t.Fatalf("Failed to register client1: %v", err)
		}

		err = server.AddClient(client2PubKey, "10.98.0.11")
		if err != nil {
			t.Fatalf("Failed to register client2: %v", err)
		}

		err = server.AddClient(client3PubKey, "10.98.0.12")
		if err != nil {
			t.Fatalf("Failed to register client3: %v", err)
		}

		// Verify all clients are connected
		peers, err := server.GetConnectedClients()
		if err != nil {
			t.Fatalf("Failed to get connected clients: %v", err)
		}

		if len(peers) != 3 {
			t.Errorf("Expected 3 connected peers, got %d", len(peers))
		}

		// Verify each client has correct configuration
		clientKeys := map[string]string{
			client1PubKey: "10.98.0.10/32",
			client2PubKey: "10.98.0.11/32",
			client3PubKey: "10.98.0.12/32",
		}

		for _, peer := range peers {
			expectedIP, exists := clientKeys[peer.PublicKey]
			if !exists {
				t.Errorf("Unexpected peer public key: %s", peer.PublicKey)
				continue
			}

			if len(peer.AllowedIPs) != 1 || peer.AllowedIPs[0] != expectedIP {
				t.Errorf("Peer %s: expected allowed IPs [%s], got %v",
					peer.PublicKey[:8], expectedIP, peer.AllowedIPs)
			}
		}

		// Remove one client and verify
		err = server.RemoveClient(client2PubKey)
		if err != nil {
			t.Fatalf("Failed to remove client2: %v", err)
		}

		peers, err = server.GetConnectedClients()
		if err != nil {
			t.Fatalf("Failed to get connected clients after removal: %v", err)
		}

		if len(peers) != 2 {
			t.Errorf("Expected 2 connected peers after removal, got %d", len(peers))
		}

		// Verify client2 is gone
		for _, peer := range peers {
			if peer.PublicKey == client2PubKey {
				t.Error("Client2 should have been removed but still exists")
			}
		}
	})
}

// TestHTTPAPIIntegration tests the HTTP API endpoints
func TestHTTPAPIIntegration(t *testing.T) {
	// Generate server keys
	serverPrivKey, serverPubKey, err := keys.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate server keys: %v", err)
	}

	// Mock configuration (similar to actual server config)
	cfg := &config.Config{
		Network: config.NetworkConfig{
			ClientIPDemo: "10.0.0.100",
		},
	}

	// Create server instance
	server, _ := NewUserspaceVPNServer("test_data")

	config := ServerConfig{
		InterfaceName: "wg-test-http",
		PrivateKey:    serverPrivKey,
		ListenPort:    51826,
		ServerIP:      "10.97.0.1/24",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Start server (skip if no TUN support)
	if err := server.Start(ctx, config); err != nil {
		if isTUNError(err) {
			t.Skipf("Skipping HTTP integration test - requires TUN support: %v", err)
		}
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop(ctx)

	// Create HTTP handler (simplified version of actual server)
	handleRegister := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req RegisterRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if req.ClientPublicKey == "" {
			http.Error(w, "clientPublicKey is required", http.StatusBadRequest)
			return
		}

		// Validate client public key
		if err := keys.ValidatePublicKey(req.ClientPublicKey); err != nil {
			http.Error(w, "Invalid client public key format", http.StatusBadRequest)
			return
		}

		// Add client to server
		clientIP := cfg.Network.ClientIPDemo
		if err := server.AddClient(req.ClientPublicKey, clientIP); err != nil {
			http.Error(w, fmt.Sprintf("Failed to add client: %v", err), http.StatusInternalServerError)
			return
		}

		// Get server info
		serverInfo, err := server.GetServerInfo()
		if err != nil {
			http.Error(w, "Failed to get server info", http.StatusInternalServerError)
			return
		}

		// Return success response
		response := RegisterResponse{
			ServerPublicKey: serverInfo.PublicKey,
			ServerEndpoint:  serverInfo.Endpoint,
			ClientIP:        clientIP + "/32",
			Message:         "Registration successful - VPN tunnel established",
			Timestamp:       time.Now().UTC().Format(time.RFC3339),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}

	handleStatus := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		peers, err := server.GetConnectedClients()
		if err != nil {
			http.Error(w, "Failed to get peers", http.StatusInternalServerError)
			return
		}

		serverInfo, err := server.GetServerInfo()
		if err != nil {
			http.Error(w, "Failed to get server info", http.StatusInternalServerError)
			return
		}

		status := "running"
		if !server.IsRunning() {
			status = "stopped"
		}

		response := StatusResponse{
			Status:         status,
			ConnectedPeers: len(peers),
			Peers:          peers,
			ServerInfo:     serverInfo,
			Timestamp:      time.Now().UTC().Format(time.RFC3339),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}

	t.Run("RegisterEndpoint", func(t *testing.T) {
		// Generate client keys
		_, clientPubKey, err := keys.GenerateKeyPair()
		if err != nil {
			t.Fatalf("Failed to generate client keys: %v", err)
		}

		// Create request
		reqBody := RegisterRequest{
			ClientPublicKey: clientPubKey,
		}
		jsonData, err := json.Marshal(reqBody)
		if err != nil {
			t.Fatalf("Failed to marshal request: %v", err)
		}

		// Make request
		req := httptest.NewRequest(http.MethodPost, "/api/register", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handleRegister(w, req)

		// Check response
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		}

		var resp RegisterResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Verify response fields
		if resp.ServerPublicKey != serverPubKey {
			t.Errorf("Expected server public key %s, got %s", serverPubKey, resp.ServerPublicKey)
		}

		if resp.ClientIP != cfg.Network.ClientIPDemo+"/32" {
			t.Errorf("Expected client IP %s/32, got %s", cfg.Network.ClientIPDemo, resp.ClientIP)
		}

		if !strings.Contains(resp.Message, "Registration successful") {
			t.Errorf("Expected success message, got %s", resp.Message)
		}
	})

	t.Run("StatusEndpoint", func(t *testing.T) {
		// Make request
		req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
		w := httptest.NewRecorder()

		handleStatus(w, req)

		// Check response
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var resp StatusResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Verify response fields
		if resp.Status != "running" {
			t.Errorf("Expected status 'running', got %s", resp.Status)
		}

		if resp.ConnectedPeers < 0 {
			t.Errorf("Expected non-negative peer count, got %d", resp.ConnectedPeers)
		}

		if resp.ServerInfo.PublicKey != serverPubKey {
			t.Errorf("Expected server public key %s, got %s", serverPubKey, resp.ServerInfo.PublicKey)
		}
	})

	t.Run("InvalidRequests", func(t *testing.T) {
		// Test invalid JSON
		req := httptest.NewRequest(http.MethodPost, "/api/register", strings.NewReader("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handleRegister(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for invalid JSON, got %d", w.Code)
		}

		// Test missing public key
		reqBody := RegisterRequest{ClientPublicKey: ""}
		jsonData, _ := json.Marshal(reqBody)
		req = httptest.NewRequest(http.MethodPost, "/api/register", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		handleRegister(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for missing public key, got %d", w.Code)
		}

		// Test invalid public key format
		reqBody = RegisterRequest{ClientPublicKey: "invalid-key-format"}
		jsonData, _ = json.Marshal(reqBody)
		req = httptest.NewRequest(http.MethodPost, "/api/register", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		handleRegister(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for invalid key format, got %d", w.Code)
		}
	})
}
