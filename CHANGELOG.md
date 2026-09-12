# Changelog

All notable changes to StumpfWorks Framework are documented here. The project
follows Semantic Versioning; APIs may change during `0.x` development.

## [Unreleased]

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

### Security

- Raised the minimum toolchain from the initial Go 1.24 assumption to Go 1.26
  after vulnerability scanning found reachable standard-library issues.
- Updated pgx to 5.11.0 and `golang.org/x/text` to a fixed release line.
