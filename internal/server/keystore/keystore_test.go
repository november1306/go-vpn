package keystore

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/november1306/go-vpn/internal/wireguard/keys"
)

func TestNewServerKeyStore_GeneratesNewKeys(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()

	// Create key store (should generate new keys)
	store, err := NewServerKeyStore(tempDir)
	if err != nil {
		t.Fatalf("Failed to create key store: %v", err)
	}

	// Verify keys are not empty
	if store.GetPrivateKey() == "" {
		t.Error("Private key should not be empty")
	}
	if store.GetPublicKey() == "" {
		t.Error("Public key should not be empty")
	}

	// Verify key file was created
	keyFilePath := filepath.Join(tempDir, "server-key.json")
	if _, err := os.Stat(keyFilePath); os.IsNotExist(err) {
		t.Error("Key file should have been created")
	}

	// Verify keys are valid
	if err := keys.ValidatePrivateKey(store.GetPrivateKey()); err != nil {
		t.Errorf("Generated private key is invalid: %v", err)
	}
	if err := keys.ValidatePublicKey(store.GetPublicKey()); err != nil {
		t.Errorf("Generated public key is invalid: %v", err)
	}
}

func TestNewServerKeyStore_LoadsExistingKeys(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()

	// Create first key store
	store1, err := NewServerKeyStore(tempDir)
	if err != nil {
		t.Fatalf("Failed to create first key store: %v", err)
	}

	originalPrivateKey := store1.GetPrivateKey()
	originalPublicKey := store1.GetPublicKey()

	// Create second key store (should load existing keys)
	store2, err := NewServerKeyStore(tempDir)
	if err != nil {
		t.Fatalf("Failed to create second key store: %v", err)
	}

	// Verify keys match
	if store2.GetPrivateKey() != originalPrivateKey {
		t.Error("Private key should match original")
	}
	if store2.GetPublicKey() != originalPublicKey {
		t.Error("Public key should match original")
	}
}

func TestNewServerKeyStore_InvalidJSON(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()

	// Write invalid JSON to key file
	keyFilePath := filepath.Join(tempDir, "server-key.json")
	if err := os.WriteFile(keyFilePath, []byte("invalid json"), 0600); err != nil {
		t.Fatalf("Failed to write invalid JSON: %v", err)
	}

	// Try to create key store (should fail)
	_, err := NewServerKeyStore(tempDir)
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

func TestNewServerKeyStore_InvalidPrivateKey(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()

	// Write invalid private key to file
	keyFilePath := filepath.Join(tempDir, "server-key.json")
	invalidKeys := ServerKeys{
		PrivateKey:  "invalid-key",
		PublicKey:   "some-public-key",
		GeneratedAt: time.Now(),
	}
	data, _ := json.MarshalIndent(invalidKeys, "", "  ")
	if err := os.WriteFile(keyFilePath, data, 0600); err != nil {
		t.Fatalf("Failed to write invalid keys: %v", err)
	}

	// Try to create key store (should fail)
	_, err := NewServerKeyStore(tempDir)
	if err == nil {
		t.Error("Expected error for invalid private key, got nil")
	}
}

func TestNewServerKeyStore_CreatesDataDirectory(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()
	dataDir := filepath.Join(tempDir, "data", "nested")

	// Create key store (should create nested directories)
	_, err := NewServerKeyStore(dataDir)
	if err != nil {
		t.Fatalf("Failed to create key store: %v", err)
	}

	// Verify directory was created
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		t.Error("Data directory should have been created")
	}
}

func TestServerKeyStore_GetKeys(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()

	// Create key store
	store, err := NewServerKeyStore(tempDir)
	if err != nil {
		t.Fatalf("Failed to create key store: %v", err)
	}

	// Get keys
	keys := store.GetKeys()

	// Verify keys match individual getters
	if keys.PrivateKey != store.GetPrivateKey() {
		t.Error("GetKeys().PrivateKey should match GetPrivateKey()")
	}
	if keys.PublicKey != store.GetPublicKey() {
		t.Error("GetKeys().PublicKey should match GetPublicKey()")
	}
}

