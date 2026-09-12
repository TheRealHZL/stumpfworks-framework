// Package config provides typed, validated application configuration.
package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"
)

type fileConfig struct {
	HTTP struct {
		Address, ReadTimeout, ReadHeaderTimeout, WriteTimeout, IdleTimeout, QueueTimeout string
		MaxHeaderBytes, MaxConcurrentRequests                                            *int
	}
	Postgres struct {
		URL                            string
		MaxConnections, MinConnections *int32
		MaxMessageBytes                *int
		ConnectTimeout                 string
		AllowInsecure                  bool
	}
	ShutdownTimeout string
}

// Overrides contains non-secret command-line values with highest precedence.
// Secrets are intentionally excluded because process arguments may be visible
// to other users on the host.
type Overrides struct{ ConfigFile, HTTPAddress string }

// Config contains the framework settings used by the minimal application.
type Config struct {
	HTTP            HTTP
	Postgres        Postgres
	ShutdownTimeout time.Duration
}

// Postgres configures an optional PostgreSQL connection.
type Postgres struct {
	URL             string
	MaxConnections  int32
	MinConnections  int32
	ConnectTimeout  time.Duration
	MaxMessageBytes int
	AllowInsecure   bool
}

// HTTP configures the HTTP server.
type HTTP struct {
	Address               string
	ReadTimeout           time.Duration
	ReadHeaderTimeout     time.Duration
	WriteTimeout          time.Duration
	IdleTimeout           time.Duration
	MaxHeaderBytes        int
	MaxConcurrentRequests int
	QueueTimeout          time.Duration
}

// Load reads SWF environment variables over secure defaults and validates the result.
func Load() (Config, error) {
	return LoadWithOverrides(Overrides{})
}

// LoadWithOverrides applies defaults, JSON file, environment variables, and
// finally explicit non-secret command-line overrides.
func LoadWithOverrides(overrides Overrides) (Config, error) {
	cfg := Config{
		HTTP:            HTTP{Address: "127.0.0.1:8080", ReadTimeout: 5 * time.Second, ReadHeaderTimeout: 5 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 1 << 20, MaxConcurrentRequests: 128, QueueTimeout: 100 * time.Millisecond},
		Postgres:        Postgres{MaxConnections: 10, MinConnections: 1, ConnectTimeout: 5 * time.Second, MaxMessageBytes: 16 << 20},
		ShutdownTimeout: 10 * time.Second,
	}
	filename := os.Getenv("SWF_CONFIG_FILE")
	if overrides.ConfigFile != "" {
		filename = overrides.ConfigFile
	}
	if filename != "" {
		if err := applyFile(&cfg, filename); err != nil {
			return Config{}, err
		}
	}
	cfg.HTTP.Address = envOr("SWF_HTTP_ADDRESS", cfg.HTTP.Address)
	cfg.Postgres.URL = envOr("SWF_POSTGRES_URL", cfg.Postgres.URL)
	var err error
	if cfg.HTTP.MaxHeaderBytes, err = intEnv("SWF_HTTP_MAX_HEADER_BYTES", cfg.HTTP.MaxHeaderBytes); err != nil {
		return Config{}, err
	}
	if cfg.HTTP.MaxConcurrentRequests, err = intEnv("SWF_HTTP_MAX_CONCURRENT_REQUESTS", cfg.HTTP.MaxConcurrentRequests); err != nil {
		return Config{}, err
	}
	if cfg.Postgres.MaxConnections, err = int32Env("SWF_POSTGRES_MAX_CONNECTIONS", cfg.Postgres.MaxConnections); err != nil {
		return Config{}, err
	}
	if cfg.Postgres.MinConnections, err = int32Env("SWF_POSTGRES_MIN_CONNECTIONS", cfg.Postgres.MinConnections); err != nil {
		return Config{}, err
	}
	if cfg.Postgres.MaxMessageBytes, err = intEnv("SWF_POSTGRES_MAX_MESSAGE_BYTES", cfg.Postgres.MaxMessageBytes); err != nil {
		return Config{}, err
	}
	if cfg.Postgres.AllowInsecure, err = boolEnv("SWF_POSTGRES_ALLOW_INSECURE", cfg.Postgres.AllowInsecure); err != nil {
		return Config{}, err
	}
	if cfg.HTTP.ReadTimeout, err = durationEnv("SWF_HTTP_READ_TIMEOUT", cfg.HTTP.ReadTimeout); err != nil {
		return Config{}, err
	}
	if cfg.HTTP.ReadHeaderTimeout, err = durationEnv("SWF_HTTP_READ_HEADER_TIMEOUT", cfg.HTTP.ReadHeaderTimeout); err != nil {
		return Config{}, err
	}
	if cfg.HTTP.QueueTimeout, err = durationEnv("SWF_HTTP_QUEUE_TIMEOUT", cfg.HTTP.QueueTimeout); err != nil {
		return Config{}, err
	}
	if cfg.HTTP.WriteTimeout, err = durationEnv("SWF_HTTP_WRITE_TIMEOUT", cfg.HTTP.WriteTimeout); err != nil {
		return Config{}, err
	}
	if cfg.HTTP.IdleTimeout, err = durationEnv("SWF_HTTP_IDLE_TIMEOUT", cfg.HTTP.IdleTimeout); err != nil {
		return Config{}, err
	}
	if cfg.ShutdownTimeout, err = durationEnv("SWF_SHUTDOWN_TIMEOUT", cfg.ShutdownTimeout); err != nil {
		return Config{}, err
	}
	if cfg.Postgres.ConnectTimeout, err = durationEnv("SWF_POSTGRES_CONNECT_TIMEOUT", cfg.Postgres.ConnectTimeout); err != nil {
		return Config{}, err
	}
	if overrides.HTTPAddress != "" {
		cfg.HTTP.Address = overrides.HTTPAddress
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf("validate configuration: %w", err)
	}
	return cfg, nil
}

