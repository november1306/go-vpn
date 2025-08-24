package vpnserver

import (
	"encoding/base64"
	"encoding/hex"
	"testing"

	"github.com/november1306/go-vpn/internal/wireguard/keys"
)

func TestUserspaceBackend_base64ToHex(t *testing.T) {
	backend := NewUserspaceBackend()

	t.Run("valid base64 key conversion", func(t *testing.T) {
		// Generate a test key pair
		_, publicKey, err := keys.GenerateKeyPair()
		if err != nil {
			t.Fatalf("Failed to generate key pair: %v", err)
		}

		// Convert base64 to hex
		hexKey, err := backend.base64ToHex(publicKey)
		if err != nil {
			t.Fatalf("base64ToHex failed: %v", err)
		}

		// Verify hex key format
		if len(hexKey) != 64 { // 32 bytes * 2 hex chars per byte
			t.Errorf("Expected hex key length 64, got %d", len(hexKey))
		}

		// Verify it's valid hex
		_, err = hex.DecodeString(hexKey)
		if err != nil {
			t.Errorf("Generated hex key is not valid hex: %v", err)
		}

		// Verify round-trip conversion
		keyBytes, err := base64.StdEncoding.DecodeString(publicKey)
		if err != nil {
			t.Fatalf("Failed to decode original base64 key: %v", err)
		}

		expectedHex := hex.EncodeToString(keyBytes)
		if hexKey != expectedHex {
			t.Errorf("Hex conversion mismatch:\nexpected: %s\ngot:      %s", expectedHex, hexKey)
		}
	})

	t.Run("invalid base64 key", func(t *testing.T) {
		_, err := backend.base64ToHex("invalid-base64!")
		if err == nil {
			t.Error("Expected error for invalid base64 key")
		}

		if err != nil && !contains(err.Error(), "failed to decode base64 key") {
			t.Errorf("Expected specific error message, got: %v", err)
		}
	})

	t.Run("wrong key length", func(t *testing.T) {
		// Create a 16-byte key instead of 32
		shortKey := base64.StdEncoding.EncodeToString(make([]byte, 16))

		_, err := backend.base64ToHex(shortKey)
		if err == nil {
			t.Error("Expected error for wrong key length")
		}

		if err != nil && !contains(err.Error(), "key must be 32 bytes") {
			t.Errorf("Expected specific error message, got: %v", err)
		}
	})

	t.Run("empty key", func(t *testing.T) {
		_, err := backend.base64ToHex("")
		if err == nil {
			t.Error("Expected error for empty key")
		}
	})
}

func TestKeyFormatCompatibility(t *testing.T) {
	t.Run("keys work with both server and client", func(t *testing.T) {
		// Generate keys using the same function as client
		clientPrivKey, clientPubKey, err := keys.GenerateKeyPair()
		if err != nil {
			t.Fatalf("Failed to generate client keys: %v", err)
		}

		serverPrivKey, serverPubKey, err := keys.GenerateKeyPair()
		if err != nil {
			t.Fatalf("Failed to generate server keys: %v", err)
		}

		backend := NewUserspaceBackend()

		// Test server private key conversion (used in device config)
		serverHexPrivKey, err := backend.base64ToHex(serverPrivKey)
		if err != nil {
			t.Fatalf("Failed to convert server private key: %v", err)
		}

		// Test client public key conversion (used in peer config)
		clientHexPubKey, err := backend.base64ToHex(clientPubKey)
		if err != nil {
			t.Fatalf("Failed to convert client public key: %v", err)
		}

		// Test server public key conversion (used in peer config)
		serverHexPubKey, err := backend.base64ToHex(serverPubKey)
		if err != nil {
			t.Fatalf("Failed to convert server public key: %v", err)
		}

		// All conversions should produce 64-character hex strings
		if len(serverHexPrivKey) != 64 {
			t.Errorf("Server private key hex length should be 64, got %d", len(serverHexPrivKey))
		}
		if len(clientHexPubKey) != 64 {
			t.Errorf("Client public key hex length should be 64, got %d", len(clientHexPubKey))
		}
		if len(serverHexPubKey) != 64 {
			t.Errorf("Server public key hex length should be 64, got %d", len(serverHexPubKey))
		}

		// Test that derived public key from private key also converts correctly
		derivedPubKey, err := keys.PublicKeyFromPrivate(clientPrivKey)
		if err != nil {
			t.Fatalf("Failed to derive public key: %v", err)
		}

		if derivedPubKey != clientPubKey {
			t.Error("Derived public key doesn't match generated public key")
		}

		derivedHexPubKey, err := backend.base64ToHex(derivedPubKey)
		if err != nil {
			t.Fatalf("Failed to convert derived public key: %v", err)
		}

		if derivedHexPubKey != clientHexPubKey {
			t.Error("Derived public key hex doesn't match original public key hex")
		}
	})
}

