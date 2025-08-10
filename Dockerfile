# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server ./cmd/server

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates iptables ip6tables

# Create non-root user
RUN addgroup -g 1001 vpn && \
    adduser -D -s /bin/sh -u 1001 -G vpn vpn

# Set working directory
WORKDIR /home/vpn

# Copy binary from builder stage
COPY --from=builder /app/server .

# Create directories for configuration and data
RUN mkdir -p /etc/vpn /var/lib/vpn && \
    chown -R vpn:vpn /etc/vpn /var/lib/vpn /home/vpn

# Expose ports
EXPOSE 8443/tcp 51820/udp

# Add health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget --quiet --tries=1 --spider http://localhost:8443/health || exit 1

# Switch to non-root user
USER vpn

# Set entrypoint
ENTRYPOINT ["./server"]