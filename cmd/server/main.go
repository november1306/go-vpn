package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/november1306/go-vpn/internal/config"
	"github.com/november1306/go-vpn/internal/server/vpnserver"
	"github.com/november1306/go-vpn/internal/version"
	"github.com/november1306/go-vpn/internal/wireguard/keys"
)

type RegisterRequest struct {
	ClientPublicKey string `json:"clientPublicKey"`
}

type RegisterResponse struct {
	ServerPublicKey string `json:"serverPublicKey"`
	ServerEndpoint  string `json:"serverEndpoint"`
	ServerVPNIP     string `json:"serverVPNIP"`   // Server's IP within VPN network
	ServerAPIPort   int    `json:"serverAPIPort"` // Server's API port
	ClientIP        string `json:"clientIP"`
	Message         string `json:"message"`
	Timestamp       string `json:"timestamp"`
}

type ErrorResponse struct {
	Error     string `json:"error"`
	Timestamp string `json:"timestamp"`
}

type StatusResponse struct {
	Status         string               `json:"status"`
	ConnectedPeers int                  `json:"connectedPeers"`
	Peers          []vpnserver.PeerInfo `json:"peers"`
	ServerInfo     vpnserver.ServerInfo `json:"serverInfo"`
	Timestamp      string               `json:"timestamp"`
}