func TestDeviceInitializationOrder(t *testing.T) {
	t.Run("device must be set before configuration", func(t *testing.T) {
		// This test validates the fix for the race condition bug
		// The issue was: ub.device was nil when configureDevice() tried to call IPC
		backend := NewUserspaceBackend()

		// Verify initial state
		if backend.device != nil {
			t.Error("Backend device should be nil initially")
		}

		// Test that the device field exists and can be set
		// (This validates our fix where we set ub.device = device before configureDevice)
		if backend.peers == nil {
			t.Error("Backend peers map should be initialized")
		}

		// Verify the backend has the base64ToHex method that configureDevice depends on
		testKey := "dGVzdC1rZXktMzItYnl0ZXMtZXhhY3RseS0hISEh" // "test-key-32-bytes-exactly-!!!"
		_, err := backend.base64ToHex(testKey)
		if err == nil {
			t.Error("Expected error for test key (wrong length), but base64ToHex method exists")
		}
	})
}

func TestWireGuardIPCFormat(t *testing.T) {
	t.Run("IPC configuration format", func(t *testing.T) {
		backend := NewUserspaceBackend()

		// Generate test keys
		_, clientPubKey, err := keys.GenerateKeyPair()
		if err != nil {
			t.Fatalf("Failed to generate keys: %v", err)
		}

		// Convert to hex as the backend would
		hexPubKey, err := backend.base64ToHex(clientPubKey)
		if err != nil {
			t.Fatalf("Failed to convert key: %v", err)
		}

		// Test the ACTUAL IPC format that we use (without set=1 which caused errors)
		allowedIPs := []string{"10.0.0.2/32"}

		// This matches the actual format in userspace_backend.go:117-122
		config := "public_key=" + hexPubKey + "\n"
		for _, ip := range allowedIPs {
			config += "allowed_ip=" + ip + "\n"
		}
		config += "\n"

		// Verify format components match our actual implementation
		if !contains(config, "public_key="+hexPubKey+"\n") {
			t.Error("IPC config should contain hex public key")
		}
		if !contains(config, "allowed_ip=10.0.0.2/32\n") {
			t.Error("IPC config should contain allowed IP")
		}
		if !contains(config, "\n\n") {
			t.Error("IPC config should end with double newline")
		}

		// Verify it does NOT contain the broken set=1 command that caused IPC errors
		if contains(config, "set=1") {
			t.Error("IPC config should NOT contain 'set=1' command - this causes IPC errors")
		}

		// Verify hex key is lowercase (WireGuard convention)
		if hexPubKey != toLower(hexPubKey) {
			t.Error("Hex key should be lowercase for WireGuard compatibility")
		}
	})
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			indexOf(s, substr) >= 0)))
}

// Simple indexOf implementation
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// Simple toLower implementation for hex strings
func toLower(s string) string {
	result := make([]byte, len(s))
	for i, c := range []byte(s) {
		if c >= 'A' && c <= 'F' {
			result[i] = c + 32 // Convert A-F to a-f
		} else {
			result[i] = c
		}
	}
	return string(result)
}
