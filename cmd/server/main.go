package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/november1306/go-vpn/internal/version"
)

func main() {
	fmt.Printf("go-vpn server %s\n", version.Version)
	
	// Keep server running until signal received
	fmt.Println("Server starting... Press Ctrl+C to stop.")
	
	// Wait for interrupt signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	
	<-c
	fmt.Println("Server shutting down...")
}
