# Cross-8 API Contract Tests (TypeScript + Playwright)

## Summary
Author API tests using Playwright's `APIRequestContext` to validate register and auth flows.

## Deliverables
- `tests/api/` (TypeScript) with Playwright config and specs:
  - `register.spec.ts`: happy path, invalid email, duplicate
  - `auth-ping.spec.ts`: valid/invalid API key
- GitHub Actions job to run API tests against ephemeral server

## Acceptance Criteria
- [ ] `npx playwright test` green locally and in CI

## Dependencies
P2-HTTP-server-and-middleware, P2-API-spec-and-authN

## Estimate
2 days


