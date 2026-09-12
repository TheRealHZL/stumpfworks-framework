package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCORSRejectsWildcardCredentials(t *testing.T) {
	if _, err := CORS(CORSOptions{AllowedOrigins: []string{"*"}, AllowCredentials: true}); err == nil {
		t.Fatal("accepted wildcard credentials")
	}
}

func TestCORSPreflight(t *testing.T) {
	wrap, err := CORS(CORSOptions{AllowedOrigins: []string{"https://app.example"}, AllowedMethods: []string{"GET", "POST"}, AllowedHeaders: []string{"Content-Type", "Authorization"}, AllowCredentials: true, MaxAge: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	handler := wrap(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("preflight reached handler") }))
	request := httptest.NewRequest(http.MethodOptions, "/", nil)
	request.Header.Set("Origin", "https://app.example")
	request.Header.Set("Access-Control-Request-Method", "POST")
	request.Header.Set("Access-Control-Request-Headers", "authorization, content-type")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("status=%d", response.Code)
	}
	if response.Header().Get("Access-Control-Allow-Origin") != "https://app.example" || response.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Fatalf("headers=%v", response.Header())
	}
}

func TestCORSRejectsUnlistedPreflight(t *testing.T) {
	wrap, err := CORS(CORSOptions{AllowedOrigins: []string{"https://app.example"}, AllowedMethods: []string{"GET"}})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodOptions, "/", nil)
	request.Header.Set("Origin", "https://evil.example")
	request.Header.Set("Access-Control-Request-Method", "GET")
	response := httptest.NewRecorder()
	wrap(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("request reached handler") })).ServeHTTP(response, request)
	if response.Code != http.StatusForbidden || response.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("unexpected response: %d %v", response.Code, response.Header())
	}
}
