// Package app manages HTTP lifecycle and graceful shutdown.
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"reflect"
	"time"
)

// Options configures an application.
type Options struct {
	Name, Version, Address                                                     string
	ReadTimeout, ReadHeaderTimeout, WriteTimeout, IdleTimeout, ShutdownTimeout time.Duration
	MaxHeaderBytes                                                             int
	Logger                                                                     *slog.Logger
}

// App is a runnable HTTP application.
type App struct {
	options    Options
	mux        *http.ServeMux
	middleware []func(http.Handler) http.Handler
	resources  []Resource
}

// Resource is owned by an application and closed during shutdown.
type Resource interface{ Close() }

// New validates options and creates an application.
func New(options Options) (*App, error) {
	if options.Name == "" {
		return nil, errors.New("application name is required")
	}
	if options.Address == "" {
		return nil, errors.New("HTTP address is required")
	}
	if options.Logger == nil {
		return nil, errors.New("logger is required")
	}
	if options.ReadTimeout <= 0 || options.ReadHeaderTimeout <= 0 || options.WriteTimeout <= 0 || options.IdleTimeout <= 0 || options.ShutdownTimeout <= 0 {
		return nil, errors.New("all timeouts must be positive")
	}
	if options.MaxHeaderBytes <= 0 {
		return nil, errors.New("max header bytes must be positive")
	}
	return &App{options: options, mux: http.NewServeMux()}, nil
}

// Handle registers a route using Go's method-aware ServeMux patterns.
func (a *App) Handle(pattern string, handler http.Handler) { a.mux.Handle(pattern, handler) }

// Use transfers lifecycle ownership of a resource to the application.
func (a *App) Use(resource Resource) error {
	if resource == nil || isNil(resource) {
		return errors.New("resource is required")
	}
	a.resources = append(a.resources, resource)
	return nil
}

func isNil(value any) bool {
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflected.IsNil()
	}
	return false
}

// UseHTTP appends HTTP middleware. Register middleware and routes before Run.
func (a *App) UseHTTP(middleware func(http.Handler) http.Handler) error {
	if middleware == nil {
		return errors.New("HTTP middleware is required")
	}
	a.middleware = append(a.middleware, middleware)
	return nil
}

// Run serves until the context is cancelled, then performs a bounded shutdown.
func (a *App) Run(ctx context.Context) error {
	if ctx == nil {
		return errors.New("run context is required")
	}
	defer a.closeResources()
	listener, err := net.Listen("tcp", a.options.Address)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", a.options.Address, err)
	}
	return a.serve(ctx, listener)
}

func (a *App) serve(ctx context.Context, listener net.Listener) error {
	handler := a.handler()
	server := &http.Server{Addr: a.options.Address, Handler: handler, ReadTimeout: a.options.ReadTimeout, ReadHeaderTimeout: a.options.ReadHeaderTimeout, WriteTimeout: a.options.WriteTimeout, IdleTimeout: a.options.IdleTimeout, MaxHeaderBytes: a.options.MaxHeaderBytes}
	errorsCh := make(chan error, 1)
	go func() { errorsCh <- server.Serve(listener) }()
	a.options.Logger.Info("application started", "name", a.options.Name, "version", a.options.Version, "address", listener.Addr().String())
	select {
	case err := <-errorsCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), a.options.ShutdownTimeout)
		a.options.Logger.Info("application stopping", "name", a.options.Name)
		shutdownErr := server.Shutdown(shutdownCtx)
		cancel()
		if shutdownErr != nil {
			_ = server.Close()
			<-errorsCh
			return fmt.Errorf("graceful shutdown: %w", shutdownErr)
		}
		err := <-errorsCh
		if errors.Is(err, http.ErrServerClosed) {
			a.options.Logger.Info("application stopped", "name", a.options.Name)
			return nil
		}
		return err
	}
}

func (a *App) handler() http.Handler {
	var handler http.Handler = a.mux
	for index := len(a.middleware) - 1; index >= 0; index-- {
		handler = a.middleware[index](handler)
	}
	return handler
}

func (a *App) closeResources() {
	for index := len(a.resources) - 1; index >= 0; index-- {
		a.resources[index].Close()
	}
}
