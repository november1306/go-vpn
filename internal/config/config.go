package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration
type Config struct {
	Server   ServerConfig  `json:"server"`
	Network  NetworkConfig `json:"network"`
	Timeouts TimeoutConfig `json:"timeouts"`
	Test     TestConfig    `json:"test"`
}

// ServerConfig contains HTTP server settings
type ServerConfig struct {
	APIPort        int    `json:"apiPort"`        // HTTP API port (default: 8443)
	VPNPort        int    `json:"vpnPort"`        // WireGuard UDP port (default: 51820)
	InterfaceName  string `json:"interfaceName"`  // WireGuard interface name (default: "wg0")
	PublicEndpoint string `json:"publicEndpoint"` // Public endpoint for clients (e.g., "1.2.3.4:51820")
}

// NetworkConfig contains VPN network settings
type NetworkConfig struct {
	ServerIP     string `json:"serverIP"`     // VPN server IP with CIDR (default: "10.0.0.1/24")
	IPAMCIDR     string `json:"ipamCIDR"`     // IP allocation range (default: "10.0.0.0/24")
	IPAMGateway  string `json:"ipamGateway"`  // Gateway IP (default: "10.0.0.1")
	ClientIPDemo string `json:"clientIPDemo"` // Demo client IP for registration (default: "10.0.0.100")
}

// TimeoutConfig contains timeout settings
type TimeoutConfig struct {
	HTTPRead    time.Duration `json:"httpRead"`    // HTTP read timeout (default: 15s)
	HTTPWrite   time.Duration `json:"httpWrite"`   // HTTP write timeout (default: 15s)
	HTTPIdle    time.Duration `json:"httpIdle"`    // HTTP idle timeout (default: 60s)
	Shutdown    time.Duration `json:"shutdown"`    // Graceful shutdown timeout (default: 10s)
	TestContext time.Duration `json:"testContext"` // Test context timeout (default: 30s)
}

// TestConfig contains test-specific settings
type TestConfig struct {
	PeerPublicKey string `json:"peerPublicKey"` // Hardcoded test peer public key
	PeerIP        string `json:"peerIP"`        // Hardcoded test peer IP (default: "10.0.0.2")
	InterfaceName string `json:"interfaceName"` // Test interface name (default: "wg-test")
}

// Load creates a Config with values from environment variables and defaults
func Load() *Config {
	return &Config{
		Server: ServerConfig{
			APIPort:        getEnvInt("PORT", getEnvInt("VPN_API_PORT", 8443)),
			VPNPort:        getEnvInt("VPN_LISTEN_PORT", 51820),
			InterfaceName:  getEnvString("VPN_INTERFACE", "wg0"),
			PublicEndpoint: getEnvString("VPN_PUBLIC_ENDPOINT", ""),
		},
		Network: NetworkConfig{
			ServerIP:     getEnvString("VPN_SERVER_IP", "10.0.0.1/24"),
			IPAMCIDR:     getEnvString("VPN_IPAM_CIDR", "10.0.0.0/24"),
			IPAMGateway:  getEnvString("VPN_IPAM_GATEWAY", "10.0.0.1"),
			ClientIPDemo: getEnvString("VPN_CLIENT_IP_DEMO", "10.0.0.100"),
		},
		Timeouts: TimeoutConfig{
			HTTPRead:    getEnvDuration("VPN_HTTP_READ_TIMEOUT", 15*time.Second),
			HTTPWrite:   getEnvDuration("VPN_HTTP_WRITE_TIMEOUT", 15*time.Second),
			HTTPIdle:    getEnvDuration("VPN_HTTP_IDLE_TIMEOUT", 60*time.Second),
			Shutdown:    getEnvDuration("VPN_SHUTDOWN_TIMEOUT", 10*time.Second),
			TestContext: getEnvDuration("VPN_TEST_CONTEXT_TIMEOUT", 30*time.Second),
		},
		Test: TestConfig{
			PeerPublicKey: getEnvString("VPN_TEST_PEER_PUBKEY", ""),
			PeerIP:        getEnvString("VPN_TEST_PEER_IP", "10.0.0.2"),
			InterfaceName: getEnvString("VPN_TEST_INTERFACE", "wg-test"),
		},
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Validate ports
	if c.Server.APIPort <= 0 || c.Server.APIPort > 65535 {
		return fmt.Errorf("invalid API port: %d", c.Server.APIPort)
	}
	if c.Server.VPNPort <= 0 || c.Server.VPNPort > 65535 {
		return fmt.Errorf("invalid VPN port: %d", c.Server.VPNPort)
	}

	// Validate interface names
	if c.Server.InterfaceName == "" {
		return fmt.Errorf("interface name cannot be empty")
	}

	// Validate network settings
	if c.Network.ServerIP == "" {
		return fmt.Errorf("server IP cannot be empty")
	}
	if c.Network.IPAMCIDR == "" {
		return fmt.Errorf("IPAM CIDR cannot be empty")
	}
	if c.Network.IPAMGateway == "" {
		return fmt.Errorf("IPAM gateway cannot be empty")
	}

	// Validate timeouts
	if c.Timeouts.HTTPRead <= 0 {
		return fmt.Errorf("HTTP read timeout must be positive")
	}
	if c.Timeouts.HTTPWrite <= 0 {
		return fmt.Errorf("HTTP write timeout must be positive")
	}
	if c.Timeouts.Shutdown <= 0 {
		return fmt.Errorf("shutdown timeout must be positive")
	}

	return nil
}

// getEnvString returns environment variable value or default
func getEnvString(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// getEnvInt returns environment variable as int or default
func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultVal
}

// getEnvDuration returns environment variable as duration or default
func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if duration, err := time.ParseDuration(val); err == nil {
			return duration
		}
	}
	return defaultVal
}
