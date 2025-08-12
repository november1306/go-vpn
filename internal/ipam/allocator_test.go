package ipam

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestNewAllocator(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid default config",
			config: Config{
				CIDR:    "10.0.0.0/24",
				Gateway: "10.0.0.1",
			},
			wantErr: false,
		},
		{
			name: "valid custom config",
			config: Config{
				CIDR:    "192.168.1.0/24",
				Gateway: "192.168.1.1",
			},
			wantErr: false,
		},
		{
			name: "invalid CIDR",
			config: Config{
				CIDR:    "invalid-cidr",
				Gateway: "10.0.0.1",
			},
			wantErr: true,
		},
		{
			name: "invalid gateway IP",
			config: Config{
				CIDR:    "10.0.0.0/24",
				Gateway: "invalid-ip",
			},
			wantErr: true,
		},
		{
			name: "gateway outside CIDR",
			config: Config{
				CIDR:    "10.0.0.0/24",
				Gateway: "192.168.1.1",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewAllocator(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewAllocator() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAllocateIP(t *testing.T) {
	allocator, err := NewAllocator(DefaultConfig())
	if err != nil {
		t.Fatalf("NewAllocator() failed: %v", err)
	}

	t.Run("first allocation", func(t *testing.T) {
		ip, err := allocator.AllocateIP(nil)
		if err != nil {
			t.Fatalf("AllocateIP() failed: %v", err)
		}
		if ip != "10.0.0.2/32" {
			t.Errorf("AllocateIP() = %v, want 10.0.0.2/32", ip)
		}
	})

	t.Run("sequential allocations", func(t *testing.T) {
		var users []UserIPInfo
		expectedIPs := []string{"10.0.0.2/32", "10.0.0.3/32", "10.0.0.4/32"}

		for i, expectedIP := range expectedIPs {
			ip, err := allocator.AllocateIP(users)
			if err != nil {
				t.Fatalf("AllocateIP() iteration %d failed: %v", i, err)
			}
			if ip != expectedIP {
				t.Errorf("AllocateIP() iteration %d = %v, want %v", i, ip, expectedIP)
			}
			users = append(users, SimpleUser{AssignedIP: ip})
		}
	})

	t.Run("skip already allocated IPs", func(t *testing.T) {
		users := []UserIPInfo{
			SimpleUser{AssignedIP: "10.0.0.2/32"},
			SimpleUser{AssignedIP: "10.0.0.4/32"}, // Skip .3
		}

		ip, err := allocator.AllocateIP(users)
		if err != nil {
			t.Fatalf("AllocateIP() failed: %v", err)
		}
		if ip != "10.0.0.3/32" {
			t.Errorf("AllocateIP() = %v, want 10.0.0.3/32", ip)
		}
	})

	t.Run("handle different IP formats", func(t *testing.T) {
		users := []UserIPInfo{
			SimpleUser{AssignedIP: "10.0.0.2"},    // Plain IP
			SimpleUser{AssignedIP: "10.0.0.3/32"}, // CIDR format
			SimpleUser{AssignedIP: "invalid-ip"},  // Invalid (should be skipped)
		}

		ip, err := allocator.AllocateIP(users)
		if err != nil {
			t.Fatalf("AllocateIP() failed: %v", err)
		}
		if ip != "10.0.0.4/32" {
			t.Errorf("AllocateIP() = %v, want 10.0.0.4/32", ip)
		}
	})
}

func TestAllocateIP_Exhaustion(t *testing.T) {
	allocator, err := NewAllocator(Config{
		CIDR:    "10.0.0.0/30", // Only 4 IPs: .0 (network), .1 (gateway), .2, .3
		Gateway: "10.0.0.1",
	})
	if err != nil {
		t.Fatalf("NewAllocator() failed: %v", err)
	}

	var users []UserIPInfo

	// Allocate all available IPs (.2 and .3)
	for i := 0; i < 2; i++ {
		ip, err := allocator.AllocateIP(users)
		if err != nil {
			t.Fatalf("AllocateIP() allocation %d failed: %v", i, err)
		}
		users = append(users, SimpleUser{AssignedIP: ip})
	}

	// Try to allocate one more - should fail
	_, err = allocator.AllocateIP(users)
	if err == nil {
		t.Error("AllocateIP() should have failed when pool exhausted")
	}
}

func TestIsIPAvailable(t *testing.T) {
	allocator, err := NewAllocator(DefaultConfig())
	if err != nil {
		t.Fatalf("NewAllocator() failed: %v", err)
	}

	users := []UserIPInfo{
		SimpleUser{AssignedIP: "10.0.0.2/32"},
		SimpleUser{AssignedIP: "10.0.0.4"},
	}

	tests := []struct {
		name     string
		targetIP string
		want     bool
	}{
		{"available IP", "10.0.0.3", true},
		{"allocated IP (CIDR)", "10.0.0.2", false},
		{"allocated IP (plain)", "10.0.0.4", false},
		{"gateway IP", "10.0.0.1", false},
		{"network address", "10.0.0.0", false},
		{"broadcast address", "10.0.0.255", false},
		{"out of range", "192.168.1.1", false},
		{"invalid IP", "invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := allocator.IsIPAvailable(tt.targetIP, users)
			if got != tt.want {
				t.Errorf("IsIPAvailable(%v) = %v, want %v", tt.targetIP, got, tt.want)
			}
		})
	}
}

func TestGetNetworkInfo(t *testing.T) {
	allocator, err := NewAllocator(DefaultConfig())
	if err != nil {
		t.Fatalf("NewAllocator() failed: %v", err)
	}

	info := allocator.GetNetworkInfo()

	if info.CIDR != "10.0.0.0/24" {
		t.Errorf("GetNetworkInfo().CIDR = %v, want 10.0.0.0/24", info.CIDR)
	}
	if info.Gateway != "10.0.0.1" {
		t.Errorf("GetNetworkInfo().Gateway = %v, want 10.0.0.1", info.Gateway)
	}
	if info.Range != "10.0.0.2-10.0.0.254" {
		t.Errorf("GetNetworkInfo().Range = %v, want 10.0.0.2-10.0.0.254", info.Range)
	}
}

func TestConcurrentAllocation(t *testing.T) {
	allocator, err := NewAllocator(DefaultConfig())
	if err != nil {
		t.Fatalf("NewAllocator() failed: %v", err)
	}

	// Test that concurrent calls to AllocateIP with the same input don't cause data races
	// This tests the internal thread safety of the allocator itself
	const numGoroutines = 10

	// Pre-allocate some users to simulate realistic scenario
	existingUsers := []UserIPInfo{
		SimpleUser{AssignedIP: "10.0.0.2/32"},
		SimpleUser{AssignedIP: "10.0.0.4/32"},
		SimpleUser{AssignedIP: "10.0.0.6/32"},
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	var results []string
	var errors []error

	// Simulate concurrent allocations with the same user list
	// This tests internal allocator thread safety
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			ip, err := allocator.AllocateIP(existingUsers)

			mu.Lock()
			if err != nil {
				errors = append(errors, err)
			} else {
				results = append(results, ip)
			}
			mu.Unlock()
		}()
	}

	wg.Wait()

	// With the same input, all should get the same next available IP (10.0.0.3/32)
	// This demonstrates that the allocator is deterministic and thread-safe
	expectedIP := "10.0.0.3/32"

	for i, result := range results {
		if result != expectedIP {
			t.Errorf("Concurrent allocation %d returned %v, expected %v", i, result, expectedIP)
		}
	}

	if len(errors) > 0 {
		t.Errorf("Unexpected errors during concurrent allocation: %v", errors)
	}

	t.Logf("All %d concurrent allocations returned consistent result: %s", len(results), expectedIP)
}

// TestConcurrentOperations tests various concurrent operations for race conditions
func TestConcurrentOperations(t *testing.T) {
	allocator, err := NewAllocator(DefaultConfig())
	if err != nil {
		t.Fatalf("NewAllocator() failed: %v", err)
	}

	const numGoroutines = 5
	var wg sync.WaitGroup

	users := []UserIPInfo{
		SimpleUser{AssignedIP: "10.0.0.2/32"},
		SimpleUser{AssignedIP: "10.0.0.3/32"},
	}

	// Test concurrent read operations don't interfere with each other
	for i := 0; i < numGoroutines; i++ {
		wg.Add(3) // 3 operations per goroutine

		go func() {
			defer wg.Done()
			allocator.AllocateIP(users)
		}()

		go func() {
			defer wg.Done()
			allocator.IsIPAvailable("10.0.0.5", users)
		}()

		go func() {
			defer wg.Done()
			allocator.GetNetworkInfo()
		}()
	}

	wg.Wait()
	t.Log("Concurrent operations completed without data races")
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.CIDR != "10.0.0.0/24" {
		t.Errorf("DefaultConfig().CIDR = %v, want 10.0.0.0/24", config.CIDR)
	}
	if config.Gateway != "10.0.0.1" {
		t.Errorf("DefaultConfig().Gateway = %v, want 10.0.0.1", config.Gateway)
	}
}

func TestSimpleUser(t *testing.T) {
	user := SimpleUser{AssignedIP: "10.0.0.5/32"}

	if user.GetAssignedIP() != "10.0.0.5/32" {
		t.Errorf("SimpleUser.GetAssignedIP() = %v, want 10.0.0.5/32", user.GetAssignedIP())
	}
}

// Benchmark allocation performance
func BenchmarkAllocateIP(b *testing.B) {
	allocator, err := NewAllocator(DefaultConfig())
	if err != nil {
		b.Fatalf("NewAllocator() failed: %v", err)
	}

	// Pre-allocate some IPs to simulate realistic scenario
	var users []UserIPInfo
	for i := 0; i < 100; i++ {
		ip, err := allocator.AllocateIP(users)
		if err != nil {
			break
		}
		users = append(users, SimpleUser{AssignedIP: ip})
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := allocator.AllocateIP(users)
		if err != nil {
			// Expected when pool is exhausted
			break
		}
	}
}

// TestOptimizedAllocation tests the performance optimizations
func TestOptimizedAllocation(t *testing.T) {
	config := DefaultConfig()
	config.EnableOptimizations = true

	allocator, err := NewAllocator(config)
	if err != nil {
		t.Fatalf("NewAllocator() failed: %v", err)
	}

	t.Run("optimized sequential allocations", func(t *testing.T) {
		var users []UserIPInfo
		expectedIPs := []string{"10.0.0.2/32", "10.0.0.3/32", "10.0.0.4/32", "10.0.0.5/32"}

		for i, expectedIP := range expectedIPs {
			ip, err := allocator.AllocateIP(users)
			if err != nil {
				t.Fatalf("AllocateIP() iteration %d failed: %v", i, err)
			}
			if ip != expectedIP {
				t.Errorf("AllocateIP() iteration %d = %v, want %v", i, ip, expectedIP)
			}
			users = append(users, SimpleUser{AssignedIP: ip})
		}
	})

	t.Run("optimized with gaps", func(t *testing.T) {
		users := []UserIPInfo{
			SimpleUser{AssignedIP: "10.0.0.2/32"},
			SimpleUser{AssignedIP: "10.0.0.4/32"}, // Skip .3
		}

		ip, err := allocator.AllocateIP(users)
		if err != nil {
			t.Fatalf("AllocateIP() failed: %v", err)
		}
		if ip != "10.0.0.3/32" {
			t.Errorf("AllocateIP() = %v, want 10.0.0.3/32", ip)
		}
	})
}

// TestAllocationStats tests statistics tracking
func TestAllocationStats(t *testing.T) {
	allocator, err := NewAllocator(DefaultConfig())
	if err != nil {
		t.Fatalf("NewAllocator() failed: %v", err)
	}

	// Initial stats should be zero
	stats := allocator.GetStats()
	if stats.TotalAllocations != 0 {
		t.Errorf("Initial TotalAllocations = %d, want 0", stats.TotalAllocations)
	}
	if stats.FailedAllocations != 0 {
		t.Errorf("Initial FailedAllocations = %d, want 0", stats.FailedAllocations)
	}

	// Successful allocation
	ip, err := allocator.AllocateIP(nil)
	if err != nil {
		t.Fatalf("AllocateIP() failed: %v", err)
	}
	if ip != "10.0.0.2/32" {
		t.Errorf("AllocateIP() = %v, want 10.0.0.2/32", ip)
	}

	// Check stats after successful allocation
	stats = allocator.GetStats()
	if stats.TotalAllocations != 1 {
		t.Errorf("TotalAllocations = %d, want 1", stats.TotalAllocations)
	}
	if stats.FailedAllocations != 0 {
		t.Errorf("FailedAllocations = %d, want 0", stats.FailedAllocations)
	}
	if stats.LastAllocationTime.IsZero() {
		t.Error("LastAllocationTime should not be zero")
	}

	// Test stats reset
	allocator.ResetStats()
	stats = allocator.GetStats()
	if stats.TotalAllocations != 0 {
		t.Errorf("After reset TotalAllocations = %d, want 0", stats.TotalAllocations)
	}
}

// TestOptimizedIsIPAvailable tests the optimized IP availability check
func TestOptimizedIsIPAvailable(t *testing.T) {
	config := DefaultConfig()
	config.EnableOptimizations = true

	allocator, err := NewAllocator(config)
	if err != nil {
		t.Fatalf("NewAllocator() failed: %v", err)
	}

	users := []UserIPInfo{
		SimpleUser{AssignedIP: "10.0.0.2/32"},
		SimpleUser{AssignedIP: "10.0.0.4"},
	}

	tests := []struct {
		name     string
		targetIP string
		want     bool
	}{
		{"available IP", "10.0.0.3", true},
		{"allocated IP (CIDR)", "10.0.0.2", false},
		{"allocated IP (plain)", "10.0.0.4", false},
		{"gateway IP", "10.0.0.1", false},
		{"network address", "10.0.0.0", false},
		{"broadcast address", "10.0.0.255", false},
		{"out of range", "192.168.1.1", false},
		{"invalid IP", "invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := allocator.IsIPAvailable(tt.targetIP, users)
			if got != tt.want {
				t.Errorf("IsIPAvailable(%v) = %v, want %v", tt.targetIP, got, tt.want)
			}
		})
	}
}

