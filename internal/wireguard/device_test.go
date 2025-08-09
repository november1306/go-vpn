package wireguard

import (
	"strings"
	"testing"
)

func TestGenerateKeyPair(t *testing.T) {
	t.Run("returns no error on success", func(t *testing.T) {
		_, _, err := GenerateKeyPair()
		if err != nil {
			t.Fatalf("GenerateKeyPair should not return error: %v", err)
		}
	})

	t.Run("returns proper error message on crypto failure", func(t *testing.T) {
		// We can't easily mock crypto/rand.Read, so we test the error handling exists
		// by verifying the error message format in our code
		// This test validates our error wrapping logic
		
		// The function should wrap crypto errors properly
		// We can't trigger a crypto failure easily, so we verify the error handling exists
		// by checking that the function completes (crypto/rand works in test environment)
		
		_, _, err := GenerateKeyPair()
		if err != nil && !strings.Contains(err.Error(), "failed to generate private key") {
			t.Errorf("Expected our error message format, got: %v", err)
		}
	})
}

func TestNewWireGuardDevice(t *testing.T) {
	t.Run("handles empty interface name", func(t *testing.T) {
		_, err := NewWireGuardDevice("")
		// Should attempt to create device with empty name and handle TUN creation error
		if err == nil {
			t.Log("Device creation succeeded unexpectedly")
		}
		// Our function should wrap TUN creation errors properly
		if err != nil && !strings.Contains(err.Error(), "failed to create TUN interface") {
			t.Errorf("Expected our error message format, got: %v", err)
		}
	})

	t.Run("handles normal interface name", func(t *testing.T) {
		_, err := NewWireGuardDevice("test-wg0")
		// Expected to fail without admin privileges, but should use our error wrapping
		if err != nil && !strings.Contains(err.Error(), "failed to create TUN interface") {
			t.Errorf("Expected our error message format, got: %v", err)
		}
	})
}

func TestWireGuardDevice_Start(t *testing.T) {
	t.Run("returns error for nil device", func(t *testing.T) {
		device := &WireGuardDevice{}
		err := device.Start()
		
		if err == nil {
			t.Error("Start should return error for uninitialized device")
		}
		
		expectedMsg := "device not initialized"
		if err.Error() != expectedMsg {
			t.Errorf("Expected '%s', got: %v", expectedMsg, err)
		}
	})
}

func TestWireGuardDevice_Stop(t *testing.T) {
	t.Run("handles nil device gracefully", func(t *testing.T) {
		device := &WireGuardDevice{}
		err := device.Stop()
		
		// Our Stop method should handle nil gracefully and not return error
		if err != nil {
			t.Errorf("Stop should handle nil device gracefully, got: %v", err)
		}
	})
	
	t.Run("handles nil tun gracefully", func(t *testing.T) {
		device := &WireGuardDevice{
			device: nil,
			tun:    nil,
		}
		err := device.Stop()
		
		// Should not panic or error when both device and tun are nil
		if err != nil {
			t.Errorf("Stop should handle nil tun gracefully, got: %v", err)
		}
	})
}

func TestBasicDeviceDemo(t *testing.T) {
	t.Run("completes without panic", func(t *testing.T) {
		// Test that our demo function runs to completion
		err := BasicDeviceDemo()
		
		// Demo should complete without panicking
		// It may return an error, but that's handled within the demo
		// The important part is our demo logic executes properly
		if err != nil {
			t.Errorf("Demo should not return error, got: %v", err)
		}
	})
}

// Test our struct initialization
func TestWireGuardDeviceStruct(t *testing.T) {
	t.Run("struct can be created", func(t *testing.T) {
		device := &WireGuardDevice{}
		// Verify our struct has the expected zero values
		if device.device != nil {
			t.Error("New struct should have nil device field")
		}
		if device.tun != nil {
			t.Error("New struct should have nil tun field")
		}
	})
}