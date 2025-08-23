package ipam

import (
	"fmt"
	"net"
	"sync"
	"time"
)

// UserIPInfo represents the minimal interface needed for IP allocation
// This allows the allocator to work with any type that provides IP information
type UserIPInfo interface {
	GetAssignedIP() string
}

// Allocator manages IP address allocation for VPN clients
// Uses optimized allocation with IP tracking for better performance
type Allocator struct {
	mu      sync.RWMutex
	cidr    *net.IPNet
	gateway net.IP
	startIP net.IP
	endIP   net.IP

	// Performance optimizations
	allocatedIPs  map[string]bool // Track allocated IPs for O(1) lookup
	lastAllocated net.IP          // Track last allocated IP for faster sequential allocation
	stats         *AllocationStats
}

// AllocationStats tracks allocation performance metrics
type AllocationStats struct {
	TotalAllocations      int64
	FailedAllocations     int64
	LastAllocationTime    time.Time
	AverageAllocationTime time.Duration
}

// Config defines the CIDR range for IP allocation
type Config struct {
	// CIDR is the network range to allocate from (e.g., "10.0.0.0/24")
	CIDR string
	// Gateway is the server IP (e.g., "10.0.0.1") - excluded from allocation
	Gateway string
	// EnableOptimizations enables performance optimizations (default: true)
	EnableOptimizations bool
}

// DefaultConfig returns the standard VPN configuration
func DefaultConfig() Config {
	return Config{
		CIDR:                "10.0.0.0/24",
		Gateway:             "10.0.0.1",
		EnableOptimizations: true,
	}
}

// ConfigFromNetwork creates an IPAM config from network configuration
func ConfigFromNetwork(cidr, gateway string) Config {
	return Config{
		CIDR:                cidr,
		Gateway:             gateway,
		EnableOptimizations: true,
	}
}

// NewAllocator creates a new IP allocator with the given configuration
func NewAllocator(config Config) (*Allocator, error) {
	// Parse CIDR
	_, cidr, err := net.ParseCIDR(config.CIDR)
	if err != nil {
		return nil, fmt.Errorf("invalid CIDR %s: %w", config.CIDR, err)
	}

	// Parse gateway
	gateway := net.ParseIP(config.Gateway)
	if gateway == nil {
		return nil, fmt.Errorf("invalid gateway IP %s", config.Gateway)
	}

	// Validate gateway is within CIDR
	if !cidr.Contains(gateway) {
		return nil, fmt.Errorf("gateway %s not in CIDR %s", config.Gateway, config.CIDR)
	}

	// Calculate allocation range (exclude network, gateway, and broadcast)
	startIP := make(net.IP, len(cidr.IP))
	copy(startIP, cidr.IP)

	// Start from .2 (skip network .0 and gateway .1)
	startIP[len(startIP)-1] = 2

	// End at .254 (skip broadcast .255)
	endIP := make(net.IP, len(cidr.IP))
	copy(endIP, cidr.IP)
	endIP[len(endIP)-1] = 254

	allocator := &Allocator{
		cidr:    cidr,
		gateway: gateway,
		startIP: startIP,
		endIP:   endIP,
		stats:   &AllocationStats{},
	}

	// Initialize optimizations if enabled
	if config.EnableOptimizations {
		allocator.allocatedIPs = make(map[string]bool)
		allocator.lastAllocated = make(net.IP, len(startIP))
		copy(allocator.lastAllocated, startIP)
		// Mark gateway as allocated
		allocator.allocatedIPs[gateway.String()] = true
	}

	return allocator, nil
}

// AllocateIP finds the next available IP address for a new client
// Returns the IP in CIDR format (e.g., "10.0.0.5/32")
func (a *Allocator) AllocateIP(existingUsers []UserIPInfo) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	var allocatedIP string
	var err error

	if a.allocatedIPs != nil {
		// Use optimized allocation
		allocatedIP, err = a.allocateIPOptimized(existingUsers)
	} else {
		// Use original linear search for backward compatibility
		allocatedIP, err = a.allocateIPLinear(existingUsers)
	}

	// Update statistics
	if err == nil {
		a.stats.TotalAllocations++
		a.stats.LastAllocationTime = time.Now()
	} else {
		a.stats.FailedAllocations++
	}

	return allocatedIP, err
}

// allocateIPOptimized uses tracking for O(1) allocation performance
func (a *Allocator) allocateIPOptimized(existingUsers []UserIPInfo) (string, error) {
	// Update our tracking from existing users
	a.updateAllocatedIPs(existingUsers)

	// Start from the beginning of the range for deterministic behavior
	ip := make(net.IP, len(a.startIP))
	copy(ip, a.startIP)

	// Calculate max attempts based on actual IP range size
	maxAttempts := int(a.endIP[len(a.endIP)-1] - a.startIP[len(a.startIP)-1] + 1)
	for attempts := 0; attempts < maxAttempts; attempts++ {
		// Check if we've reached the end
		if !a.isIPInRange(ip) {
			break
		}

		// Check if IP is available
		if !a.allocatedIPs[ip.String()] {
			// Found free IP - update tracking and return
			a.allocatedIPs[ip.String()] = true
			copy(a.lastAllocated, ip)
			return fmt.Sprintf("%s/32", ip.String()), nil
		}

		// Increment to next IP
		incrementIP(ip)
	}

	return "", fmt.Errorf("no available IPs in range %s-%s", a.startIP, a.endIP)
}

