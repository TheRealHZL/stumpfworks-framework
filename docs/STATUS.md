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
Pending transactions can now be sealed for consumer-owned PostgreSQL storage;
an integration test verifies browser binding and atomic one-use retrieval.
`ACCESS-OIDC-INTEGRATION.md` records the next consumer steps. A provider-neutral
AI interface is noted only as a future candidate in the roadmap.
The latest merges passed the full GitHub CI workflow on both `main` and
`develop`, including race tests, PostgreSQL integration, build, vet, and the
vulnerability scan.

This remains a client library, not a complete application login. Access now
wires the browser cookie, callback, account link, and local session. Scheduled
and monitored metadata refresh is still open. Identity itself was not changed
by the framework implementation.

## Live OIDC contract check on 2026-09-14

- The opt-in `test/contract` check passed against the deployed Identity issuer
  using normal TLS verification. It
  covered Discovery, JWKS, and framework PKCE S256 login initialization; no
  client secret, user login, code exchange, or application session was used.
- This framework-side check made no production changes. It was only the first
  stage of the consumer acceptance test.

## Access consumer status on 2026-09-14

The Access project's validation record reports that its framework-backed OIDC
consumer is deployed and that a real browser login with an explicitly linked
Identity account succeeded. Its PostgreSQL integration tests cover successful
and negative flows; its deployment record describes a restricted backup and
rollback procedure. These are Access-project results, not tests independently
rerun by this framework repository. Live issuer-outage acceptance remains open.
Do not put deployment hostnames, IP addresses, or secrets in this public status
document.

The framework's local cache test covers overlapping old/new JWKS keys. Access
also passed isolated issuer-outage/recovery and overlapping-key integration tests
against a disposable PostgreSQL database. In a controlled live Identity rotation,
the operator confirmed a fresh Access browser login after the new key became the
only advertised key. The OIDC contract test, Identity and Access service checks,
and Identity token-issuance audit passed. The consumer's refresh loop runs every
ten minutes and logs each result. A dedicated alert and live issuer-outage
acceptance remain open; the isolated outage test is not a live outage test. The
production provider was not deliberately shut down.

Access has prepared an optional ntfy notifier for refresh failure, stale-key
escalation, and recovery, with local TLS tests. It is not deployed or connected
to an operator topic yet; no production notification has been verified.

## Before tagging alpha.1

- [x] The maintainer confirmed Apache-2.0 on 2026-09-14. The module path
  matches the GitHub repository (ADR 0009).
- [x] Add a permanent private vulnerability-reporting address:
  `hello@stumpfworks.de`.
- [x] Perform the documented quick start in a clean local checkout. On
  2026-09-14, `go test ./...` and the minimal-app build passed; both liveness
  and readiness returned HTTP 200. No PostgreSQL service was required.
