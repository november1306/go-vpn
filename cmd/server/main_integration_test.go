package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/november1306/go-vpn/internal/wireguard/keys"
)

// TestServerMainIntegration tests the complete server main function integration
func TestServerMainIntegration(t *testing.T) {
	t.Skip("Skipping main integration test - requires refactoring of global variables")
}

// TestClientServerKeyExchange tests the key exchange process specifically
func TestClientServerKeyExchange(t *testing.T) {
	t.Run("KeyFormatValidation", func(t *testing.T) {
		// Generate test keys
		clientPrivKey, clientPubKey, err := keys.GenerateKeyPair()
		if err != nil {
			t.Fatalf("Failed to generate client keys: %v", err)
		}

		serverPrivKey, serverPubKey, err := keys.GenerateKeyPair()
		if err != nil {
			t.Fatalf("Failed to generate server keys: %v", err)
		}

		// Validate key format compatibility
		if err := keys.ValidatePublicKey(clientPubKey); err != nil {
			t.Errorf("Client public key validation failed: %v", err)
		}

		if err := keys.ValidatePrivateKey(clientPrivKey); err != nil {
			t.Errorf("Client private key validation failed: %v", err)
		}

		if err := keys.ValidatePublicKey(serverPubKey); err != nil {
			t.Errorf("Server public key validation failed: %v", err)
		}

		if err := keys.ValidatePrivateKey(serverPrivKey); err != nil {
			t.Errorf("Server private key validation failed: %v", err)
		}

		// Verify key derivation
		derivedPubKey, err := keys.PublicKeyFromPrivate(clientPrivKey)
		if err != nil {
			t.Fatalf("Failed to derive public key: %v", err)
		}

		if derivedPubKey != clientPubKey {
			t.Error("Derived public key doesn't match generated public key")
		}
	})

	t.Run("RequestResponseFormat", func(t *testing.T) {
		// Generate client keys
		_, clientPubKey, err := keys.GenerateKeyPair()
		if err != nil {
			t.Fatalf("Failed to generate client keys: %v", err)
		}

		// Test request format
		reqBody := RegisterRequest{
			ClientPublicKey: clientPubKey,
		}

		jsonData, err := json.Marshal(reqBody)
		if err != nil {
			t.Fatalf("Failed to marshal request: %v", err)
		}

		// Verify JSON format
		var parsedReq RegisterRequest
		if err := json.Unmarshal(jsonData, &parsedReq); err != nil {
			t.Errorf("Failed to parse request JSON: %v", err)
		}

		if parsedReq.ClientPublicKey != clientPubKey {
			t.Errorf("Expected client public key %s, got %s", clientPubKey, parsedReq.ClientPublicKey)
		}

		// Test response format with real server key
		_, serverPubKey, err := keys.GenerateKeyPair()
		if err != nil {
			t.Fatalf("Failed to generate server keys: %v", err)
		}
		timestamp := time.Now().UTC().Format(time.RFC3339)
		
		respBody := RegisterResponse{
			ServerPublicKey: serverPubKey,
			ServerEndpoint:  ":51820",
			ClientIP:        "10.0.0.100/32",
			Message:         "Registration successful - VPN tunnel established",
			Timestamp:       timestamp,
		}

		jsonData, err = json.Marshal(respBody)
		if err != nil {
			t.Fatalf("Failed to marshal response: %v", err)
		}

		// Verify response JSON format
		var parsedResp RegisterResponse
		if err := json.Unmarshal(jsonData, &parsedResp); err != nil {
			t.Errorf("Failed to parse response JSON: %v", err)
		}

		if parsedResp.ServerPublicKey != serverPubKey {
			t.Errorf("Expected server public key %s, got %s", serverPubKey, parsedResp.ServerPublicKey)
		}

		if parsedResp.ClientIP != "10.0.0.100/32" {
			t.Errorf("Expected client IP 10.0.0.100/32, got %s", parsedResp.ClientIP)
		}

		if !strings.Contains(parsedResp.Message, "Registration successful") {
			t.Errorf("Expected success message, got %s", parsedResp.Message)
		}
	})
}

// Helper functions for testing

func testContext(t *testing.T) context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func isTestTUNError(err error) bool {
	errStr := err.Error()
	return strings.Contains(errStr, "wintun.dll") ||
		strings.Contains(errStr, "TUN interface") ||
		strings.Contains(errStr, "tun") ||
		strings.Contains(errStr, "Unable to load library") ||
		strings.Contains(errStr, "failed to create TUN interface")
}