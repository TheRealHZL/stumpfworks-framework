# ADR 0004: Identity as an external authority

- Status: Accepted
- Date: 2026-09-12

## Context

Applications need consistent authentication and authorization information, but
their domain logic and deployment lifecycles must remain independent.

## Decision

StumpfWorks Identity remains the authoritative source for users, credentials,
badges, authentication, and central authorization information. SWF provides
protocols, validation middleware, and client SDKs; it does not absorb Identity's
user-management domain.

## Consequences

Applications can validate identities consistently without sharing databases or
becoming a single deployable monolith. Protocol compatibility, key rotation, and
failure behaviour must be contract-tested.
