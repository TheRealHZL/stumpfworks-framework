# ADR 0005: Self-hosted first

- Status: Accepted
- Date: 2026-09-12

## Context

StumpfWorks installations must keep operating without a mandatory vendor cloud.

## Decision

Every core capability must be deployable on infrastructure controlled by the
operator. External services may be optional adapters but cannot be required for
authentication, updates, diagnostics, or normal operation.

## Consequences

Configuration, data export, backup, restore, upgrades, and key ownership need
documented local workflows. Optional hosted integrations must fail without
breaking core application operation.
