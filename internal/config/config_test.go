package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	// Test default configuration
	config := Load()
	
	// Verify server defaults
	if config.Server.APIPort != 8443 {
		t.Errorf("Expected API port 8443, got %d", config.Server.APIPort)
	}
	if config.Server.VPNPort != 51820 {
		t.Errorf("Expected VPN port 51820, got %d", config.Server.VPNPort)
	}
	if config.Server.InterfaceName != "wg0" {
		t.Errorf("Expected interface wg0, got %s", config.Server.InterfaceName)
	}
	
	// Verify network defaults
	if config.Network.ServerIP != "10.0.0.1/24" {
		t.Errorf("Expected server IP 10.0.0.1/24, got %s", config.Network.ServerIP)
	}
	if config.Network.IPAMCIDR != "10.0.0.0/24" {
		t.Errorf("Expected IPAM CIDR 10.0.0.0/24, got %s", config.Network.IPAMCIDR)
	}
	if config.Network.IPAMGateway != "10.0.0.1" {
		t.Errorf("Expected IPAM gateway 10.0.0.1, got %s", config.Network.IPAMGateway)
	}
	
	// Verify timeout defaults
	if config.Timeouts.HTTPRead != 15*time.Second {
		t.Errorf("Expected HTTP read timeout 15s, got %v", config.Timeouts.HTTPRead)
	}
	if config.Timeouts.Shutdown != 10*time.Second {
		t.Errorf("Expected shutdown timeout 10s, got %v", config.Timeouts.Shutdown)
	}
	
	// Verify test defaults
	if config.Test.PeerIP != "10.0.0.2" {
		t.Errorf("Expected test peer IP 10.0.0.2, got %s", config.Test.PeerIP)
	}
	if config.Test.InterfaceName != "wg-test" {
		t.Errorf("Expected test interface wg-test, got %s", config.Test.InterfaceName)
	}
}

