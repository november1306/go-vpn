package keystore

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/november1306/go-vpn/internal/wireguard/keys"
)

// ServerKeys represents the server's WireGuard key pair
type ServerKeys struct {
	PrivateKey  string    `json:"privateKey"`
	PublicKey   string    `json:"publicKey"`
	GeneratedAt time.Time `json:"generatedAt"`
}

// ServerKeyStore manages persistent storage of server WireGuard keys
// This ensures the server maintains the same cryptographic identity across restarts
type ServerKeyStore struct {
	mu       sync.RWMutex
	keys     *ServerKeys
	filePath string
}

// NewServerKeyStore creates or loads server keys from persistent storage
// If keys don't exist, generates new ones and saves them
func NewServerKeyStore(dataDir string) (*ServerKeyStore, error) {
	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	filePath := filepath.Join(dataDir, "server-key.json")

	store := &ServerKeyStore{
		filePath: filePath,
	}

	// Try to load existing keys
	if err := store.load(); err != nil {
		// If file doesn't exist, generate new keys
		if os.IsNotExist(err) {
			slog.Info("Server key file not found, generating new keys")
			if err := store.generateAndSave(); err != nil {
				return nil, fmt.Errorf("failed to generate server keys: %w", err)
			}
			slog.Info("Generated new server key pair", "publicKey", store.keys.PublicKey)
		} else {
			return nil, fmt.Errorf("failed to load server keys: %w - delete %s to regenerate", err, filePath)
		}
	} else {
		slog.Info("Loaded existing server key", "publicKey", store.keys.PublicKey, "generatedAt", store.keys.GeneratedAt.Format(time.RFC3339))
	}

	// Validate loaded or generated keys
	if err := keys.ValidatePrivateKey(store.keys.PrivateKey); err != nil {
		return nil, fmt.Errorf("invalid server private key in storage: %w - delete %s to regenerate", err, filePath)
	}

	// Check file permissions (Unix only)
	if fileInfo, err := os.Stat(filePath); err == nil {
		mode := fileInfo.Mode()
		// On Unix, check if permissions are too permissive
		if mode.Perm()&0077 != 0 {
			slog.Warn("Server key file has insecure permissions", "path", filePath, "permissions", mode.Perm())
		}
	}

	return store, nil
}

// GetKeys returns the server's key pair (thread-safe)
func (ks *ServerKeyStore) GetKeys() ServerKeys {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	return *ks.keys
}

// GetPrivateKey returns just the private key (convenience method)
func (ks *ServerKeyStore) GetPrivateKey() string {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	return ks.keys.PrivateKey
}

// GetPublicKey returns just the public key (convenience method)
func (ks *ServerKeyStore) GetPublicKey() string {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	return ks.keys.PublicKey
}

// RegenerateKeys generates new keys and saves them (for key rotation)
func (ks *ServerKeyStore) RegenerateKeys() error {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	slog.Info("Regenerating server keys")

	privateKey, publicKey, err := keys.GenerateKeyPair()
	if err != nil {
		return fmt.Errorf("failed to generate key pair: %w", err)
	}

	ks.keys = &ServerKeys{
		PrivateKey:  privateKey,
		PublicKey:   publicKey,
		GeneratedAt: time.Now(),
	}

	if err := ks.save(); err != nil {
		return fmt.Errorf("failed to save regenerated keys: %w", err)
	}

	slog.Info("Server keys regenerated successfully", "publicKey", publicKey)
	return nil
}

// load reads server keys from disk
func (ks *ServerKeyStore) load() error {
	data, err := os.ReadFile(ks.filePath)
	if err != nil {
		return err // Return raw error to check for IsNotExist
	}

	if len(data) == 0 {
		return fmt.Errorf("server key file is empty")
	}

	var serverKeys ServerKeys
	if err := json.Unmarshal(data, &serverKeys); err != nil {
		return fmt.Errorf("failed to parse server key file: %w", err)
	}

	// Validate that we have both keys
	if serverKeys.PrivateKey == "" || serverKeys.PublicKey == "" {
		return fmt.Errorf("server key file is missing required fields")
	}

	ks.keys = &serverKeys
	return nil
}

// save writes server keys to disk with atomic operation
func (ks *ServerKeyStore) save() error {
	data, err := json.MarshalIndent(ks.keys, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal server keys: %w", err)
	}

	// Write to temporary file first, then rename (atomic operation)
	tempPath := ks.filePath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write temporary server key file: %w", err)
	}

	if err := os.Rename(tempPath, ks.filePath); err != nil {
		os.Remove(tempPath) // Clean up temp file
		return fmt.Errorf("failed to replace server key file: %w", err)
	}

	return nil
}

// generateAndSave generates new keys and saves them to disk
func (ks *ServerKeyStore) generateAndSave() error {
	privateKey, publicKey, err := keys.GenerateKeyPair()
	if err != nil {
		return fmt.Errorf("failed to generate key pair: %w", err)
	}

	ks.keys = &ServerKeys{
		PrivateKey:  privateKey,
		PublicKey:   publicKey,
		GeneratedAt: time.Now(),
	}

	if err := ks.save(); err != nil {
		return fmt.Errorf("failed to save server keys: %w", err)
	}

	return nil
}
