# ADR 0010: Consumer-owned OIDC transaction persistence

- Status: Accepted
- Date: 2026-09-13

## Context

The first real consumer candidate, Access, stores pending OIDC logins in
PostgreSQL so callbacks survive process restarts and can reach any replica.
SWF's bounded in-memory `TransactionStore` is suitable only for one process.
Pending transactions contain state, nonce, and a PKCE verifier and must be
browser-bound and consumed exactly once.

## Decision

SWF provides a small `TransactionCodec` that encrypts and authenticates a
pending transaction with a separate application-owned 32-byte key. The sealed
record is bound to the pinned issuer, client ID, and exact redirect URI. It is
valid only for the existing five-minute transaction lifetime. SWF does not own
the consumer's database schema, account linking, session, or refresh schedule.

The consumer stores only hashed state and browser binding alongside the sealed
record, and atomically deletes one matching unexpired row on callback before
opening it. `TransactionStore` remains an optional single-process helper.

## Consequences

Consumers using persistent storage must protect and back up the codec key,
share it across replicas, and handle key rotation without silently reusing
expired records. They may instead invalidate pending logins during a rotation.
The codec does not replace the database's one-use guarantee or the browser
cookie. Access needs a controlled schema and code migration before it can use
this path. No Access or Identity deployment is changed by this decision.
