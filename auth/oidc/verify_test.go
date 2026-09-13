package oidc

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"testing"
	"time"

	jose "github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
)

const (
	testIssuer = "https://identity.example.test"
	testClient = "access-client"
)

func testKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func testJWKS(t *testing.T, keys ...jose.JSONWebKey) []byte {
	t.Helper()
	encoded, err := json.Marshal(jose.JSONWebKeySet{Keys: keys})
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func publicKey(key *rsa.PrivateKey, kid string) jose.JSONWebKey {
	return jose.JSONWebKey{Key: &key.PublicKey, KeyID: kid, Algorithm: string(jose.RS256), Use: "sig"}
}

func signToken(t *testing.T, key *rsa.PrivateKey, kid string, claims map[string]any) string {
	t.Helper()
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: key}, (&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", kid))
	if err != nil {
		t.Fatal(err)
	}
	token, err := jwt.Signed(signer).Claims(claims).Serialize()
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func validClaims(now time.Time) map[string]any {
	return map[string]any{"iss": testIssuer, "sub": "opaque-subject", "aud": testClient, "iat": now.Unix(), "exp": now.Add(5 * time.Minute).Unix(), "nonce": "nonce-1"}
}

func TestVerifyIdentityLikeIDToken(t *testing.T) {
	key := testKey(t)
	now := time.Unix(1_800_000_000, 0)
	verifier, err := NewVerifier(testIssuer, testClient, testJWKS(t, publicKey(key, "current")))
	if err != nil {
		t.Fatal(err)
	}
	verifier.now = func() time.Time { return now }
	identity, err := verifier.VerifyIDToken(signToken(t, key, "current", validClaims(now)), "nonce-1")
	if err != nil || identity != (Identity{Issuer: testIssuer, Subject: "opaque-subject"}) {
		t.Fatalf("unexpected identity: %+v, %v", identity, err)
	}
}

func TestRejectsInvalidClaims(t *testing.T) {
	key := testKey(t)
	now := time.Unix(1_800_000_000, 0)
	verifier, err := NewVerifier(testIssuer, testClient, testJWKS(t, publicKey(key, "current")))
	if err != nil {
		t.Fatal(err)
	}
	verifier.now = func() time.Time { return now }
	tests := map[string]func(map[string]any){
		"issuer":       func(c map[string]any) { c["iss"] = "https://other.example.test" },
		"subject":      func(c map[string]any) { c["sub"] = "" },
		"audience":     func(c map[string]any) { c["aud"] = "other-client" },
		"multi-aud":    func(c map[string]any) { c["aud"] = []string{testClient, "other-client"} },
		"wrong-azp":    func(c map[string]any) { c["azp"] = "other-client" },
		"expired":      func(c map[string]any) { c["exp"] = now.Unix() },
		"missing-exp":  func(c map[string]any) { delete(c, "exp") },
		"missing-iat":  func(c map[string]any) { delete(c, "iat") },
		"future-iat":   func(c map[string]any) { c["iat"] = now.Add(2 * time.Minute).Unix() },
		"bad-lifetime": func(c map[string]any) { c["exp"] = now.Add(-time.Minute).Unix() },
		"future-nbf":   func(c map[string]any) { c["nbf"] = now.Add(2 * time.Minute).Unix() },
		"nonce":        func(c map[string]any) { c["nonce"] = "other-nonce" },
		"long-lived":   func(c map[string]any) { c["exp"] = now.Add(time.Hour).Unix() },
		"zero-iat":     func(c map[string]any) { c["iat"] = int64(0) },
	}
	for name, modify := range tests {
		t.Run(name, func(t *testing.T) {
			claims := validClaims(now)
			modify(claims)
			if _, err := verifier.VerifyIDToken(signToken(t, key, "current", claims), "nonce-1"); !errors.Is(err, ErrInvalidToken) {
				t.Fatalf("accepted invalid claims: %v", err)
			}
		})
	}
	claims := validClaims(now)
	claims["aud"] = []string{testClient, "other-client"}
	claims["azp"] = testClient
	if _, err := verifier.VerifyIDToken(signToken(t, key, "current", claims), "nonce-1"); err != nil {
		t.Fatalf("valid multi-audience token rejected: %v", err)
	}
}

func TestRejectsUntrustedSignatureAndHeader(t *testing.T) {
	key := testKey(t)
	other := testKey(t)
	now := time.Unix(1_800_000_000, 0)
	verifier, err := NewVerifier(testIssuer, testClient, testJWKS(t, publicKey(key, "current")))
	if err != nil {
		t.Fatal(err)
	}
	verifier.now = func() time.Time { return now }
	for name, token := range map[string]string{
		"wrong signature": signToken(t, other, "current", validClaims(now)),
		"unknown kid":     signToken(t, key, "unknown", validClaims(now)),
		"malformed":       "not.a.jwt",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := verifier.VerifyIDToken(token, "nonce-1"); !errors.Is(err, ErrInvalidToken) {
				t.Fatalf("accepted token: %v", err)
			}
		})
	}
	hmacSigner, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.HS256, Key: []byte("a-test-only-hmac-key-with-enough-length")}, (&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", "current"))
	if err != nil {
		t.Fatal(err)
	}
	hmacToken, err := jwt.Signed(hmacSigner).Claims(validClaims(now)).Serialize()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := verifier.VerifyIDToken(hmacToken, "nonce-1"); !errors.Is(err, ErrInvalidToken) {
		t.Fatal("accepted HS256 token")
	}
	if _, err := verifier.VerifyIDToken(signToken(t, key, "current", validClaims(now)), ""); !errors.Is(err, ErrInvalidToken) {
		t.Fatal("accepted empty expected nonce")
	}
	if _, err := (*Verifier)(nil).VerifyIDToken(signToken(t, key, "current", validClaims(now)), "nonce-1"); !errors.Is(err, ErrInvalidToken) {
		t.Fatal("accepted nil verifier")
	}
}

