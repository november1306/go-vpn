package vpnserver

import (
	"context"
	"fmt"
	"time"

	"github.com/november1306/go-vpn/internal/wireguard/keys"
)

// Example demonstrates basic VPN server usage
func ExampleVPNServer() {
	// Create a userspace VPN server (suitable for MVP, up to ~500 users)
	server := NewUserspaceVPNServer()
	
	// Generate server private key
	serverPrivKey, _, _ := keys.GenerateKeyPair()
	
	// Configure the server
	config := ServerConfig{
		InterfaceName: "wg0",
		PrivateKey:    serverPrivKey,
		ListenPort:    51820,
		ServerIP:      "10.0.0.1/24", // Server acts as gateway at 10.0.0.1
	}
	
	ctx := context.Background()
	
	// Start the VPN server
	if err := server.Start(ctx, config); err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
		return
	}
	defer server.Stop(ctx)
	
	// Get server info for clients
	serverInfo, _ := server.GetServerInfo()
	fmt.Printf("Server public key: %s\n", serverInfo.PublicKey)
	fmt.Printf("Clients should connect to: %s\n", serverInfo.Endpoint)
	
	// Simulate client registration - generate client key
	_, clientPubKey, _ := keys.GenerateKeyPair()
	
	// Add client to VPN (this would happen when client calls /api/register)
	if err := server.AddClient(clientPubKey, "10.0.0.2"); err != nil {
		fmt.Printf("Failed to add client: %v\n", err)
		return
	}
	
	// Check connected clients
	clients, _ := server.GetConnectedClients()
	fmt.Printf("Connected clients: %d\n", len(clients))
	
	// Remove client (when client disconnects)
	server.RemoveClient(clientPubKey)
	
	fmt.Println("Example completed successfully")
}

// Example for scaling to high-performance backend
func ExampleVPNServer_scalingPath() {
	// Current approach: Userspace backend
	userspaceServer := NewUserspaceVPNServer()
	
	// Future scaling approach: Abstract backend allows swapping implementations
	// without changing server logic
	
	// var backend WireGuardBackend
	// if needHighPerformance {
	//     backend = NewKernelBackend()      // 10x better performance on Linux
	// } else {
	//     backend = NewUserspaceBackend()   // Cross-platform compatibility
	// }
	// 
	// server := NewVPNServer(backend)
	
	fmt.Printf("Current server type: userspace (good for MVP)\n")
	fmt.Printf("Server ready: %v\n", userspaceServer != nil)
	
	// The VPNServer API remains exactly the same regardless of backend
	// This means scaling from hundreds to thousands of users requires
	// zero changes to application logic - just swap the backend
}

// Example of VPN server integration with HTTP API
func ExampleVPNServer_httpIntegration() {
	server := NewUserspaceVPNServer()
	
	// This would be integrated into cmd/server/main.go
	serverPrivKey, _, _ := keys.GenerateKeyPair()
	config := ServerConfig{
		InterfaceName: "wg0",
		PrivateKey:    serverPrivKey,
		ListenPort:    51820,
		ServerIP:      "10.0.0.1/24",
	}
	
	ctx := context.Background()
	server.Start(ctx, config)
	defer server.Stop(ctx)
	
	// HTTP handler for client registration would look like:
	// func handleRegister(w http.ResponseWriter, r *http.Request) {
	//     var req RegisterRequest
	//     json.NewDecoder(r.Body).Decode(&req)
	//     
	//     // Allocate IP for client (from P2-IP-allocator)
	//     clientIP := ipAllocator.AllocateIP()
	//     
	//     // Add client to VPN server
	//     server.AddClient(req.ClientPublicKey, clientIP)
	//     
	//     // Return server info for client connection
	//     serverInfo, _ := server.GetServerInfo()
	//     json.NewEncoder(w).Encode(RegisterResponse{
	//         ServerPublicKey: serverInfo.PublicKey,
	//         ServerEndpoint:  serverInfo.Endpoint,
	//         ClientIP:        clientIP,
	//     })
	// }
	
	fmt.Println("VPN server ready for HTTP integration")
}