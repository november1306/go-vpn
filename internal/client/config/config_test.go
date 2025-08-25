package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/november1306/go-vpn/internal/wireguard/keys"
)

func TestConfigRoundTrip(t *testing.T) {
	// Create temporary config for testing
	originalPath := os.Getenv("HOME")
	tempDir, err := os.MkdirTemp("", "go-vpn-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Override home directory for testing
	os.Setenv("HOME", tempDir)
	if runtime.GOOS == "windows" {
		os.Setenv("USERPROFILE", tempDir)
	}
	defer func() {
		os.Setenv("HOME", originalPath)
		if runtime.GOOS == "windows" {
			os.Setenv("USERPROFILE", originalPath)
		}
	}()

	// Generate test key pair
	clientPrivKey, clientPubKey, err := keys.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate keys: %v", err)
	}

	// Create test configuration
	now := time.Now()
	testConfig := &ClientConfig{
		ClientPrivateKey: clientPrivKey,
		ClientPublicKey:  clientPubKey,
		ServerPublicKey:  "test-server-public-key-base64-encoded-32bytes",
		ServerEndpoint:   "vpn.example.com:51820",
		ClientIP:         "10.0.0.2/32",
		RegisteredAt:     now,
	}

	// Test Save
	if err := Save(testConfig); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Test Load
	loadedConfig, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify all fields match
	if loadedConfig.ClientPrivateKey != testConfig.ClientPrivateKey {
		t.Errorf("ClientPrivateKey mismatch: got %s, want %s", 
			loadedConfig.ClientPrivateKey, testConfig.ClientPrivateKey)
	}

	if loadedConfig.ClientPublicKey != testConfig.ClientPublicKey {
		t.Errorf("ClientPublicKey mismatch: got %s, want %s", 
			loadedConfig.ClientPublicKey, testConfig.ClientPublicKey)
	}

	if loadedConfig.ServerPublicKey != testConfig.ServerPublicKey {
		t.Errorf("ServerPublicKey mismatch: got %s, want %s", 
			loadedConfig.ServerPublicKey, testConfig.ServerPublicKey)
	}

	if loadedConfig.ServerEndpoint != testConfig.ServerEndpoint {
		t.Errorf("ServerEndpoint mismatch: got %s, want %s", 
			loadedConfig.ServerEndpoint, testConfig.ServerEndpoint)
	}

	if loadedConfig.ClientIP != testConfig.ClientIP {
		t.Errorf("ClientIP mismatch: got %s, want %s", 
			loadedConfig.ClientIP, testConfig.ClientIP)
	}

	// IsConnected field removed - connection state is runtime-only

	// Test timestamps (allow small differences due to JSON marshaling)
	timeDiff := loadedConfig.RegisteredAt.Sub(testConfig.RegisteredAt)
	if timeDiff > time.Second || timeDiff < -time.Second {
		t.Errorf("RegisteredAt time difference too large: %v", timeDiff)
	}
}

func TestConfigFilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping permission test on Windows")
	}

	// Create temporary config for testing
	tempDir, err := os.MkdirTemp("", "go-vpn-perm-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Override home directory for testing
	originalPath := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalPath)

	// Generate test keys
	clientPrivKey, clientPubKey, err := keys.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate keys: %v", err)
	}

	// Create and save test configuration
	testConfig := &ClientConfig{
		ClientPrivateKey: clientPrivKey,
		ClientPublicKey:  clientPubKey,
		ServerPublicKey:  "test-server-key",
		ServerEndpoint:   "test.example.com:51820",
		ClientIP:         "10.0.0.2/32",
		RegisteredAt:     time.Now(),
	}

	if err := Save(testConfig); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Check file permissions
	configPath, err := GetConfigPath()
	if err != nil {
		t.Fatalf("Failed to get config path: %v", err)
	}

	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("Failed to stat config file: %v", err)
	}

	mode := info.Mode()
	expectedMode := os.FileMode(0600)

	if mode != expectedMode {
		t.Errorf("Config file has wrong permissions: got %o, want %o", mode, expectedMode)
	}

	// Check directory permissions
	configDir := filepath.Dir(configPath)
	dirInfo, err := os.Stat(configDir)
	if err != nil {
		t.Fatalf("Failed to stat config directory: %v", err)
	}

	dirMode := dirInfo.Mode()
	expectedDirMode := os.FileMode(0700) | os.ModeDir

	if dirMode != expectedDirMode {
		t.Errorf("Config directory has wrong permissions: got %o, want %o", dirMode, expectedDirMode)
	}
}

