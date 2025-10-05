# Implementation Plan

- [x] 1. Create ServerKeyStore package with key persistence logic


  - Create `internal/server/keystore/keystore.go` with ServerKeyStore struct and methods
  - Implement NewServerKeyStore() to load existing keys or generate new ones
  - Implement GetKeys(), GetPrivateKey(), GetPublicKey() accessor methods
  - Implement private load() method to read keys from JSON file
  - Implement private save() method with atomic write operations
  - Implement private generateAndSave() method for first-time key generation
  - Set file permissions to 0600 on Unix systems for security
  - Add proper error handling with clear, actionable error messages
  - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5, 2.1, 2.2, 2.3, 2.4, 4.1, 4.2, 4.3, 5.1, 5.3, 5.4, 5.5_

- [x] 1.1 Write unit tests for ServerKeyStore


  - Test new key generation on first initialization
  - Test loading existing keys from file
  - Test handling of corrupted JSON file
  - Test handling of invalid private key format
  - Test automatic data directory creation
  - Test thread-safe concurrent access to keys
  - Test atomic write operations
  - Test file permissions on Unix systems
  - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5, 2.1, 2.2, 2.5_

- [x] 2. Integrate ServerKeyStore into server main.go


  - Import the new keystore package in cmd/server/main.go
  - Replace keys.GenerateKeyPair() call with keystore.NewServerKeyStore(dataDir)
  - Update serverPrivateKey and serverPublicKey to use keyStore.GetPrivateKey() and keyStore.GetPublicKey()
  - Add logging for key generation vs key loading scenarios
  - Ensure error handling fails server startup if keys cannot be loaded
  - _Requirements: 1.1, 1.2, 3.1, 3.2, 3.3, 4.1, 4.2, 4.3, 4.4, 5.1, 5.2_

- [x] 3. Update PRD to reflect persistent server keys completion


  - Move "Persistent Server Keys" from "⏳ In Progress" to "✅ Completed Features"
  - Update the feature description to reflect the implementation
  - _Requirements: 4.4_

- [x] 4. Manual integration testing



  - Test fresh server deployment generates and saves keys
  - Test server restart loads existing keys with same public key
  - Test client registration and connection continuity across server restart
  - Test handling of corrupted key file with clear error message
  - Test upgrade path from non-persistent version
  - Verify file permissions are set correctly (0600 on Unix)
  - Verify logging messages are clear and actionable
  - _Requirements: 1.1, 1.2, 1.3, 1.4, 3.1, 3.2, 3.3, 3.4, 3.5, 4.1, 4.2, 4.3, 4.4, 4.5, 5.1, 5.2, 5.3, 5.4, 5.5_
