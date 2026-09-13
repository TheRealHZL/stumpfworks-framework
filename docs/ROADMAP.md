# Roadmap

## 0.1 — Foundation

- [x] Typed environment configuration and validation
- [x] Structured JSON logging
- [x] Build metadata
- [x] HTTP lifecycle and graceful shutdown
- [x] Liveness and readiness endpoints
- [x] PostgreSQL pool and readiness check
- [x] Explicit PostgreSQL transaction helper
- [x] Forward-only transactional migration lifecycle
- [x] Audit event model and PostgreSQL adapter
- [x] Problem Details response model
- [x] Bounded strict JSON input validation

Later milestones remain defined in `ARCHITECTURE.md` and will be broken down as
real consumers validate the design.

## 0.2 — Conventions and hardening

- [x] Central secret-field classification
- [x] Privacy-safe request logging and security headers
- [x] Opt-in strict CORS policy
- [x] Concurrent-request overload protection
- [x] Standard coded errors and RFC 9457 mapping
- [x] Opt-in Prometheus-compatible HTTP metrics
- [x] Initial Core and HTTP threat model
- [x] Initial Auth/OIDC client threat model and 0.3 contract boundary
- [ ] First Identity consumer integration

## 0.3 — OIDC client

- [x] Offline, pinned-issuer RS256 ID-token verifier with bounded JWKS and claims
- [x] Pinned-issuer Discovery and JWKS fetch with bounded responses, same-origin endpoints, no redirects, and TLS verification
- [x] One-use browser transaction, PKCE S256, confidential code exchange, and ID-token handoff
- [ ] Consumer-owned browser binding, local session integration, and JWKS refresh policy
- [ ] Contract test against the real Identity provider and first consumer
