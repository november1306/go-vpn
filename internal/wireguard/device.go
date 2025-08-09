package wireguard

import (
	"crypto/rand"
	"fmt"
	"log"

	"golang.org/x/crypto/curve25519"
	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/tun"
)

// WireGuardDevice wraps the wireguard-go device with our configuration
type WireGuardDevice struct {
	device *device.Device
	tun    tun.Device
}

// NewWireGuardDevice creates a new WireGuard device with basic configuration
func NewWireGuardDevice(interfaceName string) (*WireGuardDevice, error) {
	// Create TUN interface
	tunDevice, err := tun.CreateTUN(interfaceName, 1420)
	if err != nil {
		return nil, fmt.Errorf("failed to create TUN interface: %w", err)
	}

	// Create logger for device
	logger := device.NewLogger(
		device.LogLevelVerbose,
		fmt.Sprintf("(%s) ", interfaceName),
	)

	// Create WireGuard device
	wgDevice := device.NewDevice(tunDevice, conn.NewDefaultBind(), logger)

	return &WireGuardDevice{
		device: wgDevice,
		tun:    tunDevice,
	}, nil
}

// Start brings up the WireGuard device
func (wd *WireGuardDevice) Start() error {
	if wd.device == nil {
		return fmt.Errorf("device not initialized")
	}
	if err := wd.device.Up(); err != nil {
		return fmt.Errorf("failed to start WireGuard device: %w", err)
	}
	return nil
}

// Stop brings down the WireGuard device
func (wd *WireGuardDevice) Stop() error {
	var err error
	
	// Close device first, but don't let panic prevent TUN cleanup
	if wd.device != nil {
		func() {
			defer func() {
				if r := recover(); r != nil {
					// Log panic but continue with TUN cleanup
					log.Printf("Panic during device close: %v", r)
				}
			}()
			wd.device.Close()
		}()
	}
	
	// Always attempt TUN cleanup
	if wd.tun != nil {
		if closeErr := wd.tun.Close(); closeErr != nil {
			err = fmt.Errorf("failed to close TUN interface: %w", closeErr)
		}
	}
	
	return err
}

// GenerateKeyPair generates a new WireGuard key pair using crypto/rand
func GenerateKeyPair() (privateKey, publicKey [32]byte, err error) {
	// Generate random private key
	_, err = rand.Read(privateKey[:])
	if err != nil {
		return privateKey, publicKey, fmt.Errorf("failed to generate private key: %w", err)
	}

	// Clamp private key (WireGuard key format requirements)
	privateKey[0] &= 248
	privateKey[31] &= 127
	privateKey[31] |= 64

	// Generate public key from private key using curve25519
	publicKey = generatePublicKey(privateKey)

	return privateKey, publicKey, nil
}

// generatePublicKey derives the public key from a private key using curve25519
func generatePublicKey(privateKey [32]byte) [32]byte {
	var publicKey [32]byte
	curve25519.ScalarBaseMult(&publicKey, &privateKey)
	return publicKey
}

// BasicDeviceDemo demonstrates basic device creation and key generation
func BasicDeviceDemo() error {
	log.Println("Starting WireGuard device demonstration...")

	// Generate a key pair
	_, publicKey, err := GenerateKeyPair()
	if err != nil {
		return fmt.Errorf("key generation failed: %w", err)
	}

	log.Printf("Generated key pair - Public key: %x", publicKey)
	// Private key is not logged for security reasons

	// Create device (this will fail without proper permissions, but shows the API)
	log.Println("Creating WireGuard device...")
	wgDevice, err := NewWireGuardDevice("wg-demo")
	if err != nil {
		log.Printf("Device creation failed (expected without admin privileges): %v", err)
		log.Println("This demonstrates the API structure for later use")
		return nil
	}

	// If device creation succeeded, start and stop it
	log.Println("Starting device...")
	if err := wgDevice.Start(); err != nil {
		log.Printf("Failed to start device: %v", err)
	}

	log.Println("Stopping device...")
	if err := wgDevice.Stop(); err != nil {
		log.Printf("Failed to stop device: %v", err)
	}

	log.Println("WireGuard device demonstration completed")
	return nil
}