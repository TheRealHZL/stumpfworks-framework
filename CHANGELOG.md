# Changelog

All notable changes to StumpfWorks Framework are documented here. The project
follows Semantic Versioning; APIs may change during `0.x` development.

## [Unreleased]

## [0.1.0-alpha.1] - 2026-09-14

### Added

- Application lifecycle with bounded HTTP shutdown and owned resources.
- Strict layered JSON/environment configuration.
- Structured logging with centralized secret-field redaction.
- Cryptographically random UUID generation and coded internal errors.
- Secure HTTP middleware, correlation IDs, privacy-safe request logs, health
  endpoints, RFC 9457 Problem Details, bounded JSON decoding, strict opt-in CORS,
  and concurrent-request overload protection.
- TLS-by-default PostgreSQL pooling and forward-only transactional migrations.
- Immutable audit-event envelope, PostgreSQL schema, and append-only store API.
- Unit, integration, race, build, vet, and vulnerability checks in CI.
- Pinned-issuer OIDC Discovery/JWKS, offline RS256 ID-token validation,
  Authorization Code with PKCE S256, browser-bound one-use transactions, and
  encrypted transaction persistence for consumer-owned storage.
- Bounded OIDC metadata cache, managed refresh loop, and freshness status.
- Opt-in Prometheus-compatible HTTP request metrics.

### Security

- Raised the minimum toolchain from the initial Go 1.24 assumption to Go 1.26
  after vulnerability scanning found reachable standard-library issues.
- Updated pgx to 5.11.0 and `golang.org/x/text` to a fixed release line.

### Fixed

- Serialized creation of the migration tracking table under the advisory lock,
  preventing concurrent first-start PostgreSQL catalog races.
