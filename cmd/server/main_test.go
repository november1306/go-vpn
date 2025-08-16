package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/november1306/go-vpn/internal/config"
	"github.com/november1306/go-vpn/internal/server/vpnserver"
	"github.com/november1306/go-vpn/internal/wireguard/keys"
)

func init() {
	// Initialize test configuration
	cfg = config.Load()
	
	// Initialize VPN server for testing (will fail on Windows but handlers still work)
	vpnServer = vpnserver.NewUserspaceVPNServer()
}

func TestHandleRegister(t *testing.T) {
	t.Run("successful registration", func(t *testing.T) {
		// Generate valid client key for test
		_, clientPubKey, err := keys.GenerateKeyPair()
		if err != nil {
			t.Fatalf("Failed to generate client key: %v", err)
		}

		reqBody := RegisterRequest{
			ClientPublicKey: clientPubKey,
		}
		
		jsonData, err := json.Marshal(reqBody)
		if err != nil {
			t.Fatalf("Failed to marshal request: %v", err)
		}

		req := httptest.NewRequest(http.MethodPost, "/api/register", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		
		rr := httptest.NewRecorder()
		handleRegister(rr, req)

		// Expect failure since VPN server won't be running in tests
		if rr.Code != http.StatusInternalServerError {
			t.Errorf("Expected status %d (VPN server not running), got %d", http.StatusInternalServerError, rr.Code)
		}

		var errResp ErrorResponse
		if err := json.NewDecoder(rr.Body).Decode(&errResp); err != nil {
			t.Fatalf("Failed to decode error response: %v", err)
		}

		if !strings.Contains(errResp.Error, "VPN server not running") {
			t.Errorf("Expected VPN server error, got %s", errResp.Error)
		}
	})

	t.Run("invalid method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/register", nil)
		rr := httptest.NewRecorder()
		
		handleRegister(rr, req)

		if rr.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, rr.Code)
		}

		// Check response is JSON
		contentType := rr.Header().Get("Content-Type")
		if !strings.Contains(contentType, "application/json") {
			t.Errorf("Expected JSON content type, got %s", contentType)
		}

		var errResp ErrorResponse
		if err := json.NewDecoder(rr.Body).Decode(&errResp); err != nil {
			t.Fatalf("Failed to decode error response: %v", err)
		}

		if errResp.Error == "" {
			t.Error("Expected error message in response")
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/register", strings.NewReader("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		
		rr := httptest.NewRecorder()
		handleRegister(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rr.Code)
		}

		var errResp ErrorResponse
		if err := json.NewDecoder(rr.Body).Decode(&errResp); err != nil {
			t.Fatalf("Failed to decode error response: %v", err)
		}

		if !strings.Contains(errResp.Error, "Invalid JSON") {
			t.Errorf("Expected 'Invalid JSON' error, got %s", errResp.Error)
		}
	})

	t.Run("missing client public key", func(t *testing.T) {
		reqBody := RegisterRequest{
			ClientPublicKey: "",
		}
		
		jsonData, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/register", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		
		rr := httptest.NewRecorder()
		handleRegister(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rr.Code)
		}

		var errResp ErrorResponse
		if err := json.NewDecoder(rr.Body).Decode(&errResp); err != nil {
			t.Fatalf("Failed to decode error response: %v", err)
		}

		if !strings.Contains(errResp.Error, "clientPublicKey is required") {
			t.Errorf("Expected 'clientPublicKey is required' error, got %s", errResp.Error)
		}
	})

	t.Run("invalid client public key format", func(t *testing.T) {
		reqBody := RegisterRequest{
			ClientPublicKey: "invalid-key-format",
		}
		
		jsonData, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/register", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		
		rr := httptest.NewRecorder()
		handleRegister(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rr.Code)
		}

		var errResp ErrorResponse
		if err := json.NewDecoder(rr.Body).Decode(&errResp); err != nil {
			t.Fatalf("Failed to decode error response: %v", err)
		}

		if !strings.Contains(errResp.Error, "Invalid client public key format") {
			t.Errorf("Expected key format error, got %s", errResp.Error)
		}
	})
}

func TestWriteErrorJSON(t *testing.T) {
	rr := httptest.NewRecorder()
	writeErrorJSON(rr, http.StatusBadRequest, "test error")

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}

	contentType := rr.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("Expected JSON content type, got %s", contentType)
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(rr.Body).Decode(&errResp); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}

	if errResp.Error != "test error" {
		t.Errorf("Expected 'test error', got %s", errResp.Error)
	}

	if errResp.Timestamp == "" {
		t.Error("Expected timestamp in error response")
	}
}