// TestBackwardCompatibility tests that the allocator works without optimizations
func TestBackwardCompatibility(t *testing.T) {
	config := DefaultConfig()
	config.EnableOptimizations = false

	allocator, err := NewAllocator(config)
	if err != nil {
		t.Fatalf("NewAllocator() failed: %v", err)
	}

	// Should work exactly like the original implementation
	ip, err := allocator.AllocateIP(nil)
	if err != nil {
		t.Fatalf("AllocateIP() failed: %v", err)
	}
	if ip != "10.0.0.2/32" {
		t.Errorf("AllocateIP() = %v, want 10.0.0.2/32", ip)
	}

	// IsIPAvailable should fall back to linear search
	users := []UserIPInfo{SimpleUser{AssignedIP: "10.0.0.2/32"}}
	if allocator.IsIPAvailable("10.0.0.2", users) {
		t.Error("IsIPAvailable should return false for allocated IP")
	}
	if !allocator.IsIPAvailable("10.0.0.3", users) {
		t.Error("IsIPAvailable should return true for available IP")
	}
}

// TestLargeScaleAllocation tests performance with many users
func TestLargeScaleAllocation(t *testing.T) {
	allocator, err := NewAllocator(DefaultConfig())
	if err != nil {
		t.Fatalf("NewAllocator() failed: %v", err)
	}

	// Simulate 100 existing users with scattered IPs
	var users []UserIPInfo
	for i := 0; i < 100; i++ {
		ip := fmt.Sprintf("10.0.0.%d/32", 2+i*2) // Every other IP
		users = append(users, SimpleUser{AssignedIP: ip})
	}

	// Allocate next IP - should be fast even with many users
	start := time.Now()
	ip, err := allocator.AllocateIP(users)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("AllocateIP() failed: %v", err)
	}
	if ip != "10.0.0.3/32" {
		t.Errorf("AllocateIP() = %v, want 10.0.0.3/32", ip)
	}

	// Performance check - should complete quickly
	if duration > 100*time.Millisecond {
		t.Errorf("Allocation took too long: %v", duration)
	}

	t.Logf("Large scale allocation completed in %v", duration)
}

