package health

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestReadyReportsFailedChecks(t *testing.T) {
	h := New()
	_ = h.Register("database", func(_ context.Context) error { return errors.New("unavailable") })
	recorder := httptest.NewRecorder()
	h.Ready(recorder, httptest.NewRequest(http.MethodGet, "/health/ready", nil))
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("got status %d", recorder.Code)
	}
}

func TestReadyTimesOutAndContainsPanics(t *testing.T) {
	h, err := NewWithTimeout(20 * time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	_ = h.Register("slow", func(ctx context.Context) error { <-ctx.Done(); return ctx.Err() })
	_ = h.Register("panic", func(context.Context) error { panic("private") })
	requestCtx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	request := httptest.NewRequest(http.MethodGet, "/health/ready", nil).WithContext(requestCtx)
	recorder := httptest.NewRecorder()
	h.Ready(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("got status %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), `"panic"`) || !strings.Contains(recorder.Body.String(), `"slow"`) {
		t.Fatalf("missing failed checks: %s", recorder.Body.String())
	}
	if strings.Contains(recorder.Body.String(), "private") {
		t.Fatalf("panic exposed: %s", recorder.Body.String())
	}
}

func TestRegisterRejectsInvalidCheck(t *testing.T) {
	h := New()
	if err := h.Register("", func(context.Context) error { return nil }); err == nil {
		t.Fatal("accepted empty name")
	}
	if err := h.Register("database", nil); err == nil {
		t.Fatal("accepted nil check")
	}
}