func TestLoadNonExistentConfig(t *testing.T) {
	// Create temporary directory without config file
	tempDir, err := os.MkdirTemp("", "go-vpn-noconfig-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Override home directory for testing
	originalPath := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	if runtime.GOOS == "windows" {
		os.Setenv("USERPROFILE", tempDir)
	}
	defer func() {
		os.Setenv("HOME", originalPath)
		if runtime.GOOS == "windows" {
			os.Setenv("USERPROFILE", originalPath)
		}
	}()

	// Try to load non-existent config
	_, err = Load()
	if err == nil {
		t.Error("Expected error when loading non-existent config, got nil")
	}

	expectedMsg := "configuration file not found"
	if !contains(err.Error(), expectedMsg) {
		t.Errorf("Expected error message to contain '%s', got: %s", expectedMsg, err.Error())
	}
}

func TestConfigExists(t *testing.T) {
	// Create temporary config for testing
	tempDir, err := os.MkdirTemp("", "go-vpn-exists-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Override home directory for testing
	originalPath := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	if runtime.GOOS == "windows" {
		os.Setenv("USERPROFILE", tempDir)
	}
	defer func() {
		os.Setenv("HOME", originalPath)
		if runtime.GOOS == "windows" {
			os.Setenv("USERPROFILE", originalPath)
		}
	}()

	// Initially should not exist
	if Exists() {
		t.Error("Config should not exist initially")
	}

	// Generate test keys and save config
	clientPrivKey, clientPubKey, err := keys.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate keys: %v", err)
	}

	testConfig := &ClientConfig{
		ClientPrivateKey: clientPrivKey,
		ClientPublicKey:  clientPubKey,
		ServerPublicKey:  "test-server-key",
		ServerEndpoint:   "test.example.com:51820",
		ClientIP:         "10.0.0.2/32",
		RegisteredAt:     time.Now(),
	}

	if err := Save(testConfig); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Now should exist
	if !Exists() {
		t.Error("Config should exist after saving")
	}
}

func TestDeleteConfig(t *testing.T) {
	// Create temporary config for testing
	tempDir, err := os.MkdirTemp("", "go-vpn-delete-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Override home directory for testing
	originalPath := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	if runtime.GOOS == "windows" {
		os.Setenv("USERPROFILE", tempDir)
	}
	defer func() {
		os.Setenv("HOME", originalPath)
		if runtime.GOOS == "windows" {
			os.Setenv("USERPROFILE", originalPath)
		}
	}()

	// Generate test keys and save config
	clientPrivKey, clientPubKey, err := keys.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate keys: %v", err)
	}

	testConfig := &ClientConfig{
		ClientPrivateKey: clientPrivKey,
		ClientPublicKey:  clientPubKey,
		ServerPublicKey:  "test-server-key",
		ServerEndpoint:   "test.example.com:51820",
		ClientIP:         "10.0.0.2/32",
		RegisteredAt:     time.Now(),
	}

	if err := Save(testConfig); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Verify config exists
	if !Exists() {
		t.Error("Config should exist before deletion")
	}

	// Delete config
	if err := Delete(); err != nil {
		t.Fatalf("Failed to delete config: %v", err)
	}

	// Verify config no longer exists
	if Exists() {
		t.Error("Config should not exist after deletion")
	}

	// Delete again should not error
	if err := Delete(); err != nil {
		t.Errorf("Second delete should not error: %v", err)
	}
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