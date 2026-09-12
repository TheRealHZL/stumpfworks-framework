# ADR 0002: Modular monorepo for SWF

- Status: Accepted
- Date: 2026-09-12

## Decision

Start SWF as one repository and one Go module. Keep public packages cohesive and
place non-contractual implementation details under `internal/`. Split modules
only after a demonstrated release or ownership need.
