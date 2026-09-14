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
rerun by this framework repository. Key rotation and issuer-outage acceptance
remain open. Do not put deployment hostnames, IP addresses, or secrets in this
public status document.

The framework's local cache test now also covers overlapping old/new JWKS keys
before removal of the old key. This is not evidence of a live provider rotation.
The cache exposes freshness timestamps and state for consumer monitoring, but
Access now runs the optional managed refresh loop every ten minutes and logs
each result. The first refresh after its 2026-09-14 update succeeded; service
readiness and both login methods remained available. A dedicated alert and a
fresh interactive browser login after that binary update remain open. Local
outage/recovery tests are not evidence of a live provider outage test.

## Before tagging alpha.1

- Confirm the Apache-2.0 license choice with the maintainer. The module path
  now matches the GitHub repository (ADR 0009).
- Add a permanent private vulnerability-reporting address.
- Perform the documented quick start in a clean checkout.
