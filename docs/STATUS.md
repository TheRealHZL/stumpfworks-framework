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

## Verification delegated to CI

- Race detection requires CGO, which is unavailable in the current Windows
  sandbox; Linux CI runs `go test -race ./...`.
- The PostgreSQL 17 integration test is implemented, but the local Docker daemon
  is not running. CI provisions PostgreSQL and verifies migrations, audit insert,
  read-back, and migration idempotency.

## Before tagging alpha.1

- Confirm the module path and Apache-2.0 license choice with the maintainer.
- Run the GitHub workflow in the eventual remote repository.
- Add a permanent private vulnerability-reporting address.
- Perform the documented quick start in a clean checkout.
