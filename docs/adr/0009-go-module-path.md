# ADR 0009: Go module path follows the repository

- Status: Accepted
- Date: 2026-09-13

## Context

The initial module path was `github.com/stumpfworks/framework`, but the private
GitHub repository is `github.com/TheRealHZL/stumpfworks-framework`. A mismatch
would complicate imports from Identity and Access and require an unplanned vanity
import or redirect service.

## Decision

Use `github.com/TheRealHZL/stumpfworks-framework` as the canonical Go module
path. Package imports, examples, tests, and build metadata use that path.

## Consequences

Consumers need GitHub access to this private repository. They should configure
`GOPRIVATE=github.com/TheRealHZL/stumpfworks-framework` in their own build
environment and use their normal credential helper; credentials do not belong
in source, command examples, or CI logs. A future repository transfer would be
a deliberate module-path migration, not an implicit rename.
