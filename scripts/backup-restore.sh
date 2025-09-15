#!/bin/bash

# Go-VPN Backup and Restore Script
# Handles peer data backup and restore operations

set -euo pipefail

BACKUP_DIR="/var/backups/go-vpn"
DATA_DIR="/root/go-vpn/data"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

usage() {
    echo "Usage: $0 {backup|restore|list}"
    echo "  backup  - Create backup of peer data"
    echo "  restore - Restore from latest backup"
    echo "  list    - List available backups"
    exit 1
}

backup() {
    echo "Creating backup..."
    mkdir -p "$BACKUP_DIR"
    
    if [[ -d "$DATA_DIR" ]]; then
        tar -czf "$BACKUP_DIR/peers_$TIMESTAMP.tar.gz" -C "$DATA_DIR" .
        echo "✓ Backup created: $BACKUP_DIR/peers_$TIMESTAMP.tar.gz"
        
        # Keep only last 10 backups
        ls -t "$BACKUP_DIR"/peers_*.tar.gz | tail -n +11 | xargs -r rm
        echo "✓ Old backups cleaned up"
    else
        echo "✗ Data directory not found: $DATA_DIR"
        exit 1
    fi
}

restore() {
    echo "Restoring from backup..."
    
    LATEST=$(ls -t "$BACKUP_DIR"/peers_*.tar.gz 2>/dev/null | head -1)
    if [[ -z "$LATEST" ]]; then
        echo "✗ No backups found in $BACKUP_DIR"
        exit 1
    fi
    
    echo "Using backup: $LATEST"
    read -p "This will overwrite current data. Continue? (y/N): " -n 1 -r
    echo
    
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        mkdir -p "$DATA_DIR"
        tar -xzf "$LATEST" -C "$DATA_DIR"
        echo "✓ Restored from: $LATEST"
    else
        echo "Restore cancelled"
    fi
}

list_backups() {
    echo "Available backups:"
    if ls "$BACKUP_DIR"/peers_*.tar.gz >/dev/null 2>&1; then
        ls -lh "$BACKUP_DIR"/peers_*.tar.gz
    else
        echo "No backups found"
    fi
}

case "${1:-}" in
    backup)   backup ;;
    restore)  restore ;;
    list)     list_backups ;;
    *)        usage ;;
esac