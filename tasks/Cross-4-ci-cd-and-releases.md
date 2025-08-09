# Cross-4 CI/CD and Releases

## Summary
Set up CI matrix, vet/lint, tests, and release packaging.

## Deliverables
- GitHub Actions: build/test on linux/windows; race detector; caching
- Lint with `golangci-lint`
- `goreleaser.yaml` for binaries (server, vpn-cli) across OS/arch

## Acceptance Criteria
- [ ] Tag push produces draft release with checksums

## Dependencies
P1-1.1

## Estimate
2 days


