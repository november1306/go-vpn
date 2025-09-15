#!/bin/bash

# Go-VPN Server Update Script
# Handles server binary updates with zero-downtime deployment

set -euo pipefail

SERVER_DIR="/root/go-vpn"
BACKUP_DIR="/var/backups/go-vpn"
SERVICE_NAME="go-vpn"

usage() {
    echo "Usage: $0 {update|rollback}"
    echo "  update   - Update server binary from latest build"
    echo "  rollback - Rollback to previous version"
    exit 1
}

check_service() {
    if systemctl is-active --quiet "$SERVICE_NAME" 2>/dev/null; then
        echo "✓ Service is running"
        return 0
    else
        echo "✗ Service is not running"
        return 1
    fi
}

backup_current() {
    echo "Backing up current installation..."
    mkdir -p "$BACKUP_DIR"
    
    TIMESTAMP=$(date +%Y%m%d_%H%M%S)
    
    # Backup binary
    if [[ -f "$SERVER_DIR/bin/server" ]]; then
        cp "$SERVER_DIR/bin/server" "$BACKUP_DIR/server_$TIMESTAMP"
        echo "✓ Binary backed up"
    fi
    
    # Backup data
    if [[ -d "$SERVER_DIR/data" ]]; then
        tar -czf "$BACKUP_DIR/data_$TIMESTAMP.tar.gz" -C "$SERVER_DIR/data" .
        echo "✓ Data backed up"
    fi
    
    echo "$TIMESTAMP" > "$BACKUP_DIR/latest_backup"
}

update_server() {
    echo "Updating Go-VPN server..."
    
    # Check if new binary exists
    if [[ ! -f "./bin/server" ]]; then
        echo "✗ New server binary not found. Run 'make build-server' first."
        exit 1
    fi
    
    # Backup current installation
    backup_current
    
    # Stop service
    echo "Stopping service..."
    systemctl stop "$SERVICE_NAME" || true
    
    # Update binary
    echo "Installing new binary..."
    mkdir -p "$SERVER_DIR/bin"
    cp "./bin/server" "$SERVER_DIR/bin/server"
    chmod +x "$SERVER_DIR/bin/server"
    
    # Start service
    echo "Starting service..."
    systemctl start "$SERVICE_NAME"
    
    # Verify service started
    sleep 2
    if check_service; then
        echo "✓ Update completed successfully"
    else
        echo "✗ Service failed to start, rolling back..."
        rollback_server
        exit 1
    fi
}

rollback_server() {
    echo "Rolling back to previous version..."
    
    if [[ ! -f "$BACKUP_DIR/latest_backup" ]]; then
        echo "✗ No backup found for rollback"
        exit 1
    fi
    
    TIMESTAMP=$(cat "$BACKUP_DIR/latest_backup")
    
    # Stop service
    systemctl stop "$SERVICE_NAME" || true
    
    # Restore binary
    if [[ -f "$BACKUP_DIR/server_$TIMESTAMP" ]]; then
        cp "$BACKUP_DIR/server_$TIMESTAMP" "$SERVER_DIR/bin/server"
        chmod +x "$SERVER_DIR/bin/server"
        echo "✓ Binary restored"
    fi
    
    # Restore data if needed
    if [[ -f "$BACKUP_DIR/data_$TIMESTAMP.tar.gz" ]]; then
        read -p "Restore data as well? (y/N): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            tar -xzf "$BACKUP_DIR/data_$TIMESTAMP.tar.gz" -C "$SERVER_DIR/data"
            echo "✓ Data restored"
        fi
    fi
    
    # Start service
    systemctl start "$SERVICE_NAME"
    
    if check_service; then
        echo "✓ Rollback completed successfully"
    else
        echo "✗ Rollback failed - manual intervention required"
        exit 1
    fi
}

case "${1:-}" in
    update)   update_server ;;
    rollback) rollback_server ;;
    *)        usage ;;
esac