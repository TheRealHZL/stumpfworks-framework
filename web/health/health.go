// Package health provides liveness and readiness HTTP handlers.
package health

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"
)

// Check reports whether a dependency is ready.
type Check func(context.Context) error

// Handler owns the application's health endpoints.
type Handler struct {
	mu      sync.RWMutex
	checks  map[string]Check
	timeout time.Duration
}

// New creates an empty health handler. An application with no dependencies is ready.
func New() *Handler { return &Handler{checks: make(map[string]Check), timeout: 2 * time.Second} }

// NewWithTimeout creates a handler with one overall readiness deadline.
func NewWithTimeout(timeout time.Duration) (*Handler, error) {
	if timeout <= 0 {
		return nil, errors.New("health check timeout must be positive")
	}
	return &Handler{checks: make(map[string]Check), timeout: timeout}, nil
}

// Register adds or replaces a named readiness check.
func (h *Handler) Register(name string, check Check) error {
	if name == "" {
		return errors.New("health check name is required")
	}
	if check == nil {
		return errors.New("health check function is required")
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.checks[name] = check
	return nil
}

// Live reports process liveness without checking dependencies.
func (h *Handler) Live(w http.ResponseWriter, _ *http.Request) {
	respond(w, http.StatusOK, map[string]any{"status": "up"})
}

// Ready runs all registered dependency checks.
func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	checks := make(map[string]Check, len(h.checks))
	for name, check := range h.checks {
		checks[name] = check
	}
	h.mu.RUnlock()
	checkCtx, cancel := context.WithTimeout(r.Context(), h.timeout)
	defer cancel()
	type result struct {
		name string
		err  error
	}
	results := make(chan result, len(checks))
	for name, check := range checks {
		go func() {
			var err error
			defer func() {
				if recovered := recover(); recovered != nil {
					err = fmt.Errorf("health check panicked")
				}
				results <- result{name: name, err: err}
			}()
			err = check(checkCtx)
		}()
	}
	failed := make([]string, 0)
	pending := make(map[string]struct{}, len(checks))
	for name := range checks {
		pending[name] = struct{}{}
	}
	for len(pending) > 0 {
		select {
		case result := <-results:
			delete(pending, result.name)
			if result.err != nil {
				failed = append(failed, result.name)
			}
		case <-checkCtx.Done():
			for name := range pending {
				failed = append(failed, name)
			}
			pending = nil
		}
	}
	sort.Strings(failed)
	if len(failed) > 0 {
		respond(w, http.StatusServiceUnavailable, map[string]any{"status": "down", "failed_checks": failed})
		return
	}
	respond(w, http.StatusOK, map[string]any{"status": "up"})
}

func respond(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
