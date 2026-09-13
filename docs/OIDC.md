# OIDC ID-token verification

`auth/oidc` is a narrow confidential OIDC relying-party client for one
operator-pinned HTTPS issuer. It retrieves Discovery and JWKS without redirects,
checks the advertised issuer and same-origin endpoints, performs Authorization
Code + PKCE S256, and verifies the ID token. It returns only `(issuer, subject)`;
it neither creates an application account nor grants roles or permissions.

```go
configuration, err := oidc.Discover(ctx, expectedIssuer, clientID, httpClient)
if err != nil { /* reject discovery */ }
client, err := oidc.NewLoginClient(configuration, clientID, clientSecret, exactRedirectURI, httpClient)
if err != nil { /* reject configuration */ }
authorizationURL, transaction, err := client.Begin()
if err != nil { /* reject login start */ }
// Bind transaction server-side to this browser before redirecting to authorizationURL.
identity, err := client.Complete(ctx, transaction, callbackQuery)
if err != nil { /* reject login without exposing token contents */ }
// Look up identity.Issuer + identity.Subject in the application's explicit
// account-link table, then create a new application-owned session.
```

`Discover` uses a five-second total deadline, bounds both JSON responses, and
rejects redirects or cross-origin metadata endpoints. The default HTTP client
verifies TLS. If a custom client is supplied for a private CA, its transport
must retain certificate and hostname verification. The application owns cache
age, refresh cadence, and atomic replacement of the returned configuration;
there is no automatic background refresh. Unknown keys fail closed until a
trusted refresh completes. Tokens cannot redirect key fetching via `jku`,
`x5u`, or embedded keys. Only RS256 public signing keys with unique `kid`,
`alg=RS256`, and `use=sig` are accepted.

`Begin` generates unpredictable state, nonce, and PKCE verifier. The
application must store the returned transaction server-side and bind its lookup
to the initiating browser; using state alone as the lookup key is not browser
binding. `Complete` consumes the transaction even on failure and validates the
state before exchanging the code. The application owns its local session,
account link, roles, logout, and recovery login. Never pass an access token to
the ID-token verifier. Never log tokens, codes, state, nonce, client secrets, or
raw callback URLs. The verifier requires an ID-token lifetime of at most ten
minutes and accepts at most one minute of future clock skew for `iat`/`nbf`.

Identity's current provider issues RS256 ID tokens with a five-minute lifetime
and a nonce, so its output is compatible by design. A live consumer contract
test and application-owned session integration are still required before
production use. Identity remains on Go 1.24 and is not changed by this module.
