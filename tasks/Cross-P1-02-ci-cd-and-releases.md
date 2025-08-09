# Cross-4 CI/CD and Releases

**Priority: HIGH** (Development workflow)
**Phase: 1** (Early in development)

## Summary
Set up CI pipeline, linting, testing, and automated releases.

## Deliverables
- GitHub Actions workflows:
  - Build/test on Linux (required for all PRs)
  - Linting with `golangci-lint`
  - Race detector enabled
  - Go modules caching
- `goreleaser.yaml` for cross-platform binary releases
- Release workflow triggered by git tags

## Acceptance Criteria
- [ ] All PRs must pass CI checks
- [ ] `git tag v0.1.0 && git push --tags` produces release with binaries
- [ ] Release includes checksums and platform-specific binaries

## Dependencies
P1-1.1 (project setup)

## Estimate
2 days






