# Requirements Document

## Introduction

The VPN server currently regenerates its WireGuard key pair on every restart, forcing all clients to re-register after server redeployment or restart. This creates a poor user experience and breaks the fundamental VPN workflow. This feature will implement persistent server key storage, ensuring the server maintains the same identity across restarts and allowing clients to register once and connect reliably without re-registration.

## Requirements

### Requirement 1: Server Key Persistence

**User Story:** As a VPN server administrator, I want the server to maintain the same WireGuard keys across restarts, so that existing client configurations remain valid without requiring re-registration.

#### Acceptance Criteria

1. WHEN the server starts for the first time THEN it SHALL generate a new WireGuard key pair and persist it to disk
2. WHEN the server starts and a persisted key exists THEN it SHALL load the existing key pair instead of generating a new one
3. WHEN the server key is persisted THEN it SHALL be stored in the data directory with secure file permissions (0600 on Unix systems)
4. WHEN the server key file is corrupted or invalid THEN the server SHALL log an error and fail to start with a clear error message
5. IF the server key file does not exist THEN the server SHALL generate a new key pair and save it automatically

### Requirement 2: Secure Key Storage

**User Story:** As a security-conscious administrator, I want server private keys stored securely with appropriate file permissions, so that unauthorized users cannot access sensitive cryptographic material.

#### Acceptance Criteria

1. WHEN the server saves the private key to disk THEN it SHALL set file permissions to 0600 (owner read/write only) on Unix systems
2. WHEN the server saves the private key THEN it SHALL use atomic write operations to prevent corruption during write failures
3. WHEN the key file is created THEN it SHALL be stored in JSON format with clear structure
4. WHEN storing keys THEN the server SHALL only persist the private key (public key can be derived)
5. IF the key file has insecure permissions THEN the server SHALL log a warning on startup

### Requirement 3: Client Connection Continuity

**User Story:** As a VPN client user, I want to register once and connect reliably across server restarts, so that I don't need to re-register every time the server is redeployed.

#### Acceptance Criteria

1. WHEN a client registers with the server THEN the client configuration SHALL remain valid across server restarts
2. WHEN the server restarts THEN existing clients SHALL be able to connect without re-registration
3. WHEN a client attempts to connect after server restart THEN the WireGuard tunnel SHALL establish successfully using the persisted server key
4. WHEN the server public key changes (new key generation) THEN existing client configurations SHALL become invalid and require re-registration
5. IF the server is redeployed with the same data directory THEN all existing client registrations SHALL remain valid

### Requirement 4: Operational Visibility

**User Story:** As a VPN server administrator, I want clear logging about server key operations, so that I can understand whether keys are being loaded or generated and troubleshoot issues.

#### Acceptance Criteria

1. WHEN the server generates a new key pair THEN it SHALL log "Generated new server key pair" with the public key
2. WHEN the server loads an existing key THEN it SHALL log "Loaded existing server key" with the public key
3. WHEN the server fails to load a key file THEN it SHALL log the specific error with actionable guidance
4. WHEN the server starts THEN it SHALL display the server public key in the startup output
5. IF key file permissions are insecure THEN the server SHALL log a security warning

### Requirement 5: Backward Compatibility

**User Story:** As a VPN server administrator upgrading from a previous version, I want the server to work seamlessly with existing deployments, so that the upgrade process is smooth without manual intervention.

#### Acceptance Criteria

1. WHEN the server starts and no key file exists THEN it SHALL automatically generate and save a new key (first-time setup)
2. WHEN upgrading from a version without persistent keys THEN the server SHALL generate and persist keys on first startup
3. WHEN the data directory exists but has no key file THEN the server SHALL create the key file without affecting existing peer data
4. WHEN the server is deployed to a new environment THEN it SHALL automatically initialize with new keys
5. IF the data directory does not exist THEN the server SHALL create it with appropriate permissions before saving keys
