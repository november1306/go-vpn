# Cross-6 Basic Testing and Manual Validation

## Summary
Create basic integration tests and manual testing checklist for release validation.

## Deliverables
- Integration tests in `test/` directory:
  - Server startup and WireGuard interface creation
  - Registration API endpoint functionality  
  - Basic CLI command execution
- Manual testing checklist `TESTING.md` with core functionality verification

## Acceptance Criteria
- [ ] Integration tests run with `go test ./test/...`
- [ ] Manual checklist covers server startup → registration → connect flow

**Priority: MEDIUM** (Quality assurance)
**Phase: 3** (After core functionality complete)

## Dependencies
P2-2.7 (server integration), P3-3.7 (CLI integration)

## Estimate
3 days






