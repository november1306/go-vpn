#!/bin/bash
set -e

# Container integration test script
# Usage: ./scripts/test-container.sh [image-tag]

IMAGE_TAG="${1:-go-vpn:test}"
CONTAINER_NAME="go-vpn-integration-test"
API_PORT="8443"
VPN_PORT="51820"

echo "🐳 Starting container integration tests for ${IMAGE_TAG}"

# Cleanup function
cleanup() {
    echo "🧹 Cleaning up test container..."
    docker stop ${CONTAINER_NAME} 2>/dev/null || true
    docker rm ${CONTAINER_NAME} 2>/dev/null || true
}

# Set trap for cleanup on exit
trap cleanup EXIT

# Start container
echo "📦 Starting container..."
docker run -d --name ${CONTAINER_NAME} \
    -p ${API_PORT}:${API_PORT} \
    -p ${VPN_PORT}:${VPN_PORT}/udp \
    --cap-add NET_ADMIN \
    --cap-add SYS_MODULE \
    ${IMAGE_TAG}

# Wait for container to start
echo "⏳ Waiting for container to start..."
sleep 3

# Test 1: Container is running
echo "✅ Test 1: Checking if container is running..."
if docker ps | grep -q ${CONTAINER_NAME}; then
    echo "   ✓ Container is running"
else
    echo "   ❌ Container failed to start"
    docker logs ${CONTAINER_NAME}
    exit 1
fi

# Test 2: Container logs show expected output
echo "✅ Test 2: Checking container logs..."
LOGS=$(docker logs ${CONTAINER_NAME} 2>&1)
if echo "$LOGS" | grep -q "go-vpn server" && echo "$LOGS" | grep -q "Server starting"; then
    echo "   ✓ Server started successfully"
    echo "   📝 Logs: $LOGS"
else
    echo "   ❌ Unexpected log output"
    echo "   📝 Logs: $LOGS"
    exit 1
fi

# Test 3: Container health (basic process check)
echo "✅ Test 3: Checking container process..."
if docker exec ${CONTAINER_NAME} ps aux | grep -q "./server"; then
    echo "   ✓ Server process is running"
else
    echo "   ❌ Server process not found"
    docker exec ${CONTAINER_NAME} ps aux
    exit 1
fi

# Test 4: Network ports are bound (when server implements listening)
echo "✅ Test 4: Checking network configuration..."
if docker port ${CONTAINER_NAME} ${API_PORT} | grep -q "0.0.0.0:${API_PORT}"; then
    echo "   ✓ API port ${API_PORT} is exposed"
else
    echo "   ⚠️  API port ${API_PORT} not bound (expected until server implements HTTP listener)"
fi

if docker port ${CONTAINER_NAME} ${VPN_PORT} | grep -q "0.0.0.0:${VPN_PORT}"; then
    echo "   ✓ VPN port ${VPN_PORT} is exposed"
else
    echo "   ⚠️  VPN port ${VPN_PORT} not bound (expected until server implements WireGuard listener)"
fi

# Test 5: Container file permissions
echo "✅ Test 5: Checking file permissions and user..."
CONTAINER_USER=$(docker exec ${CONTAINER_NAME} whoami)
if [ "$CONTAINER_USER" = "vpn" ]; then
    echo "   ✓ Container running as non-root user: $CONTAINER_USER"
else
    echo "   ❌ Container not running as expected user (got: $CONTAINER_USER)"
    exit 1
fi

# Test 6: Required directories exist
echo "✅ Test 6: Checking required directories..."
if docker exec ${CONTAINER_NAME} test -d /etc/vpn; then
    echo "   ✓ /etc/vpn directory exists"
else
    echo "   ❌ /etc/vpn directory missing"
    exit 1
fi

if docker exec ${CONTAINER_NAME} test -d /var/lib/vpn; then
    echo "   ✓ /var/lib/vpn directory exists"
else
    echo "   ❌ /var/lib/vpn directory missing"
    exit 1
fi

# Test 7: Binary exists and is executable
echo "✅ Test 7: Checking binary..."
if docker exec ${CONTAINER_NAME} test -x ./server; then
    echo "   ✓ Server binary is executable"
else
    echo "   ❌ Server binary not found or not executable"
    exit 1
fi

# Future tests (uncomment when features are implemented):
#
# # Test 8: Health endpoint
# echo "✅ Test 8: Testing health endpoint..."
# if curl -f http://localhost:${API_PORT}/health; then
#     echo "   ✓ Health endpoint responding"
# else
#     echo "   ❌ Health endpoint not responding"
#     exit 1
# fi
#
# # Test 9: API endpoint basic auth
# echo "✅ Test 9: Testing API authentication..."
# if curl -f -H "Authorization: Bearer test" http://localhost:${API_PORT}/api/status; then
#     echo "   ✓ API endpoint responding with auth"
# else
#     echo "   ⚠️  API endpoint not yet implemented"
# fi

echo ""
echo "🎉 All container integration tests passed!"
echo "📊 Container stats:"
docker stats ${CONTAINER_NAME} --no-stream --format "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}"