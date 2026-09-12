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

## Before tagging alpha.1

- Confirm the module path and Apache-2.0 license choice with the maintainer.
- Add a permanent private vulnerability-reporting address.
- Perform the documented quick start in a clean checkout.
