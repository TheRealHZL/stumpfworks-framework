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

At the time of this decision the repository was private and consumers needed
GitHub access with `GOPRIVATE`. The repository is now public, so that private
module configuration is no longer required. The canonical module path remains
`github.com/TheRealHZL/stumpfworks-framework`; a future transfer that breaks
that path would require a deliberate module-path migration.
