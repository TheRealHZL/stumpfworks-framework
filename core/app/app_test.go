package app

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

func validOptions() Options {
	return Options{Name: "test", Version: "dev", Address: "127.0.0.1:0", ReadTimeout: time.Second, ReadHeaderTimeout: time.Second, WriteTimeout: time.Second, IdleTimeout: time.Second, ShutdownTimeout: time.Second, MaxHeaderBytes: 1 << 20, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
}

func TestNewRejectsUnsafeOptions(t *testing.T) {
	options := validOptions()
	options.ReadTimeout = 0
	if _, err := New(options); err == nil {
		t.Fatal("expected timeout validation error")
	}
}

func TestRunStopsWhenContextIsCancelled(t *testing.T) {
	application, err := New(validOptions())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	done := make(chan error, 1)
	application.Handle("GET /", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	go func() { done <- application.Run(ctx) }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run() error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("application did not stop")
	}
}

type recordedResource struct {
	name   string
	closed *[]string
}

func (r *recordedResource) Close() { *r.closed = append(*r.closed, r.name) }

func TestRunClosesResourcesInReverseOrder(t *testing.T) {
	application, err := New(validOptions())
	if err != nil {
		t.Fatal(err)
	}
	closed := make([]string, 0, 2)
	_ = application.Use(&recordedResource{name: "first", closed: &closed})
	_ = application.Use(&recordedResource{name: "second", closed: &closed})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := application.Run(ctx); err != nil {
		t.Fatal(err)
	}
	if len(closed) != 2 || closed[0] != "second" || closed[1] != "first" {
		t.Fatalf("close order: %v", closed)
	}
}

func TestUseRejectsTypedNilResource(t *testing.T) {
	application, err := New(validOptions())
	if err != nil {
		t.Fatal(err)
	}
	var resource *recordedResource
	if err := application.Use(resource); err == nil {
		t.Fatal("accepted typed nil resource")
	}
}

func TestUseHTTPRejectsNil(t *testing.T) {
	application, err := New(validOptions())
	if err != nil {
		t.Fatal(err)
	}
	if err := application.UseHTTP(nil); err == nil {
		t.Fatal("accepted nil middleware")
	}
}

func TestUseHTTPPreservesRegistrationOrder(t *testing.T) {
	application, err := New(validOptions())
	if err != nil {
		t.Fatal(err)
	}
	order := make([]string, 0, 4)
	wrap := func(name string) func(http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, name+" before")
				next.ServeHTTP(w, r)
				order = append(order, name+" after")
			})
		}
	}
	_ = application.UseHTTP(wrap("first"))
	_ = application.UseHTTP(wrap("second"))
	application.Handle("GET /", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	application.handler().ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	want := []string{"first before", "second before", "second after", "first after"}
	if !reflect.DeepEqual(order, want) {
		t.Fatalf("order=%v want=%v", order, want)
	}
}

func TestGracefulShutdownWaitsForInflightRequest(t *testing.T) {
	options := validOptions()
	options.ShutdownTimeout = time.Second
	application, err := New(options)
	if err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	release := make(chan struct{})
	application.Handle("GET /work", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(started)
		<-release
		w.WriteHeader(http.StatusNoContent)
	}))
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	serverDone := make(chan error, 1)
	go func() { serverDone <- application.serve(ctx, listener) }()
	requestDone := make(chan error, 1)
	go func() {
		response, err := http.Get("http://" + listener.Addr().String() + "/work")
		if err == nil {
			_ = response.Body.Close()
			if response.StatusCode != http.StatusNoContent {
				err = errors.New("unexpected response status")
			}
		}
		requestDone <- err
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("request did not start")
	}
	cancel()
	select {
	case err := <-serverDone:
		t.Fatalf("server stopped before request completed: %v", err)
	case <-time.After(30 * time.Millisecond):
	}
	close(release)
	if err := <-requestDone; err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if err := <-serverDone; err != nil {
		t.Fatalf("server failed: %v", err)
	}
}

func TestShutdownDeadlineForcesConnectionClosed(t *testing.T) {
	options := validOptions()
	options.ShutdownTimeout = 20 * time.Millisecond
	application, err := New(options)
	if err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	block := make(chan struct{})
	application.Handle("GET /blocked", http.HandlerFunc(func(http.ResponseWriter, *http.Request) { close(started); <-block }))
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- application.serve(ctx, listener) }()
	go func() { _, _ = http.Get("http://" + listener.Addr().String() + "/blocked") }()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("request did not start")
	}
	cancel()
	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), "graceful shutdown") {
			t.Fatalf("unexpected error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("forced shutdown did not finish")
	}
	close(block)
}