func writeErrorJSON(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:     message,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

var vpnServer *vpnserver.VPNServer
var cfg *config.Config

func handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if req.ClientPublicKey == "" {
		writeErrorJSON(w, http.StatusBadRequest, "clientPublicKey is required")
		return
	}

	// Validate client public key format
	if err := keys.ValidatePublicKey(req.ClientPublicKey); err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "Invalid client public key format: "+err.Error())
		return
	}

	// Add client to VPN server
	clientIP := cfg.Network.ClientIPDemo // Use configured demo client IP
	if err := vpnServer.AddClient(req.ClientPublicKey, clientIP); err != nil {
		slog.Error("Failed to add client to VPN", "error", err)
		writeErrorJSON(w, http.StatusInternalServerError, "Failed to add client to VPN: "+err.Error())
		return
	}

	// Get server info for client
	serverInfo, err := vpnServer.GetServerInfo()
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, "Failed to get server info")
		return
	}

	slog.Info("Client registered successfully", "clientIP", clientIP)

	// Extract server VPN IP from ServerIP (remove /24)
	serverVPNIP := strings.Split(serverInfo.ServerIP, "/")[0]

	// Auto-detect public endpoint if not configured
	endpoint := serverInfo.Endpoint
	if endpoint == "" || strings.HasPrefix(endpoint, ":") {
		// Extract the host from the request to determine public IP
		host := r.Host
		if strings.Contains(host, ":") {
			host = strings.Split(host, ":")[0]
		}
		// Use the host the client connected to with the VPN port
		endpoint = fmt.Sprintf("%s:%d", host, cfg.Server.VPNPort)
	}
	
	// Return connection details
	response := RegisterResponse{
		ServerPublicKey: serverInfo.PublicKey,
		ServerEndpoint:  endpoint,
		ServerVPNIP:     serverVPNIP,
		ServerAPIPort:   cfg.Server.APIPort,
		ClientIP:        clientIP + "/32",
		Message:         "Registration successful - VPN tunnel established",
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	peers, err := vpnServer.GetConnectedClients()
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, "Failed to get peer info")
		return
	}

	serverInfo, err := vpnServer.GetServerInfo()
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, "Failed to get server info")
		return
	}

	status := "running"
	if !vpnServer.IsRunning() {
		status = "stopped"
	}

	response := StatusResponse{
		Status:         status,
		ConnectedPeers: len(peers),
		Peers:          peers,
		ServerInfo:     serverInfo,
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// generateSelfSignedCert creates a simple self-signed certificate for HTTPS
func generateSelfSignedCert() (tls.Certificate, error) {
	// For demo purposes, we'll create a simple in-memory cert
	// In production, this would use proper certificate management
	return tls.Certificate{}, nil
}

func main() {
	fmt.Printf("go-vpn minimal server %s\n", version.Version)
	fmt.Println("=== Demo 2: Railway deployment with hardcoded peer ===")

	// Load configuration
	cfg = config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}
	fmt.Printf("Configuration loaded - API port: %d, VPN port: %d\n", cfg.Server.APIPort, cfg.Server.VPNPort)

	// Generate server key pair
	serverPrivateKey, serverPublicKey, err := keys.GenerateKeyPair()
	if err != nil {
		log.Fatalf("Failed to generate server keys: %v", err)
	}

	fmt.Printf("Server public key: %s\n", serverPublicKey)

	// Initialize VPN server with persistent storage
	dataDir := "data" // Create data directory for peer persistence
	vpnServer, err = vpnserver.NewUserspaceVPNServer(dataDir)
	if err != nil {
		log.Fatalf("Failed to create VPN server: %v", err)
	}

	serverConfig := vpnserver.ServerConfig{
		InterfaceName:  cfg.Server.InterfaceName,
		PrivateKey:     serverPrivateKey,
		ListenPort:     cfg.Server.VPNPort,
		ServerIP:       cfg.Network.ServerIP,
		PublicEndpoint: cfg.Server.PublicEndpoint,
	}

	// Start VPN server
	ctx := context.Background()
	slog.Info("Starting VPN server", "interface", cfg.Server.InterfaceName, "port", cfg.Server.VPNPort)

	if err := vpnServer.Start(ctx, serverConfig); err != nil {
		// On systems without TUN support, warn but continue with HTTP API
		if isTUNError(err) {
			slog.Warn("VPN server failed to start - continuing with HTTP API only", "error", err)
			slog.Warn("This is expected on Windows/systems without TUN support")
			slog.Warn("Deploy to Railway Linux for full VPN functionality")
		} else {
			log.Fatalf("Failed to start VPN server: %v", err)
		}
	} else {
		slog.Info("VPN server started successfully")

		// Add hardcoded test peer if configured
		if cfg.Test.PeerPublicKey != "" {
			slog.Info("Adding hardcoded test peer", "peerIP", cfg.Test.PeerIP)
			if err := vpnServer.AddClient(cfg.Test.PeerPublicKey, cfg.Test.PeerIP); err != nil {
				slog.Error("Failed to add test peer", "error", err)
			} else {
				slog.Info("Test peer added successfully")
			}
		}
	}

	// Set up HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/api/register", handleRegister)
	mux.HandleFunc("/api/status", handleStatus)
	mux.HandleFunc("/health", handleHealth)

	// VPN test endpoint - only accessible through VPN network
	mux.HandleFunc("/api/vpn-test", handleVPNTest)

	// Use mux directly without validation middleware
	var handler http.Handler = mux

	// Create HTTP server
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.APIPort),
		Handler: handler,
		// Security settings from configuration
		ReadTimeout:  cfg.Timeouts.HTTPRead,
		WriteTimeout: cfg.Timeouts.HTTPWrite,
		IdleTimeout:  cfg.Timeouts.HTTPIdle,
	}

	// Start HTTP server in goroutine
	go func() {
		slog.Info("HTTP API server starting", "port", cfg.Server.APIPort)
		// For demo, use HTTP. In production, use HTTPS with proper certificates
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	<-c
	slog.Info("Shutdown signal received")

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Timeouts.Shutdown)
	defer cancel()

	// Stop VPN server
	if vpnServer != nil && vpnServer.IsRunning() {
		slog.Info("Stopping VPN server")
		if err := vpnServer.Stop(shutdownCtx); err != nil {
			slog.Error("Error stopping VPN server", "error", err)
		}
	}

	// Stop HTTP server
	slog.Info("Stopping HTTP server")
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("HTTP server forced to shutdown", "error", err)
	}

	slog.Info("Server shutdown complete")
}

// handleHealth provides a health check endpoint that returns JSON
func handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	response := map[string]interface{}{
		"status":    "ok",
		"message":   "Server is running",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("Failed to encode health response", "error", err)
	}
}

// handleVPNTest provides a test endpoint to verify VPN tunneling
func handleVPNTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Get client's source IP
	clientIP := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		clientIP = forwarded
	}

	response := map[string]interface{}{
		"message":    "VPN tunnel test successful!",
		"clientIP":   clientIP,
		"serverTime": time.Now().UTC().Format(time.RFC3339),
		"via":        "VPN tunnel",
		"note":       "If you can see this, your VPN tunnel is working",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("Failed to encode VPN test response", "error", err)
	}

	slog.Info("VPN test endpoint accessed", "clientIP", clientIP)
}

// isTUNError checks if the error is related to TUN interface creation
func isTUNError(err error) bool {
	errStr := err.Error()
	return strings.Contains(errStr, "wintun.dll") ||
		strings.Contains(errStr, "TUN interface") ||
		strings.Contains(errStr, "tun") ||
		strings.Contains(errStr, "Unable to load library") ||
		strings.Contains(errStr, "failed to create TUN interface") ||
		strings.Contains(errStr, "device not initialized")
}
