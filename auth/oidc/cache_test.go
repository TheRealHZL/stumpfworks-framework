package oidc

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestConfigurationCacheRefreshRotationAndExpiry(t *testing.T) {
	oldKey := testKey(t)
	newKey := testKey(t)
	var activeKeys atomic.Value
	activeKeys.Store(testJWKS(t, publicKey(oldKey, "old")))
	var broken atomic.Bool
	var server *httptest.Server
	server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if broken.Load() {
			http.Error(w, "unavailable", http.StatusServiceUnavailable)
			return
		}
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"issuer": server.URL, "authorization_endpoint": server.URL + "/authorize",
				"token_endpoint": server.URL + "/token", "jwks_uri": server.URL + "/jwks",
				"response_types_supported": []string{"code"}, "code_challenge_methods_supported": []string{"S256"},
				"id_token_signing_alg_values_supported": []string{"RS256"},
			})
		case "/jwks":
			_, _ = w.Write(activeKeys.Load().([]byte))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	cache, err := NewConfigurationCache(server.URL, testClient, server.Client(), time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cache.Current(); !errors.Is(err, ErrStaleConfiguration) {
		t.Fatal("empty cache accepted")
	}
	clock := time.Now()
	cache.now = func() time.Time { return clock }
	if err := cache.Refresh(t.Context()); err != nil {
		t.Fatal(err)
	}
	first, err := cache.Current()
	if err != nil {
		t.Fatal(err)
	}
	loginClient, err := NewLoginClient(first, testClient, "test-secret", "https://access.example.test/callback", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	loginClient.now = func() time.Time { return clock }
	oldToken := signToken(t, oldKey, "old", map[string]any{"iss": server.URL, "sub": "subject", "aud": testClient, "iat": clock.Unix(), "exp": clock.Add(5 * time.Minute).Unix(), "nonce": "nonce"})
	if _, err := first.Verifier.VerifyIDToken(oldToken, "nonce"); err != nil {
		t.Fatal(err)
	}
	first.TokenEndpoint = "https://evil.example.test/token"
	second, err := cache.Current()
	if err != nil || second.TokenEndpoint != server.URL+"/token" {
		t.Fatal("caller mutation altered cache")
	}
	activeKeys.Store(testJWKS(t, publicKey(newKey, "new")))
	broken.Store(true)
	if err := cache.Refresh(t.Context()); err == nil {
		t.Fatal("failed refresh accepted")
	}
	clock = clock.Add(30 * time.Second)
	if _, err := cache.Current(); err != nil {
		t.Fatalf("valid key window discarded early: %v", err)
	}
	if _, _, err := loginClient.Begin(); err != nil {
		t.Fatalf("fresh client rejected: %v", err)
	}
	clock = clock.Add(30 * time.Second)
	if _, err := cache.Current(); !errors.Is(err, ErrStaleConfiguration) {
		t.Fatal("stale keys accepted after failed refresh")
	}
	if _, _, err := loginClient.Begin(); !errors.Is(err, ErrStaleConfiguration) {
		t.Fatal("reused login client accepted stale keys")
	}
	broken.Store(false)
	if err := cache.Refresh(t.Context()); err != nil {
		t.Fatal(err)
	}
	rotated, err := cache.Current()
	if err != nil {
		t.Fatal(err)
	}
	newToken := signToken(t, newKey, "new", map[string]any{"iss": server.URL, "sub": "subject", "aud": testClient, "iat": clock.Unix(), "exp": clock.Add(5 * time.Minute).Unix(), "nonce": "nonce"})
	rotated.Verifier.now = func() time.Time { return clock }
	if _, err := rotated.Verifier.VerifyIDToken(newToken, "nonce"); err != nil {
		t.Fatalf("rotated key rejected: %v", err)
	}
	if _, err := rotated.Verifier.VerifyIDToken(oldToken, "nonce"); !errors.Is(err, ErrInvalidToken) {
		t.Fatal("removed key still accepted in new snapshot")
	}
}

func TestConfigurationCacheRejectsInvalidPolicy(t *testing.T) {
	for _, age := range []time.Duration{0, time.Second, time.Hour + time.Second} {
		if _, err := NewConfigurationCache(testIssuer, testClient, nil, age); err == nil {
			t.Fatalf("accepted invalid max age %s", age)
		}
	}
	if _, err := (*ConfigurationCache)(nil).Current(); !errors.Is(err, ErrStaleConfiguration) {
		t.Fatal("nil cache accepted")
	}
}