// allocateIPLinear is the original linear search implementation
func (a *Allocator) allocateIPLinear(existingUsers []UserIPInfo) (string, error) {
	// Build set of allocated IPs for fast lookup
	allocated := make(map[string]bool)
	for _, user := range existingUsers {
		if assignedIP := user.GetAssignedIP(); assignedIP != "" {
			// Extract IP from CIDR format if needed
			ip, _, err := net.ParseCIDR(assignedIP)
			if err != nil {
				// Try parsing as plain IP
				ip = net.ParseIP(assignedIP)
				if ip == nil {
					continue // Skip invalid IPs
				}
			}
			allocated[ip.String()] = true
		}
	}

	// Also mark gateway as allocated
	allocated[a.gateway.String()] = true

	// Linear search for next free IP
	ip := make(net.IP, len(a.startIP))
	copy(ip, a.startIP)

	for {
		// Check if we've reached the end
		if !a.isIPInRange(ip) {
			break
		}

		// Skip if already allocated
		if !allocated[ip.String()] {
			// Found free IP - return in /32 CIDR format for client
			return fmt.Sprintf("%s/32", ip.String()), nil
		}

		// Increment to next IP
		incrementIP(ip)
	}

	return "", fmt.Errorf("no available IPs in range %s-%s", a.startIP, a.endIP)
}

// updateAllocatedIPs updates the internal tracking from existing users
func (a *Allocator) updateAllocatedIPs(existingUsers []UserIPInfo) {
	// Only recreate map if size changed significantly to avoid unnecessary allocations
	expectedSize := len(existingUsers) + 1 // +1 for gateway
	if len(a.allocatedIPs) == 0 || len(a.allocatedIPs) < expectedSize/2 || len(a.allocatedIPs) > expectedSize*2 {
		a.allocatedIPs = make(map[string]bool, expectedSize)
	} else {
		// Clear existing entries efficiently
		for k := range a.allocatedIPs {
			delete(a.allocatedIPs, k)
		}
	}

	// Always ensure gateway is marked as allocated
	a.allocatedIPs[a.gateway.String()] = true

	// Add existing users
	for _, user := range existingUsers {
		if assignedIP := user.GetAssignedIP(); assignedIP != "" {
			ip, _, err := net.ParseCIDR(assignedIP)
			if err != nil {
				ip = net.ParseIP(assignedIP)
			}
			if ip != nil {
				a.allocatedIPs[ip.String()] = true
			}
		}
	}
}

// IsIPAvailable checks if a specific IP is available for allocation
func (a *Allocator) IsIPAvailable(targetIP string, existingUsers []UserIPInfo) bool {
	// Parse target IP
	ip := net.ParseIP(targetIP)
	if ip == nil {
		return false
	}

	// Check if IP is in our allocation range
	if !a.isIPInRange(ip) {
		return false
	}

	// Check if IP is gateway
	if ip.Equal(a.gateway) {
		return false
	}

	// Use optimized lookup if available
	if a.allocatedIPs != nil {
		// Build a temporary map for this check to avoid race conditions
		allocated := make(map[string]bool)
		allocated[a.gateway.String()] = true

		for _, user := range existingUsers {
			if assignedIP := user.GetAssignedIP(); assignedIP != "" {
				userIP, _, err := net.ParseCIDR(assignedIP)
				if err != nil {
					userIP = net.ParseIP(assignedIP)
				}
				if userIP != nil {
					allocated[userIP.String()] = true
				}
			}
		}
		return !allocated[ip.String()]
	}

	// Fallback to linear search
	for _, user := range existingUsers {
		if assignedIP := user.GetAssignedIP(); assignedIP != "" {
			userIP, _, err := net.ParseCIDR(assignedIP)
			if err != nil {
				userIP = net.ParseIP(assignedIP)
			}
			if userIP != nil && userIP.Equal(ip) {
				return false
			}
		}
	}

	return true
}

// GetNetworkInfo returns information about the allocation network
func (a *Allocator) GetNetworkInfo() NetworkInfo {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return NetworkInfo{
		CIDR:    a.cidr.String(),
		Gateway: a.gateway.String(),
		Range:   fmt.Sprintf("%s-%s", a.startIP, a.endIP),
	}
}

// GetStats returns allocation statistics
func (a *Allocator) GetStats() AllocationStats {
	a.mu.RLock()
	defer a.mu.RUnlock()

	stats := *a.stats // Return a copy
	return stats
}

// ResetStats resets allocation statistics
func (a *Allocator) ResetStats() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.stats = &AllocationStats{}
}

// NetworkInfo provides network configuration details
type NetworkInfo struct {
	CIDR    string // Network CIDR (e.g., "10.0.0.0/24")
	Gateway string // Gateway IP (e.g., "10.0.0.1")
	Range   string // Allocation range (e.g., "10.0.0.2-10.0.0.254")
}

// isIPInRange checks if an IP is within the allocation range
func (a *Allocator) isIPInRange(ip net.IP) bool {
	// Quick bounds check on last octet for performance
	lastOctet := ip[len(ip)-1]
	if lastOctet < a.startIP[len(a.startIP)-1] || lastOctet > a.endIP[len(a.endIP)-1] {
		return false
	}

	return a.cidr.Contains(ip) &&
		!ip.Equal(a.startIP.Mask(a.cidr.Mask)) // Not network address
}

// incrementIP increments an IP address by 1
func incrementIP(ip net.IP) {
	for i := len(ip) - 1; i >= 0; i-- {
		ip[i]++
		if ip[i] != 0 {
			break
		}
	}
}

// SimpleUser is a minimal implementation of UserIPInfo for testing
type SimpleUser struct {
	AssignedIP string
}

// GetAssignedIP implements UserIPInfo interface
func (u SimpleUser) GetAssignedIP() string {
	return u.AssignedIP
}
