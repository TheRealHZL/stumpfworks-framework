package middleware

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestChainReplacesUnsafeRequestID(t *testing.T) {
	var contextID string
	handler := Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contextID = RequestID(r.Context())
		w.WriteHeader(http.StatusNoContent)
	}), slog.New(slog.NewTextHandler(io.Discard, nil)))
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("X-Request-ID", "unsafe value")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	id := response.Header().Get("X-Request-ID")
	if id == "" || id == request.Header.Get("X-Request-ID") || id != contextID {
		t.Fatalf("unexpected request ID %q", id)
	}
	if len(id) != 36 {
		t.Fatalf("generated ID length %d", len(id))
	}
}

func TestChainRecoversWithoutExposingPanic(t *testing.T) {
	handler := Chain(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("private detail") }), slog.New(slog.NewTextHandler(io.Discard, nil)))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("got status %d", response.Code)
	}
	if strings.Contains(response.Body.String(), "private detail") {
		t.Fatal("panic detail exposed")
	}
	if response.Header().Get("Content-Type") != "application/problem+json" {
		t.Fatalf("content type %q", response.Header().Get("Content-Type"))
	}
}

func TestChainLogsRouteWithoutRequestData(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
	mux := http.NewServeMux()
	mux.HandleFunc("POST /items/{id}", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusCreated) })
	handler := Chain(mux, logger)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/items/personal-id?token=private", nil))
	logged := output.String()
	if !strings.Contains(logged, `"status":201`) || !strings.Contains(logged, `"route":"POST /items/{id}"`) {
		t.Fatalf("unexpected log: %s", logged)
	}
	if strings.Contains(logged, "private") || strings.Contains(logged, "personal-id") {
		t.Fatalf("request data logged: %s", logged)
	}
}