func TestLoadWithEnvironmentVariables(t *testing.T) {
	// Set environment variables
	os.Setenv("VPN_API_PORT", "9443")
	os.Setenv("VPN_LISTEN_PORT", "51821")
	os.Setenv("VPN_INTERFACE", "wg1")
	os.Setenv("VPN_SERVER_IP", "192.168.1.1/24")
	os.Setenv("VPN_HTTP_READ_TIMEOUT", "30s")
	os.Setenv("VPN_TEST_PEER_IP", "192.168.1.10")
	
	defer func() {
		// Clean up environment variables
		os.Unsetenv("VPN_API_PORT")
		os.Unsetenv("VPN_LISTEN_PORT")
		os.Unsetenv("VPN_INTERFACE")
		os.Unsetenv("VPN_SERVER_IP")
		os.Unsetenv("VPN_HTTP_READ_TIMEOUT")
		os.Unsetenv("VPN_TEST_PEER_IP")
	}()
	
	config := Load()
	
	// Verify environment variables override defaults
	if config.Server.APIPort != 9443 {
		t.Errorf("Expected API port 9443, got %d", config.Server.APIPort)
	}
	if config.Server.VPNPort != 51821 {
		t.Errorf("Expected VPN port 51821, got %d", config.Server.VPNPort)
	}
	if config.Server.InterfaceName != "wg1" {
		t.Errorf("Expected interface wg1, got %s", config.Server.InterfaceName)
	}
	if config.Network.ServerIP != "192.168.1.1/24" {
		t.Errorf("Expected server IP 192.168.1.1/24, got %s", config.Network.ServerIP)
	}
	if config.Timeouts.HTTPRead != 30*time.Second {
		t.Errorf("Expected HTTP read timeout 30s, got %v", config.Timeouts.HTTPRead)
	}
	if config.Test.PeerIP != "192.168.1.10" {
		t.Errorf("Expected test peer IP 192.168.1.10, got %s", config.Test.PeerIP)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name:    "valid default config",
			config:  *Load(),
			wantErr: false,
		},
		{
			name: "invalid API port - too high",
			config: Config{
				Server: ServerConfig{APIPort: 70000, VPNPort: 51820, InterfaceName: "wg0"},
				Network: NetworkConfig{
					ServerIP: "10.0.0.1/24", IPAMCIDR: "10.0.0.0/24", IPAMGateway: "10.0.0.1",
				},
				Timeouts: TimeoutConfig{HTTPRead: 15 * time.Second, HTTPWrite: 15 * time.Second, Shutdown: 10 * time.Second},
			},
			wantErr: true,
		},
		{
			name: "invalid API port - zero",
			config: Config{
				Server: ServerConfig{APIPort: 0, VPNPort: 51820, InterfaceName: "wg0"},
				Network: NetworkConfig{
					ServerIP: "10.0.0.1/24", IPAMCIDR: "10.0.0.0/24", IPAMGateway: "10.0.0.1",
				},
				Timeouts: TimeoutConfig{HTTPRead: 15 * time.Second, HTTPWrite: 15 * time.Second, Shutdown: 10 * time.Second},
			},
			wantErr: true,
		},
		{
			name: "empty interface name",
			config: Config{
				Server: ServerConfig{APIPort: 8443, VPNPort: 51820, InterfaceName: ""},
				Network: NetworkConfig{
					ServerIP: "10.0.0.1/24", IPAMCIDR: "10.0.0.0/24", IPAMGateway: "10.0.0.1",
				},
				Timeouts: TimeoutConfig{HTTPRead: 15 * time.Second, HTTPWrite: 15 * time.Second, Shutdown: 10 * time.Second},
			},
			wantErr: true,
		},
		{
			name: "empty server IP",
			config: Config{
				Server: ServerConfig{APIPort: 8443, VPNPort: 51820, InterfaceName: "wg0"},
				Network: NetworkConfig{
					ServerIP: "", IPAMCIDR: "10.0.0.0/24", IPAMGateway: "10.0.0.1",
				},
				Timeouts: TimeoutConfig{HTTPRead: 15 * time.Second, HTTPWrite: 15 * time.Second, Shutdown: 10 * time.Second},
			},
			wantErr: true,
		},
		{
			name: "zero timeout",
			config: Config{
				Server: ServerConfig{APIPort: 8443, VPNPort: 51820, InterfaceName: "wg0"},
				Network: NetworkConfig{
					ServerIP: "10.0.0.1/24", IPAMCIDR: "10.0.0.0/24", IPAMGateway: "10.0.0.1",
				},
				Timeouts: TimeoutConfig{HTTPRead: 0, HTTPWrite: 15 * time.Second, Shutdown: 10 * time.Second},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetEnvHelpers(t *testing.T) {
	// Test getEnvString
	os.Setenv("TEST_STRING", "test_value")
	if val := getEnvString("TEST_STRING", "default"); val != "test_value" {
		t.Errorf("getEnvString() = %v, want test_value", val)
	}
	if val := getEnvString("NONEXISTENT", "default"); val != "default" {
		t.Errorf("getEnvString() = %v, want default", val)
	}
	os.Unsetenv("TEST_STRING")
	
	// Test getEnvInt
	os.Setenv("TEST_INT", "123")
	if val := getEnvInt("TEST_INT", 456); val != 123 {
		t.Errorf("getEnvInt() = %v, want 123", val)
	}
	if val := getEnvInt("NONEXISTENT", 456); val != 456 {
		t.Errorf("getEnvInt() = %v, want 456", val)
	}
	os.Setenv("TEST_INT", "invalid")
	if val := getEnvInt("TEST_INT", 456); val != 456 {
		t.Errorf("getEnvInt() with invalid value = %v, want 456", val)
	}
	os.Unsetenv("TEST_INT")
	
	// Test getEnvDuration
	os.Setenv("TEST_DURATION", "5m")
	if val := getEnvDuration("TEST_DURATION", 10*time.Second); val != 5*time.Minute {
		t.Errorf("getEnvDuration() = %v, want 5m", val)
	}
	if val := getEnvDuration("NONEXISTENT", 10*time.Second); val != 10*time.Second {
		t.Errorf("getEnvDuration() = %v, want 10s", val)
	}
	os.Setenv("TEST_DURATION", "invalid")
	if val := getEnvDuration("TEST_DURATION", 10*time.Second); val != 10*time.Second {
		t.Errorf("getEnvDuration() with invalid value = %v, want 10s", val)
	}
	os.Unsetenv("TEST_DURATION")
}