package keys

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/curve25519"
)

// GenerateKeyPair generates a WireGuard-compatible private/public key pair.
// Returns base64-encoded private and public keys suitable for WireGuard configuration.
func GenerateKeyPair() (privateKey string, publicKey string, err error) {
	// Generate 32 random bytes for private key
	privateKeyBytes := make([]byte, 32)
	if _, err := rand.Read(privateKeyBytes); err != nil {
		return "", "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Clamp the private key according to Curve25519 requirements
	// This ensures the key is in the proper format for WireGuard
	privateKeyBytes[0] &= 248
	privateKeyBytes[31] &= 127
	privateKeyBytes[31] |= 64

	// Generate public key from private key using Curve25519
	publicKeyBytes, err := curve25519.X25519(privateKeyBytes, curve25519.Basepoint)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate public key: %w", err)
	}

	// Encode keys as base64 strings (WireGuard format)
	privateKeyB64 := base64.StdEncoding.EncodeToString(privateKeyBytes)
	publicKeyB64 := base64.StdEncoding.EncodeToString(publicKeyBytes)

	return privateKeyB64, publicKeyB64, nil
}

// ValidatePrivateKey validates that a base64-encoded private key is properly formatted
func ValidatePrivateKey(privateKey string) error {
	keyBytes, err := base64.StdEncoding.DecodeString(privateKey)
	if err != nil {
		return fmt.Errorf("invalid base64 encoding: %w", err)
	}

	if len(keyBytes) != 32 {
		return fmt.Errorf("private key must be exactly 32 bytes, got %d", len(keyBytes))
	}

	return nil
}

// ValidatePublicKey validates that a base64-encoded public key is properly formatted
func ValidatePublicKey(publicKey string) error {
	keyBytes, err := base64.StdEncoding.DecodeString(publicKey)
	if err != nil {
		return fmt.Errorf("invalid base64 encoding: %w", err)
	}

	if len(keyBytes) != 32 {
		return fmt.Errorf("public key must be exactly 32 bytes, got %d", len(keyBytes))
	}

	return nil
}

// PublicKeyFromPrivate derives the public key from a given private key
func PublicKeyFromPrivate(privateKey string) (string, error) {
	privateKeyBytes, err := base64.StdEncoding.DecodeString(privateKey)
	if err != nil {
		return "", fmt.Errorf("invalid private key base64: %w", err)
	}

	if len(privateKeyBytes) != 32 {
		return "", fmt.Errorf("private key must be exactly 32 bytes, got %d", len(privateKeyBytes))
	}

	// Generate public key from private key
	publicKeyBytes, err := curve25519.X25519(privateKeyBytes, curve25519.Basepoint)
	if err != nil {
		return "", fmt.Errorf("failed to derive public key: %w", err)
	}

	return base64.StdEncoding.EncodeToString(publicKeyBytes), nil
}
