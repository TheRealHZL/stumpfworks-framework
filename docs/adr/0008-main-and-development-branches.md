# ADR 0008: Main and development branches

- Status: Accepted
- Date: 2026-09-12

## Context

The project needs a clearly stable line and a shared integration line. A separate
`stable` branch in addition to `main` would duplicate their purpose and invite
divergence. Delivery channels such as stable, testing, and nightly are release
metadata, not long-lived source branches.

## Decision

Use `main` as the protected, releasable branch and `develop` as the protected
integration branch. Short-lived feature and fix branches start from `develop`
and return through pull requests. Release candidates are merged from `develop`
to `main`; releases are identified by signed semantic-version tags.

Urgent production fixes branch from `main`, return to `main`, and are then merged
back into `develop`.

## Consequences

There is no additional `stable` branch. CI must pass on pull requests and both
long-lived branches. Branch-protection rules will be configured after the first
private GitHub push, once required check names exist.
