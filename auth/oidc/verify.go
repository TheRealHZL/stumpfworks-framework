// Package oidc verifies ID tokens from an operator-pinned OIDC issuer.
// It does not perform discovery, token exchange, account linking, or sessions.
package oidc

import (
	"crypto/rsa"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"time"

	jose "github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
)

const (
	maxJWKSBytes  = 64 << 10
	maxTokenBytes = 16 << 10
	maxKeys       = 16
	clockSkew     = time.Minute
	maxLifetime   = 10 * time.Minute
)

// ErrInvalidToken never includes untrusted token, key, or claim contents.
var ErrInvalidToken = errors.New("invalid OIDC ID token")

// Identity is a verified external identity, not a local user or permission set.
type Identity struct {
	Issuer  string
	Subject string
}

// Verifier holds an immutable snapshot of issuer signing keys. To rotate keys,
// construct a new Verifier from a freshly and securely fetched JWKS and swap it
// into the consuming application. Never fetch a JWKS URL from an untrusted JWT.
type Verifier struct {
	issuer   string
	clientID string
	keys     map[string]*rsa.PublicKey
	now      func() time.Time
}

// NewVerifier accepts only RS256 public signing keys from a bounded JWKS.
// The caller must fetch it over verified TLS from its pinned issuer and control
// cache freshness, size and retry policy. No network operation occurs here.
func NewVerifier(issuer, clientID string, jwks []byte) (*Verifier, error) {
	u, err := url.Parse(issuer)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || strings.HasSuffix(issuer, "/") {
		return nil, errors.New("issuer must be an exact HTTPS origin")
	}
	if clientID == "" || len(clientID) > 256 || strings.TrimSpace(clientID) != clientID {
		return nil, errors.New("client ID is required")
	}
	if len(jwks) == 0 || len(jwks) > maxJWKSBytes {
		return nil, errors.New("JWKS size is invalid")
	}
	var set jose.JSONWebKeySet
	if err := json.Unmarshal(jwks, &set); err != nil || len(set.Keys) == 0 || len(set.Keys) > maxKeys {
		return nil, errors.New("JWKS is invalid")
	}
	keys := make(map[string]*rsa.PublicKey, len(set.Keys))
	for _, key := range set.Keys {
		public, ok := key.Key.(*rsa.PublicKey)
		if !ok || !key.Valid() || !key.IsPublic() || public.N.BitLen() < 2048 || key.KeyID == "" || len(key.KeyID) > 128 || key.Algorithm != string(jose.RS256) || key.Use != "sig" || key.CertificatesURL != nil {
			return nil, errors.New("JWKS contains an unsupported key")
		}
		if _, duplicate := keys[key.KeyID]; duplicate {
			return nil, errors.New("JWKS contains duplicate key IDs")
		}
		keys[key.KeyID] = public
	}
	return &Verifier{issuer: issuer, clientID: clientID, keys: keys, now: time.Now}, nil
}

type idClaims struct {
	jwt.Claims
	Nonce           string `json:"nonce"`
	AuthorizedParty string `json:"azp"`
}

// VerifyIDToken validates a token for one pending login transaction. The caller
// must provide its own nonempty expected nonce and enforce one-time transaction
// use. Access tokens are not accepted by this API.
func (verifier *Verifier) VerifyIDToken(raw, expectedNonce string) (Identity, error) {
	if verifier == nil || expectedNonce == "" || len(expectedNonce) > 256 || raw == "" || len(raw) > maxTokenBytes {
		return Identity{}, ErrInvalidToken
	}
	parsed, err := jwt.ParseSigned(raw, []jose.SignatureAlgorithm{jose.RS256})
	if err != nil || len(parsed.Headers) != 1 {
		return Identity{}, ErrInvalidToken
	}
	header := parsed.Headers[0]
	if header.Algorithm != string(jose.RS256) || header.KeyID == "" || header.JSONWebKey != nil {
		return Identity{}, ErrInvalidToken
	}
	for name, value := range header.ExtraHeaders {
		if name != "typ" || value != "JWT" {
			return Identity{}, ErrInvalidToken
		}
	}
	key, ok := verifier.keys[header.KeyID]
	if !ok {
		return Identity{}, ErrInvalidToken
	}
	var claims idClaims
	if err := parsed.Claims(key, &claims); err != nil {
		return Identity{}, ErrInvalidToken
	}
	now := verifier.now()
	if claims.Issuer != verifier.issuer || claims.Subject == "" || len(claims.Subject) > 256 || !claims.Audience.Contains(verifier.clientID) || claims.Expiry == nil || claims.IssuedAt == nil || claims.Nonce != expectedNonce {
		return Identity{}, ErrInvalidToken
	}
	if (len(claims.Audience) != 1 && claims.AuthorizedParty != verifier.clientID) ||
		(claims.AuthorizedParty != "" && claims.AuthorizedParty != verifier.clientID) {
		return Identity{}, ErrInvalidToken
	}
	issuedAt, expiry := claims.IssuedAt.Time(), claims.Expiry.Time()
	if claims.IssuedAt.Time().Unix() <= 0 || !now.Before(expiry) || !expiry.After(issuedAt) ||
		expiry.Sub(issuedAt) > maxLifetime || issuedAt.After(now.Add(clockSkew)) {
		return Identity{}, ErrInvalidToken
	}
	if claims.NotBefore != nil && claims.NotBefore.Time().After(now.Add(clockSkew)) {
		return Identity{}, ErrInvalidToken
	}
	return Identity{Issuer: claims.Issuer, Subject: claims.Subject}, nil
}
