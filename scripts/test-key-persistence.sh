#!/bin/bash
# Test script to verify server key persistence

set -e

echo "=== Testing Server Key Persistence ==="
echo ""

# Clean up any existing data directory
if [ -d "data" ]; then
    echo "Cleaning up existing data directory..."
    rm -rf data
fi

# Build the server
echo "Building server..."
go build -o build/server ./cmd/server

# Start server for the first time
echo ""
echo "=== Test 1: First startup (should generate new keys) ==="
timeout 3s ./build/server > server-log-1.txt 2>&1 || true

# Extract the public key from first run
PUBLIC_KEY_1=$(grep "Server public key:" server-log-1.txt | awk '{print $NF}')
echo "First startup public key: $PUBLIC_KEY_1"

# Check if server-key.json was created
if [ -f "data/server-key.json" ]; then
    echo "✅ Key file created: data/server-key.json"
    echo "Key file contents:"
    cat data/server-key.json
else
    echo "❌ Key file was not created"
    exit 1
fi

# Start server for the second time
echo ""
echo "=== Test 2: Second startup (should load existing keys) ==="
timeout 3s ./build/server > server-log-2.txt 2>&1 || true

# Extract the public key from second run
PUBLIC_KEY_2=$(grep "Server public key:" server-log-2.txt | awk '{print $NF}')
echo "Second startup public key: $PUBLIC_KEY_2"

# Compare public keys
if [ "$PUBLIC_KEY_1" == "$PUBLIC_KEY_2" ]; then
    echo "✅ Public keys match - persistence working!"
else
    echo "❌ Public keys don't match - persistence failed"
    echo "First:  $PUBLIC_KEY_1"
    echo "Second: $PUBLIC_KEY_2"
    exit 1
fi

# Check for "Loaded existing server key" message
if grep -q "Loaded existing server key" server-log-2.txt; then
    echo "✅ Server logged that it loaded existing keys"
else
    echo "❌ Server did not log loading existing keys"
    exit 1
fi

# Test 3: Corrupt the key file
echo ""
echo "=== Test 3: Corrupted key file (should fail with clear error) ==="
echo "invalid json" > data/server-key.json
timeout 3s ./build/server > server-log-3.txt 2>&1 || true

if grep -q "failed to load server keys" server-log-3.txt; then
    echo "✅ Server failed with clear error message for corrupted file"
else
    echo "❌ Server did not handle corrupted file properly"
    cat server-log-3.txt
    exit 1
fi

# Clean up
echo ""
echo "=== Cleanup ==="
rm -f server-log-*.txt
rm -rf data
rm -f build/server

echo ""
echo "✅ All tests passed!"
