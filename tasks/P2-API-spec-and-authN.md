# P2 API Spec and Authentication

## Summary
Define versioned REST API, authentication scheme, and error model.

## Deliverables
- `api/openapi.yaml` (v3) documenting endpoints:
  - `POST /v1/register` (public): { email, client_public_key } -> { api_key, server_public_key, endpoint, assigned_ip }
  - `GET /v1/auth/ping` (auth required via `X-API-Key`)
  - `DELETE /v1/peers/{email}` (admin-only; optional for MVP)
- Auth header: `X-API-Key: <token>`; keys are random bytes base64url encoded
- Error envelope: `{ code, message, request_id }`, standard HTTP status codes
- TLS: HTTPS only (self-signed for dev, configurable cert/key for prod)

## Acceptance Criteria
- [ ] OpenAPI passes lint (spectral) and checked into repo

## Dependencies
P1-1.1

## Estimate
2 days


