package oidc

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"
)

const maxPendingTransactions = 10000

// TransactionStore is a bounded, process-local store for pending logins. It
// binds a transaction to an independent random browser secret. Applications
// with multiple replicas need shared, atomic storage instead.
type TransactionStore struct {
	mu       sync.Mutex
	capacity int
	entries  map[string]pendingEntry
	now      func() time.Time
}

type pendingEntry struct {
	transaction *Transaction
	bindingHash [sha256.Size]byte
}

// NewTransactionStore creates a bounded store. It never evicts a live login
// to make room; callers can show a temporary unavailable response when full.
func NewTransactionStore(capacity int) (*TransactionStore, error) {
	if capacity < 1 || capacity > maxPendingTransactions {
		return nil, errors.New("invalid OIDC transaction capacity")
	}
	return &TransactionStore{capacity: capacity, entries: make(map[string]pendingEntry), now: time.Now}, nil
}

// NewBrowserBinding returns a 256-bit random secret for a Secure, HttpOnly,
// host-only, SameSite=Lax cookie. Never put it in a URL or log. The cookie must
// be read by the application's fixed OIDC callback endpoint.
func NewBrowserBinding() (string, error) { return randomURLString() }

// BindingCookie prepares the independent browser secret as a hardened cookie.
// The application must set it before redirecting to the issuer and clear it
// after the callback. Name must use the browser-enforced __Host- prefix.
func BindingCookie(name, browserBinding string) (*http.Cookie, error) {
	if _, valid := decodeBinding(browserBinding); !valid {
		return nil, ErrLoginFailed
	}
	cookie := &http.Cookie{Name: name, Value: browserBinding, Path: "/", MaxAge: int(transactionLifetime.Seconds()), Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode}
	if err := validateBindingCookie(cookie); err != nil {
		return nil, err
	}
	return cookie, nil
}

// ClearBindingCookie removes the browser binding after a callback, including
// failed callbacks. It uses the same host-only cookie scope as BindingCookie.
func ClearBindingCookie(name string) (*http.Cookie, error) {
	cookie := &http.Cookie{Name: name, Path: "/", MaxAge: -1, Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode}
	if err := validateBindingCookie(cookie); err != nil {
		return nil, err
	}
	return cookie, nil
}

func validateBindingCookie(cookie *http.Cookie) error {
	if !strings.HasPrefix(cookie.Name, "__Host-") {
		return errors.New("OIDC binding cookie requires __Host- prefix")
	}
	if err := cookie.Valid(); err != nil {
		return errors.New("invalid OIDC binding cookie name")
	}
	return nil
}

func decodeBinding(binding string) ([]byte, bool) {
	if len(binding) != 43 {
		return nil, false
	}
	decoded, err := base64.RawURLEncoding.DecodeString(binding)
	return decoded, err == nil && len(decoded) == 32 && base64.RawURLEncoding.EncodeToString(decoded) == binding
}

// Put stores one transaction under its state, hashed against browserBinding.
// The binding must be independent of the transaction's state and nonce.
func (store *TransactionStore) Put(transaction *Transaction, browserBinding string) error {
	decoded, valid := decodeBinding(browserBinding)
	if store == nil || transaction == nil || transaction.owner == nil || len(transaction.state) != 43 || transaction.used.Load() || !valid {
		return ErrLoginFailed
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if len(store.entries) >= store.capacity {
		store.pruneExpired()
	}
	now := store.now()
	if now.Sub(transaction.createdAt) > transactionLifetime || transaction.createdAt.After(now.Add(clockSkew)) {
		return ErrLoginFailed
	}
	if _, exists := store.entries[transaction.state]; exists || len(store.entries) >= store.capacity {
		return ErrLoginFailed
	}
	store.entries[transaction.state] = pendingEntry{transaction: transaction, bindingHash: sha256.Sum256(decoded)}
	return nil
}

// Take atomically removes one transaction only when both state and the
// independent browser cookie match. Failed binding does not consume it.
func (store *TransactionStore) Take(state, browserBinding string) (*Transaction, error) {
	decoded, valid := decodeBinding(browserBinding)
	if store == nil || len(state) != 43 || !valid {
		return nil, ErrLoginFailed
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	entry, exists := store.entries[state]
	if !exists {
		return nil, ErrLoginFailed
	}
	if store.now().Sub(entry.transaction.createdAt) > transactionLifetime || entry.transaction.used.Load() {
		delete(store.entries, state)
		return nil, ErrLoginFailed
	}
	hash := sha256.Sum256(decoded)
	if subtle.ConstantTimeCompare(hash[:], entry.bindingHash[:]) != 1 {
		return nil, ErrLoginFailed
	}
	delete(store.entries, state)
	if entry.transaction.used.Load() {
		return nil, ErrLoginFailed
	}
	return entry.transaction, nil
}

func (store *TransactionStore) pruneExpired() {
	now := store.now()
	for state, entry := range store.entries {
		if now.Sub(entry.transaction.createdAt) > transactionLifetime || entry.transaction.used.Load() {
			delete(store.entries, state)
		}
	}
}