func applyFile(cfg *Config, filename string) error {
	contents, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("read configuration file: %w", err)
	}
	var values fileConfig
	decoder := json.NewDecoder(bytes.NewReader(contents))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&values); err != nil {
		return fmt.Errorf("parse configuration file: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("parse configuration file: multiple JSON values")
	}
	if values.HTTP.Address != "" {
		cfg.HTTP.Address = values.HTTP.Address
	}
	var parseErr error
	if values.HTTP.ReadTimeout != "" {
		cfg.HTTP.ReadTimeout, parseErr = time.ParseDuration(values.HTTP.ReadTimeout)
		if parseErr != nil {
			return fmt.Errorf("parse HTTP read timeout: %w", parseErr)
		}
	}
	if values.HTTP.ReadHeaderTimeout != "" {
		cfg.HTTP.ReadHeaderTimeout, parseErr = time.ParseDuration(values.HTTP.ReadHeaderTimeout)
		if parseErr != nil {
			return fmt.Errorf("parse HTTP read header timeout: %w", parseErr)
		}
	}
	if values.HTTP.MaxHeaderBytes != nil {
		cfg.HTTP.MaxHeaderBytes = *values.HTTP.MaxHeaderBytes
	}
	if values.HTTP.MaxConcurrentRequests != nil {
		cfg.HTTP.MaxConcurrentRequests = *values.HTTP.MaxConcurrentRequests
	}
	if values.HTTP.QueueTimeout != "" {
		cfg.HTTP.QueueTimeout, parseErr = time.ParseDuration(values.HTTP.QueueTimeout)
		if parseErr != nil {
			return fmt.Errorf("parse HTTP queue timeout: %w", parseErr)
		}
	}
	if values.HTTP.WriteTimeout != "" {
		cfg.HTTP.WriteTimeout, parseErr = time.ParseDuration(values.HTTP.WriteTimeout)
		if parseErr != nil {
			return fmt.Errorf("parse HTTP write timeout: %w", parseErr)
		}
	}
	if values.HTTP.IdleTimeout != "" {
		cfg.HTTP.IdleTimeout, parseErr = time.ParseDuration(values.HTTP.IdleTimeout)
		if parseErr != nil {
			return fmt.Errorf("parse HTTP idle timeout: %w", parseErr)
		}
	}
	if values.Postgres.URL != "" {
		cfg.Postgres.URL = values.Postgres.URL
	}
	if values.Postgres.MaxConnections != nil {
		cfg.Postgres.MaxConnections = *values.Postgres.MaxConnections
	}
	if values.Postgres.MinConnections != nil {
		cfg.Postgres.MinConnections = *values.Postgres.MinConnections
	}
	if values.Postgres.MaxMessageBytes != nil {
		cfg.Postgres.MaxMessageBytes = *values.Postgres.MaxMessageBytes
	}
	if values.Postgres.AllowInsecure {
		cfg.Postgres.AllowInsecure = true
	}
	if values.Postgres.ConnectTimeout != "" {
		cfg.Postgres.ConnectTimeout, parseErr = time.ParseDuration(values.Postgres.ConnectTimeout)
		if parseErr != nil {
			return fmt.Errorf("parse PostgreSQL connect timeout: %w", parseErr)
		}
	}
	if values.ShutdownTimeout != "" {
		cfg.ShutdownTimeout, parseErr = time.ParseDuration(values.ShutdownTimeout)
		if parseErr != nil {
			return fmt.Errorf("parse shutdown timeout: %w", parseErr)
		}
	}
	return nil
}

