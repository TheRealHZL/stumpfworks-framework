package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestMetricsUseRoutePatternAndBoundedMethod(t *testing.T) {
	registry := New()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /users/{id}", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	handler := registry.Middleware(mux)
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/users/private-id?token=private", nil))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("UNUSUAL-METHOD", "/users/private-id", nil))
	response := httptest.NewRecorder()
	registry.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body := response.Body.String()
	if response.Code != http.StatusOK || !strings.Contains(response.Header().Get("Content-Type"), "text/plain") {
		t.Fatalf("unexpected exposition response: %d, %q", response.Code, response.Header().Get("Content-Type"))
	}
	if !strings.Contains(body, `swf_http_requests_total{method="GET",route="GET /users/{id}",status="204"} 1`) {
		t.Fatalf("missing registered route: %s", body)
	}
	if !strings.Contains(body, `swf_http_requests_total{method="OTHER",route="unmatched",status="405"} 1`) {
		t.Fatalf("missing bounded unknown method: %s", body)
	}
	if strings.Contains(body, "private-id") || strings.Contains(body, "token=private") || strings.Contains(body, "UNUSUAL-METHOD") {
		t.Fatalf("request data exposed: %s", body)
	}
	if !strings.Contains(body, `le="+Inf"} 1`) {
		t.Fatal("missing infinite histogram bucket")
	}
}

func TestMetricsConcurrentRequestsAndScrapes(t *testing.T) {
	registry := New()
	handler := registry.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusAccepted) }))
	var group sync.WaitGroup
	for range 100 {
		group.Go(func() {
			handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/", nil))
			registry.Handler().ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/metrics", nil))
		})
	}
	group.Wait()
	response := httptest.NewRecorder()
	registry.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if !strings.Contains(response.Body.String(), `swf_http_requests_total{method="POST",route="unmatched",status="202"} 100`) {
		t.Fatalf("unexpected request count: %s", response.Body.String())
	}
}

func TestMetricLabelEscaping(t *testing.T) {
	got := (key{method: "GET", route: "a\"b\\c\nd", status: 200}).text()
	if got != `{method="GET",route="a\"b\\c\nd",status="200"}` {
		t.Fatalf("unexpected escaped label: %q", got)
	}
}
