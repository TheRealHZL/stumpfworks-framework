# Access OIDC consumer integration plan

Status: preparation only. Neither Access nor Identity is changed here.

## Existing consumer boundary

Access already has an OIDC login in `internal/auth/oidc.go`. It stores hashed
state and browser binding in PostgreSQL, atomically deletes an attempt on
callback, links only `(issuer, subject)` to an existing user, and owns its
session, rate limits, audit, and local fallback login. Preserve those behaviours.
Its Go 1.26 baseline matches SWF. Identity currently advertises Authorization
Code, PKCE S256, RS256, and `client_secret_basic`, which match `auth/oidc`.

## Proposed replacement boundary

1. At startup, configure one pinned issuer, exact callback URI, trusted Homelab
   CA, and `ConfigurationCache`. Refresh before accepting new OIDC logins,
   publish a new `LoginClient` after each successful refresh, and schedule
   monitored refreshes before the configured maximum age.
2. At login start, call `LoginClient.Begin` and `NewBrowserBinding`. Seal the
   transaction with `TransactionCodec` using a separate persistent random
   32-byte key. Store ciphertext and hashed state/browser binding in Access's
   PostgreSQL attempts table. Access keeps its own rate limiting and redirect.
3. At callback, read and clear the host-only browser cookie. Atomically delete
   the matching unexpired row using both digests. Call `codec.Open` and then
   `transaction.Complete`. Pass only the verified `(issuer, subject)` to Access's
   existing account-link and session transaction. Never grant roles from claims.
4. Keep the old login implementation available for rollback until the new path
   passes tests. Do not use the process-local `TransactionStore` for Access.

The current attempts table stores plaintext nonce and PKCE verifier; migration
needs a sealed `bytea` record. Existing pending attempts can expire after five
minutes before cutover. The codec key must survive restarts and be shared by
replicas; rotation must overlap pending attempts or intentionally invalidate
them. It must not be logged or committed.

## Acceptance before rollout

- Browser test with Identity's cross-site 303-to-GET callback and the
  `SameSite=Lax` binding cookie. Access currently uses `SameSite=None`; do not
  change it without this test.
- Live Development contract test against Identity with verified Homelab TLS:
  discovery, JWKS, login, PKCE, nonce, callback, and explicit account linking.
- Negative tests for wrong browser, state, code replay, expired/tampered sealed
  record, wrong issuer/key, unlinked or inactive user, and issuer outage.
- Confirm existing Access sessions, roles, physical grants, local login,
  audit, and rate limits are unchanged. No production credentials in tests.
