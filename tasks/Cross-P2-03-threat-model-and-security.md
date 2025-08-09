# Cross-5 Threat Model and Security Baseline

**Priority: HIGH** (Security documentation)
**Phase: 2** (Before external release)

## Summary
Document security model and implement basic hardening measures.

## Deliverables
- `docs/SECURITY.md` covering:
  - Threat model and attack vectors
  - WireGuard protocol security guarantees
  - API key security and limitations
  - File permission requirements
  - Network security considerations
- Basic security hardening:
  - Secure file permissions (0600 for config files)
  - Input validation on API endpoints
  - Rate limiting for registration endpoint
  - Log sanitization (no key material logged)

## Acceptance Criteria
- [ ] Security documentation complete and linked from README
- [ ] All sensitive files use proper permissions
- [ ] API endpoints validate input properly

## Dependencies
P2-2.7 (server integration)

## Estimate
2 days






