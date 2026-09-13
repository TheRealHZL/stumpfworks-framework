# OIDC ID-token verification

`auth/oidc` is the first narrow part of the 0.3 client. It verifies an ID token
from one operator-pinned HTTPS issuer using an immutable snapshot of that
issuer's public JWKS. It returns only `(issuer, subject)`; it neither creates an
application account nor grants roles or permissions.

```go
verifier, err := oidc.NewVerifier(expectedIssuer, clientID, trustedJWKS)
if err != nil { /* reject configuration or key refresh */ }
identity, err := verifier.VerifyIDToken(idToken, pendingLogin.Nonce)
if err != nil { /* reject login without exposing token contents */ }
// Look up identity.Issuer + identity.Subject in the application's explicit
// account-link table, then create a new application-owned session.
```

The application must fetch JWKS only from its configured issuer over verified
TLS, enforce a bounded response and cache lifetime, and replace the verifier
after a valid key refresh. It must never follow a `jku`, `x5u`, or embedded key
from a token. Unknown keys fail closed until a trusted refresh is complete.
This package currently accepts only RS256 public signing keys with unique
`kid`, `alg=RS256`, and `use=sig`; it does not implement Discovery or JWKS
fetching.

The caller owns the browser transaction: generate unpredictable one-use state,
nonce and PKCE S256 verifier; bind them to the browser; check the callback and
token exchange; consume the transaction once. `VerifyIDToken` checks a nonempty
expected nonce but cannot enforce one-use state by itself. Never pass an access
token here. Never log tokens, codes, state, nonce, client secrets, or raw callback
URLs. The current verifier requires an ID-token lifetime of at most ten minutes
and accepts at most one minute of future clock skew for `iat`/`nbf`.

Identity's current provider issues RS256 ID tokens with a five-minute lifetime
and a nonce, so its output is compatible by design. A live consumer contract
test is still required before production integration. Identity remains on Go
1.24 and is not changed by this module.
