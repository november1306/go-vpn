package vpnserver

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// PeerConfig represents a persisted peer configuration
type PeerConfig struct {
	PublicKey    string    `json:"publicKey"`
	AllowedIPs   string    `json:"allowedIPs"`
	RegisteredAt time.Time `json:"registeredAt"`
}

// PeerStore manages persistent storage of WireGuard peer configurations
// This ensures peers survive server restarts - following WireGuard best practices
type PeerStore struct {
	mu       sync.RWMutex
	peers    map[string]*PeerConfig
	filePath string
}

// NewPeerStore creates a new peer store with the specified storage file
func NewPeerStore(dataDir string) (*PeerStore, error) {
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	filePath := filepath.Join(dataDir, "peers.json")
	
	store := &PeerStore{
		peers:    make(map[string]*PeerConfig),
		filePath: filePath,
	}

	// Load existing peers
	if err := store.load(); err != nil {
		return nil, fmt.Errorf("failed to load peer store: %w", err)
	}

	return store, nil
}

// AddPeer adds a peer configuration to persistent storage
func (ps *PeerStore) AddPeer(publicKey, allowedIPs string) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	ps.peers[publicKey] = &PeerConfig{
		PublicKey:    publicKey,
		AllowedIPs:   allowedIPs,
		RegisteredAt: time.Now(),
	}

	return ps.save()
}

// RemovePeer removes a peer from persistent storage
func (ps *PeerStore) RemovePeer(publicKey string) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	delete(ps.peers, publicKey)
	return ps.save()
}

// GetPeer retrieves a peer configuration
func (ps *PeerStore) GetPeer(publicKey string) (*PeerConfig, bool) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	peer, exists := ps.peers[publicKey]
	return peer, exists
}

// ListPeers returns all registered peers
func (ps *PeerStore) ListPeers() map[string]*PeerConfig {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	// Return a copy to avoid race conditions
	result := make(map[string]*PeerConfig)
	for k, v := range ps.peers {
		result[k] = v
	}
	return result
}

// load reads peer configurations from disk
func (ps *PeerStore) load() error {
	if _, err := os.Stat(ps.filePath); os.IsNotExist(err) {
		// File doesn't exist yet, that's okay
		return nil
	}

	data, err := os.ReadFile(ps.filePath)
	if err != nil {
		return fmt.Errorf("failed to read peer store file: %w", err)
	}

	if len(data) == 0 {
		// Empty file, that's okay
		return nil
	}

	var peers map[string]*PeerConfig
	if err := json.Unmarshal(data, &peers); err != nil {
		return fmt.Errorf("failed to parse peer store file: %w", err)
	}

	ps.peers = peers
	return nil
}

// save writes peer configurations to disk
func (ps *PeerStore) save() error {
	data, err := json.MarshalIndent(ps.peers, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal peer store: %w", err)
	}

	// Write to temporary file first, then rename (atomic operation)
	tempPath := ps.filePath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write temporary peer store file: %w", err)
	}

	if err := os.Rename(tempPath, ps.filePath); err != nil {
		os.Remove(tempPath) // Clean up temp file
		return fmt.Errorf("failed to replace peer store file: %w", err)
	}

	return nil
}

// Count returns the number of registered peers
func (ps *PeerStore) Count() int {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return len(ps.peers)
}