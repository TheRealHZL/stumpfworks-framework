package oidc

import (
	"errors"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func testLoginClient(t *testing.T) *LoginClient {
	t.Helper()
	key := testKey(t)
	verifier, err := NewVerifier(testIssuer, testClient, testJWKS(t, publicKey(key, "current")))
	if err != nil {
		t.Fatal(err)
	}
	client, err := NewLoginClient(&Configuration{AuthorizationEndpoint: testIssuer + "/authorize", TokenEndpoint: testIssuer + "/token", Verifier: verifier}, testClient, "test-secret", "https://access.example.test/callback", nil)
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func TestStoreBindsAndConsumesTransaction(t *testing.T) {
	client := testLoginClient(t)
	store, err := NewTransactionStore(2)
	if err != nil {
		t.Fatal(err)
	}
	_, transaction, err := client.Begin()
	if err != nil {
		t.Fatal(err)
	}
	binding, err := NewBrowserBinding()
	if err != nil {
		t.Fatal(err)
	}
	wrongBinding, err := NewBrowserBinding()
	if err != nil {
		t.Fatal(err)
	}
	if binding == transaction.State() || len(binding) != 43 {
		t.Fatal("browser binding is not independent and random")
	}
	cookie, err := BindingCookie("__Host-swf_oidc", binding)
	if err != nil || !cookie.Secure || !cookie.HttpOnly || cookie.SameSite != http.SameSiteLaxMode || cookie.Domain != "" || cookie.Path != "/" || cookie.MaxAge != 300 {
		t.Fatalf("unsafe binding cookie: %+v, %v", cookie, err)
	}
	if _, err := BindingCookie("swf_oidc", binding); err == nil {
		t.Fatal("accepted cookie without host-only prefix")
	}
	if err := store.Put(transaction, binding); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Take(transaction.State(), wrongBinding); !errors.Is(err, ErrLoginFailed) {
		t.Fatal("wrong browser binding accepted")
	}
	got, err := store.Take(transaction.State(), binding)
	if err != nil || got != transaction {
		t.Fatalf("valid browser rejected: %v", err)
	}
	if _, err := store.Take(transaction.State(), binding); !errors.Is(err, ErrLoginFailed) {
		t.Fatal("transaction replay accepted")
	}
}

func TestStoreCapacityExpiryAndInvalidInput(t *testing.T) {
	client := testLoginClient(t)
	store, err := NewTransactionStore(1)
	if err != nil {
		t.Fatal(err)
	}
	binding, err := NewBrowserBinding()
	if err != nil {
		t.Fatal(err)
	}
	_, first, _ := client.Begin()
	_, second, _ := client.Begin()
	if err := store.Put(first, binding); err != nil {
		t.Fatal(err)
	}
	if err := store.Put(second, binding); !errors.Is(err, ErrLoginFailed) {
		t.Fatal("capacity limit not enforced")
	}
	if err := store.Put(first, binding); !errors.Is(err, ErrLoginFailed) {
		t.Fatal("duplicate state accepted")
	}
	if _, err := store.Take(first.State(), "invalid"); !errors.Is(err, ErrLoginFailed) {
		t.Fatal("invalid browser secret accepted")
	}
	store.now = func() time.Time { return first.createdAt.Add(transactionLifetime + time.Second) }
	if _, err := store.Take(first.State(), binding); !errors.Is(err, ErrLoginFailed) {
		t.Fatal("expired transaction accepted")
	}
	_, fresh, _ := client.Begin()
	fresh.createdAt = store.now()
	if err := store.Put(fresh, binding); err != nil {
		t.Fatalf("expired entry not released: %v", err)
	}
	if _, err := NewTransactionStore(0); err == nil {
		t.Fatal("zero capacity accepted")
	}
	if _, err := NewTransactionStore(maxPendingTransactions + 1); err == nil {
		t.Fatal("unbounded capacity accepted")
	}
}

func TestStoreConcurrentTakeIsOneUse(t *testing.T) {
	client := testLoginClient(t)
	store, _ := NewTransactionStore(1)
	binding, _ := NewBrowserBinding()
	_, transaction, _ := client.Begin()
	if err := store.Put(transaction, binding); err != nil {
		t.Fatal(err)
	}
	var successes atomic.Int32
	var group sync.WaitGroup
	for range 20 {
		group.Go(func() {
			if _, err := store.Take(transaction.State(), binding); err == nil {
				successes.Add(1)
			}
		})
	}
	group.Wait()
	if successes.Load() != 1 {
		t.Fatalf("transaction used %d times", successes.Load())
	}
}
