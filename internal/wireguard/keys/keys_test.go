package keys

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestGenerateKeyPair(t *testing.T) {
	t.Run("generates valid key pair", func(t *testing.T) {
		privateKey, publicKey, err := GenerateKeyPair()
		if err != nil {
			t.Fatalf("GenerateKeyPair() failed: %v", err)
		}

		// Test key lengths
		if len(privateKey) == 0 {
			t.Error("private key is empty")
		}
		if len(publicKey) == 0 {
			t.Error("public key is empty")
		}

		// Test base64 encoding
		if _, err := base64.StdEncoding.DecodeString(privateKey); err != nil {
			t.Errorf("private key is not valid base64: %v", err)
		}
		if _, err := base64.StdEncoding.DecodeString(publicKey); err != nil {
			t.Errorf("public key is not valid base64: %v", err)
		}

		// Test key byte lengths
		privBytes, _ := base64.StdEncoding.DecodeString(privateKey)
		pubBytes, _ := base64.StdEncoding.DecodeString(publicKey)

		if len(privBytes) != 32 {
			t.Errorf("private key should be 32 bytes, got %d", len(privBytes))
		}
		if len(pubBytes) != 32 {
			t.Errorf("public key should be 32 bytes, got %d", len(pubBytes))
		}
	})

	t.Run("generates unique keys", func(t *testing.T) {
		priv1, pub1, err1 := GenerateKeyPair()
		priv2, pub2, err2 := GenerateKeyPair()

		if err1 != nil || err2 != nil {
			t.Fatalf("GenerateKeyPair() failed: %v, %v", err1, err2)
		}

		if priv1 == priv2 {
			t.Error("generated identical private keys (should be unique)")
		}
		if pub1 == pub2 {
			t.Error("generated identical public keys (should be unique)")
		}
	})

	t.Run("private key clamping", func(t *testing.T) {
		privateKey, _, err := GenerateKeyPair()
		if err != nil {
			t.Fatalf("GenerateKeyPair() failed: %v", err)
		}

		privBytes, _ := base64.StdEncoding.DecodeString(privateKey)

		// Check Curve25519 clamping requirements
		if privBytes[0]&7 != 0 {
			t.Error("private key[0] should have bottom 3 bits cleared")
		}
		if privBytes[31]&128 != 0 {
			t.Error("private key[31] should have top bit cleared")
		}
		if privBytes[31]&64 == 0 {
			t.Error("private key[31] should have second-to-top bit set")
		}
	})
}