func TestServerKeyStore_RegenerateKeys(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()

	// Create key store
	store, err := NewServerKeyStore(tempDir)
	if err != nil {
		t.Fatalf("Failed to create key store: %v", err)
	}

	originalPrivateKey := store.GetPrivateKey()
	originalPublicKey := store.GetPublicKey()

	// Regenerate keys
	if err := store.RegenerateKeys(); err != nil {
		t.Fatalf("Failed to regenerate keys: %v", err)
	}

	// Verify keys changed
	if store.GetPrivateKey() == originalPrivateKey {
		t.Error("Private key should have changed after regeneration")
	}
	if store.GetPublicKey() == originalPublicKey {
		t.Error("Public key should have changed after regeneration")
	}

	// Verify new keys are valid
	if err := keys.ValidatePrivateKey(store.GetPrivateKey()); err != nil {
		t.Errorf("Regenerated private key is invalid: %v", err)
	}
	if err := keys.ValidatePublicKey(store.GetPublicKey()); err != nil {
		t.Errorf("Regenerated public key is invalid: %v", err)
	}

	// Verify keys persisted to disk
	store2, err := NewServerKeyStore(tempDir)
	if err != nil {
		t.Fatalf("Failed to load key store after regeneration: %v", err)
	}

	if store2.GetPrivateKey() != store.GetPrivateKey() {
		t.Error("Regenerated keys should persist to disk")
	}
}

func TestServerKeyStore_AtomicWrite(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()

	// Create key store
	store, err := NewServerKeyStore(tempDir)
	if err != nil {
		t.Fatalf("Failed to create key store: %v", err)
	}

	keyFilePath := filepath.Join(tempDir, "server-key.json")

	// Read original file content
	originalData, err := os.ReadFile(keyFilePath)
	if err != nil {
		t.Fatalf("Failed to read key file: %v", err)
	}

	// Regenerate keys (triggers save)
	if err := store.RegenerateKeys(); err != nil {
		t.Fatalf("Failed to regenerate keys: %v", err)
	}

	// Read new file content
	newData, err := os.ReadFile(keyFilePath)
	if err != nil {
		t.Fatalf("Failed to read key file after regeneration: %v", err)
	}

	// Verify file content changed
	if string(originalData) == string(newData) {
		t.Error("File content should have changed")
	}

	// Verify no temp file left behind
	tempFilePath := keyFilePath + ".tmp"
	if _, err := os.Stat(tempFilePath); !os.IsNotExist(err) {
		t.Error("Temporary file should not exist after atomic write")
	}

	// Verify file is valid JSON
	var serverKeys ServerKeys
	if err := json.Unmarshal(newData, &serverKeys); err != nil {
		t.Errorf("File should contain valid JSON: %v", err)
	}
}

func TestServerKeyStore_FilePermissions(t *testing.T) {
	// Skip on Windows (different permission model)
	if os.Getenv("OS") == "Windows_NT" {
		t.Skip("Skipping file permission test on Windows")
	}

	// Create temporary directory
	tempDir := t.TempDir()

	// Create key store
	_, err := NewServerKeyStore(tempDir)
	if err != nil {
		t.Fatalf("Failed to create key store: %v", err)
	}

	// Check file permissions
	keyFilePath := filepath.Join(tempDir, "server-key.json")
	fileInfo, err := os.Stat(keyFilePath)
	if err != nil {
		t.Fatalf("Failed to stat key file: %v", err)
	}

	// Verify permissions are 0600 (owner read/write only)
	expectedPerms := os.FileMode(0600)
	actualPerms := fileInfo.Mode().Perm()
	if actualPerms != expectedPerms {
		t.Errorf("Expected permissions %v, got %v", expectedPerms, actualPerms)
	}
}

func TestServerKeyStore_ConcurrentAccess(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()

	// Create key store
	store, err := NewServerKeyStore(tempDir)
	if err != nil {
		t.Fatalf("Failed to create key store: %v", err)
	}

	// Launch multiple goroutines to read keys concurrently
	var wg sync.WaitGroup
	numGoroutines := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Read keys multiple times
			for j := 0; j < 10; j++ {
				_ = store.GetPrivateKey()
				_ = store.GetPublicKey()
				_ = store.GetKeys()
			}
		}()
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// If we get here without panicking, concurrent access is safe
}

func TestServerKeyStore_EmptyFile(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()

	// Write empty file
	keyFilePath := filepath.Join(tempDir, "server-key.json")
	if err := os.WriteFile(keyFilePath, []byte(""), 0600); err != nil {
		t.Fatalf("Failed to write empty file: %v", err)
	}

	// Try to create key store (should fail)
	_, err := NewServerKeyStore(tempDir)
	if err == nil {
		t.Error("Expected error for empty file, got nil")
	}
}

func TestServerKeyStore_MissingFields(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()

	// Write JSON with missing fields
	keyFilePath := filepath.Join(tempDir, "server-key.json")
	incompleteKeys := map[string]interface{}{
		"privateKey": "some-key",
		// Missing publicKey
	}
	data, _ := json.MarshalIndent(incompleteKeys, "", "  ")
	if err := os.WriteFile(keyFilePath, data, 0600); err != nil {
		t.Fatalf("Failed to write incomplete keys: %v", err)
	}

	// Try to create key store (should fail)
	_, err := NewServerKeyStore(tempDir)
	if err == nil {
		t.Error("Expected error for missing fields, got nil")
	}
}