func TestJWKSRotationAndRejection(t *testing.T) {
	oldKey := testKey(t)
	newKey := testKey(t)
	now := time.Unix(1_800_000_000, 0)
	verifier, err := NewVerifier(testIssuer, testClient, testJWKS(t, publicKey(oldKey, "old"), publicKey(newKey, "new")))
	if err != nil {
		t.Fatal(err)
	}
	verifier.now = func() time.Time { return now }
	for _, pair := range []struct {
		key *rsa.PrivateKey
		kid string
	}{{oldKey, "old"}, {newKey, "new"}} {
		if _, err := verifier.VerifyIDToken(signToken(t, pair.key, pair.kid, validClaims(now)), "nonce-1"); err != nil {
			t.Fatalf("overlapping key %s rejected: %v", pair.kid, err)
		}
	}
	invalid := []struct {
		name string
		jwks []byte
	}{
		{"duplicate kid", testJWKS(t, publicKey(oldKey, "same"), publicKey(newKey, "same"))},
		{"private key", testJWKS(t, jose.JSONWebKey{Key: oldKey, KeyID: "private", Algorithm: "RS256", Use: "sig"})},
		{"wrong algorithm", testJWKS(t, jose.JSONWebKey{Key: &oldKey.PublicKey, KeyID: "wrong", Algorithm: "PS256", Use: "sig"})},
		{"wrong use", testJWKS(t, jose.JSONWebKey{Key: &oldKey.PublicKey, KeyID: "wrong", Algorithm: "RS256", Use: "enc"})},
		{"missing kid", testJWKS(t, publicKey(oldKey, ""))},
		{"empty", []byte(`{"keys":[]}`)},
	}
	for _, test := range invalid {
		t.Run(test.name, func(t *testing.T) {
			if _, err := NewVerifier(testIssuer, testClient, test.jwks); err == nil {
				t.Fatal("accepted invalid JWKS")
			}
		})
	}
}

func TestRejectsInvalidConfiguration(t *testing.T) {
	key := testKey(t)
	set := testJWKS(t, publicKey(key, "current"))
	for _, issuer := range []string{"", "http://identity.example.test", "https://identity.example.test/path", "https://identity.example.test/", "https://user@identity.example.test"} {
		if _, err := NewVerifier(issuer, testClient, set); err == nil {
			t.Fatalf("accepted issuer %q", issuer)
		}
	}
	if _, err := NewVerifier(testIssuer, "", set); err == nil {
		t.Fatal("accepted empty client ID")
	}
}
