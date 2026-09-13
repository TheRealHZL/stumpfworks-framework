package oidc

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"
)

// ErrStaleConfiguration means no sufficiently fresh issuer keys are available.
// New logins should fail closed until Refresh succeeds.
var ErrStaleConfiguration = errors.New("OIDC configuration is unavailable or stale")

// ConfigurationCache holds an atomically refreshed, age-bounded snapshot of
// Discovery and JWKS. It has no background goroutine: the application schedules
// Refresh, observes errors, and decides how to report issuer outages.
type ConfigurationCache struct {
	issuer   string
	clientID string
	client   *http.Client
	maxAge   time.Duration

	refreshMu sync.Mutex
	mu        sync.RWMutex
	current   *Configuration
	fetchedAt time.Time
	now       func() time.Time
}

// NewConfigurationCache sets the maximum age of keys used for new logins.
// The caller must invoke Refresh before Current first succeeds.
func NewConfigurationCache(issuer, clientID string, client *http.Client, maxAge time.Duration) (*ConfigurationCache, error) {
	if !validIssuer(issuer) || clientID == "" || maxAge < time.Minute || maxAge > time.Hour {
		return nil, errors.New("invalid OIDC cache configuration")
	}
	if client != nil {
		copyClient := *client
		client = &copyClient
	}
	return &ConfigurationCache{issuer: issuer, clientID: clientID, client: client, maxAge: maxAge, now: time.Now}, nil
}

// Refresh retrieves a complete vetted configuration before replacing the old
// snapshot. A failed refresh does not extend its freshness deadline.
func (cache *ConfigurationCache) Refresh(ctx context.Context) error {
	if cache == nil {
		return ErrStaleConfiguration
	}
	cache.refreshMu.Lock()
	defer cache.refreshMu.Unlock()
	configuration, err := Discover(ctx, cache.issuer, cache.clientID, cache.client)
	if err != nil {
		return err
	}
	cache.mu.Lock()
	cache.current = configuration
	cache.fetchedAt = cache.now()
	cache.mu.Unlock()
	return nil
}

// Current returns a copy of the vetted endpoints and immutable verifier only
// while the snapshot is fresh. Existing application sessions are unaffected.
func (cache *ConfigurationCache) Current() (*Configuration, error) {
	if cache == nil {
		return nil, ErrStaleConfiguration
	}
	cache.mu.RLock()
	defer cache.mu.RUnlock()
	now := cache.now()
	if cache.current == nil || cache.fetchedAt.After(now.Add(clockSkew)) || now.Sub(cache.fetchedAt) >= cache.maxAge {
		return nil, ErrStaleConfiguration
	}
	snapshot := *cache.current
	snapshot.validUntil = cache.fetchedAt.Add(cache.maxAge)
	return &snapshot, nil
}
