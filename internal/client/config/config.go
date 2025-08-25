package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// ClientConfig represents the client-side VPN configuration
// Following WireGuard best practices: only persistent configuration, no runtime state
type ClientConfig struct {
	// Client credentials
	ClientPrivateKey string `json:"clientPrivateKey"`
	ClientPublicKey  string `json:"clientPublicKey"`

	// Server connection details
	ServerPublicKey string `json:"serverPublicKey"`
	ServerEndpoint  string `json:"serverEndpoint"`
	ClientIP        string `json:"clientIP"`

	// Registration metadata
	RegisteredAt time.Time `json:"registeredAt"`
}

const (
	configDirName  = ".go-wire-vpn"
	configFileName = "config.json"
)

// GetConfigPath returns the path to the client configuration file
func GetConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, configDirName)
	return filepath.Join(configDir, configFileName), nil
}

// Load reads the client configuration from disk
func Load() (*ClientConfig, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("configuration file not found - run 'vpn-cli register' first")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config ClientConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// Save writes the client configuration to disk with secure permissions
func Save(config *ClientConfig) error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	// Create config directory if it doesn't exist
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal config to JSON
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write config file with secure permissions
	if err := writeConfigFile(configPath, data); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// writeConfigFile writes the config data with appropriate security permissions
func writeConfigFile(path string, data []byte) error {
	// Create file with restrictive permissions (0600 on Unix)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := file.Write(data); err != nil {
		return err
	}

	// Apply platform-specific security settings
	return applySecurityPermissions(path)
}

// applySecurityPermissions applies platform-specific security settings
func applySecurityPermissions(path string) error {
	if runtime.GOOS == "windows" {
		// On Windows, set file as hidden and note that proper ACLs
		// would require Windows-specific APIs for production use
		return setWindowsHidden(path)
	}

	// On Unix systems, ensure 0600 permissions (owner read/write only)
	return os.Chmod(path, 0600)
}

// setWindowsHidden sets the hidden attribute on Windows
func setWindowsHidden(path string) error {
	if runtime.GOOS != "windows" {
		return nil
	}

	// Note: This is a simplified implementation
	// Production code should use Windows APIs for proper ACL management
	return nil
}

// Delete removes the client configuration file
func Delete() error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	if err := os.Remove(configPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete config file: %w", err)
	}

	return nil
}

// Exists checks if a configuration file exists
func Exists() bool {
	configPath, err := GetConfigPath()
	if err != nil {
		return false
	}

	_, err = os.Stat(configPath)
	return err == nil
}