// Validate rejects configurations that cannot be operated safely.
func (c Config) Validate() error {
	if c.HTTP.Address == "" {
		return errors.New("HTTP address must not be empty")
	}
	if c.HTTP.ReadTimeout <= 0 {
		return errors.New("HTTP read timeout must be positive")
	}
	if c.HTTP.ReadHeaderTimeout <= 0 {
		return errors.New("HTTP read header timeout must be positive")
	}
	if c.HTTP.MaxHeaderBytes <= 0 {
		return errors.New("HTTP max header bytes must be positive")
	}
	if c.HTTP.MaxConcurrentRequests <= 0 {
		return errors.New("HTTP max concurrent requests must be positive")
	}
	if c.HTTP.QueueTimeout < 0 {
		return errors.New("HTTP queue timeout must not be negative")
	}
	if c.HTTP.WriteTimeout <= 0 {
		return errors.New("HTTP write timeout must be positive")
	}
	if c.HTTP.IdleTimeout <= 0 {
		return errors.New("HTTP idle timeout must be positive")
	}
	if c.ShutdownTimeout <= 0 {
		return errors.New("shutdown timeout must be positive")
	}
	if c.Postgres.MaxConnections <= 0 {
		return errors.New("PostgreSQL max connections must be positive")
	}
	if c.Postgres.MinConnections < 0 || c.Postgres.MinConnections > c.Postgres.MaxConnections {
		return errors.New("PostgreSQL min connections must be between zero and max connections")
	}
	if c.Postgres.ConnectTimeout <= 0 {
		return errors.New("PostgreSQL connect timeout must be positive")
	}
	if c.Postgres.MaxMessageBytes <= 0 {
		return errors.New("PostgreSQL max message bytes must be positive")
	}
	return nil
}

func envOr(name, fallback string) string {
	if value, ok := os.LookupEnv(name); ok {
		return value
	}
	return fallback
}

func durationEnv(name string, fallback time.Duration) (time.Duration, error) {
	value, ok := os.LookupEnv(name)
	if !ok {
		return fallback, nil
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", name, err)
	}
	return duration, nil
}

func int32Env(name string, fallback int32) (int32, error) {
	value, ok := os.LookupEnv(name)
	if !ok {
		return fallback, nil
	}
	parsed, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", name, err)
	}
	return int32(parsed), nil
}

func intEnv(name string, fallback int) (int, error) {
	value, ok := os.LookupEnv(name)
	if !ok {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", name, err)
	}
	return parsed, nil
}

func boolEnv(name string, fallback bool) (bool, error) {
	value, ok := os.LookupEnv(name)
	if !ok {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("parse %s: %w", name, err)
	}
	return parsed, nil
}
