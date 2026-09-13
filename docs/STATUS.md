# Implementation Status

## Candidate: 0.1.0-alpha.1

The initial vertical slice is implemented and builds on Go 1.26. It covers Core,
configuration, logging, HTTP lifecycle, health, PostgreSQL, migrations, strict
request decoding, and the audit foundation.

## Verified locally on 2026-09-12

- `gofmt`, `go vet`, `go test ./...`, and `go build ./...` pass.
- Ten shuffled repetitions of all unit tests pass.
- The minimal app returns 200 for liveness and readiness, with correlation and
  security headers.
- Invalid configuration exits non-zero with a structured error.
- `govulncheck` 1.8.0 reports no reachable vulnerabilities after the Go 1.26
  and dependency upgrade.

## Verified in GitHub CI

- The first private-repository workflow on `main` completed successfully.
- Linux race detection passed.
- PostgreSQL 17 integration passed, including migration application, audit
  insert/read-back, idempotency, concurrent migration runners, and transaction
  rollback.
- The Linux build and current pinned `govulncheck` scan passed.

The local Docker daemon remains unavailable, so PostgreSQL integration is
verified in CI rather than on the Windows workstation.

## OIDC client progress on 2026-09-13

The framework now has pinned-issuer Discovery and JWKS retrieval, an offline
RS256 ID-token verifier, confidential Authorization Code + PKCE S256 login,
one-use browser-bound transactions, and bounded Discovery/JWKS freshness.
The latest merges passed the full GitHub CI workflow on both `main` and
`develop`, including race tests, PostgreSQL integration, build, vet, and the
vulnerability scan.

This is a client library, not a complete application login. A consumer still
needs to wire the browser cookie and callback to its own session/account-link
store, schedule and monitor metadata refresh, and run a contract test against
the real Identity provider. Identity itself was not changed by this work.

## Before tagging alpha.1

- Confirm the Apache-2.0 license choice with the maintainer. The module path
  now matches the private GitHub repository (ADR 0009).
- Add a permanent private vulnerability-reporting address.
- Perform the documented quick start in a clean checkout.
