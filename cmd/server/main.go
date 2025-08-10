package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/november1306/go-vpn/internal/version"
	"github.com/november1306/go-vpn/internal/wireguard/keys"
)

type RegisterRequest struct {
	ClientPublicKey string `json:"clientPublicKey"`
}

type RegisterResponse struct {
	ServerPublicKey string `json:"serverPublicKey"`
	Message         string `json:"message"`
	Timestamp       string `json:"timestamp"`
}

var serverPrivateKey string
var serverPublicKey string

func handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.ClientPublicKey == "" {
		http.Error(w, "clientPublicKey is required", http.StatusBadRequest)
		return
	}

	// Validate client public key format
	if err := keys.ValidatePublicKey(req.ClientPublicKey); err != nil {
		http.Error(w, "Invalid client public key format: "+err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("Client registered with public key: %s", req.ClientPublicKey)

	// Return server public key
	response := RegisterResponse{
		ServerPublicKey: serverPublicKey,
		Message:         "Registration successful - demo mode",
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func main() {
	fmt.Printf("go-vpn server %s\n", version.Version)
	
	// Generate server key pair
	var err error
	serverPrivateKey, serverPublicKey, err = keys.GenerateKeyPair()
	if err != nil {
		log.Fatalf("Failed to generate server keys: %v", err)
	}
	
	fmt.Printf("Server public key: %s\n", serverPublicKey)
	
	// Set up HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/api/register", handleRegister)
	
	server := &http.Server{
		Addr:    ":8443",
		Handler: mux,
	}
	
	// Start server in goroutine
	go func() {
		fmt.Println("Server starting on :8443...")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()
	
	// Wait for interrupt signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	
	<-c
	fmt.Println("Server shutting down...")
	
	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}
}