// TestConcurrentOptimizedAllocation tests thread safety with optimizations
func TestConcurrentOptimizedAllocation(t *testing.T) {
	config := DefaultConfig()
	config.EnableOptimizations = true

	allocator, err := NewAllocator(config)
	if err != nil {
		t.Fatalf("NewAllocator() failed: %v", err)
	}

	const numGoroutines = 10

	// Pre-allocate some users to simulate realistic scenario
	existingUsers := []UserIPInfo{
		SimpleUser{AssignedIP: "10.0.0.2/32"},
		SimpleUser{AssignedIP: "10.0.0.4/32"},
		SimpleUser{AssignedIP: "10.0.0.6/32"},
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	var results []string
	var errors []error

	// Simulate concurrent allocations with the same user list
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			ip, err := allocator.AllocateIP(existingUsers)

			mu.Lock()
			if err != nil {
				errors = append(errors, err)
			} else {
				results = append(results, ip)
			}
			mu.Unlock()
		}()
	}

	wg.Wait()

	// With the same input, all should get the same next available IP (10.0.0.3/32)
	expectedIP := "10.0.0.3/32"

	for i, result := range results {
		if result != expectedIP {
			t.Errorf("Concurrent optimized allocation %d returned %v, expected %v", i, result, expectedIP)
		}
	}

	if len(errors) > 0 {
		t.Errorf("Unexpected errors during concurrent optimized allocation: %v", errors)
	}

	t.Logf("All %d concurrent optimized allocations returned consistent result: %s", len(results), expectedIP)
}