func TestValidatePrivateKey(t *testing.T) {
	t.Run("valid private key", func(t *testing.T) {
		privateKey, _, err := GenerateKeyPair()
		if err != nil {
			t.Fatalf("GenerateKeyPair() failed: %v", err)
		}

		if err := ValidatePrivateKey(privateKey); err != nil {
			t.Errorf("ValidatePrivateKey() failed for valid key: %v", err)
		}
	})

	t.Run("invalid base64", func(t *testing.T) {
		err := ValidatePrivateKey("invalid-base64!")
		if err == nil {
			t.Error("ValidatePrivateKey() should fail for invalid base64")
		}
		if !strings.Contains(err.Error(), "invalid base64 encoding") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("wrong length", func(t *testing.T) {
		shortKey := base64.StdEncoding.EncodeToString(make([]byte, 16)) // 16 bytes instead of 32
		err := ValidatePrivateKey(shortKey)
		if err == nil {
			t.Error("ValidatePrivateKey() should fail for wrong length")
		}
		if !strings.Contains(err.Error(), "must be exactly 32 bytes") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("empty key", func(t *testing.T) {
		err := ValidatePrivateKey("")
		if err == nil {
			t.Error("ValidatePrivateKey() should fail for empty key")
		}
	})
}

func TestValidatePublicKey(t *testing.T) {
	t.Run("valid public key", func(t *testing.T) {
		_, publicKey, err := GenerateKeyPair()
		if err != nil {
			t.Fatalf("GenerateKeyPair() failed: %v", err)
		}

		if err := ValidatePublicKey(publicKey); err != nil {
			t.Errorf("ValidatePublicKey() failed for valid key: %v", err)
		}
	})

	t.Run("invalid base64", func(t *testing.T) {
		err := ValidatePublicKey("invalid-base64!")
		if err == nil {
			t.Error("ValidatePublicKey() should fail for invalid base64")
		}
		if !strings.Contains(err.Error(), "invalid base64 encoding") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("wrong length", func(t *testing.T) {
		shortKey := base64.StdEncoding.EncodeToString(make([]byte, 16)) // 16 bytes instead of 32
		err := ValidatePublicKey(shortKey)
		if err == nil {
			t.Error("ValidatePublicKey() should fail for wrong length")
		}
		if !strings.Contains(err.Error(), "must be exactly 32 bytes") {
			t.Errorf("unexpected error message: %v", err)
		}
	})
}

func TestPublicKeyFromPrivate(t *testing.T) {
	t.Run("derives correct public key", func(t *testing.T) {
		privateKey, expectedPublicKey, err := GenerateKeyPair()
		if err != nil {
			t.Fatalf("GenerateKeyPair() failed: %v", err)
		}

		derivedPublicKey, err := PublicKeyFromPrivate(privateKey)
		if err != nil {
			t.Fatalf("PublicKeyFromPrivate() failed: %v", err)
		}

		if derivedPublicKey != expectedPublicKey {
			t.Errorf("derived public key doesn't match expected:\nexpected: %s\nderived:  %s", 
				expectedPublicKey, derivedPublicKey)
		}
	})

	t.Run("invalid private key base64", func(t *testing.T) {
		_, err := PublicKeyFromPrivate("invalid-base64!")
		if err == nil {
			t.Error("PublicKeyFromPrivate() should fail for invalid base64")
		}
		if !strings.Contains(err.Error(), "invalid private key base64") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("wrong private key length", func(t *testing.T) {
		shortKey := base64.StdEncoding.EncodeToString(make([]byte, 16))
		_, err := PublicKeyFromPrivate(shortKey)
		if err == nil {
			t.Error("PublicKeyFromPrivate() should fail for wrong length")
		}
		if !strings.Contains(err.Error(), "must be exactly 32 bytes") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("consistent derivation", func(t *testing.T) {
		privateKey, _, err := GenerateKeyPair()
		if err != nil {
			t.Fatalf("GenerateKeyPair() failed: %v", err)
		}

		// Derive public key multiple times
		pub1, err1 := PublicKeyFromPrivate(privateKey)
		pub2, err2 := PublicKeyFromPrivate(privateKey)

		if err1 != nil || err2 != nil {
			t.Fatalf("PublicKeyFromPrivate() failed: %v, %v", err1, err2)
		}

		if pub1 != pub2 {
			t.Error("PublicKeyFromPrivate() should return consistent results")
		}
	})
}

func TestWireGuardCompatibility(t *testing.T) {
	t.Run("key format matches WireGuard expectations", func(t *testing.T) {
		privateKey, publicKey, err := GenerateKeyPair()
		if err != nil {
			t.Fatalf("GenerateKeyPair() failed: %v", err)
		}

		// WireGuard keys are 32 bytes base64 encoded
		// 32 bytes base64 encodes to 44 characters (no padding needed)
		// But we should be flexible about padding since base64.StdEncoding may add it

		// Decode and check actual byte length
		privBytes, err := base64.StdEncoding.DecodeString(privateKey)
		if err != nil {
			t.Errorf("private key is not valid base64: %v", err)
		}
		if len(privBytes) != 32 {
			t.Errorf("private key should be 32 bytes when decoded, got %d", len(privBytes))
		}

		pubBytes, err := base64.StdEncoding.DecodeString(publicKey)
		if err != nil {
			t.Errorf("public key is not valid base64: %v", err)
		}
		if len(pubBytes) != 32 {
			t.Errorf("public key should be 32 bytes when decoded, got %d", len(pubBytes))
		}

		// Keys should be valid base64 strings suitable for WireGuard config files
		// The exact character length can vary slightly due to base64 encoding specifics
		if len(privateKey) < 43 || len(privateKey) > 45 {
			t.Errorf("private key length should be 43-45 characters, got %d", len(privateKey))
		}
		if len(publicKey) < 43 || len(publicKey) > 45 {
			t.Errorf("public key length should be 43-45 characters, got %d", len(publicKey))
		}
	})
}

// Benchmark tests
func BenchmarkGenerateKeyPair(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _, err := GenerateKeyPair()
		if err != nil {
			b.Fatalf("GenerateKeyPair() failed: %v", err)
		}
	}
}

func BenchmarkPublicKeyFromPrivate(b *testing.B) {
	privateKey, _, err := GenerateKeyPair()
	if err != nil {
		b.Fatalf("GenerateKeyPair() failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := PublicKeyFromPrivate(privateKey)
		if err != nil {
			b.Fatalf("PublicKeyFromPrivate() failed: %v", err)
		}
	}
}