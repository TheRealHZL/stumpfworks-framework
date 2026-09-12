package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.HTTP.Address != "127.0.0.1:8080" {
		t.Fatalf("unexpected address %q", cfg.HTTP.Address)
	}
	if cfg.Postgres.MaxConnections != 10 {
		t.Fatalf("unexpected max connections %d", cfg.Postgres.MaxConnections)
	}
}

func TestLoadFileThenEnvironmentOverride(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "config.json")
	contents := []byte(`{"HTTP":{"Address":"127.0.0.1:9000","ReadTimeout":"7s"},"Postgres":{"URL":"postgres://file/database","MaxConnections":4}}`)
	if err := os.WriteFile(filename, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SWF_CONFIG_FILE", filename)
	t.Setenv("SWF_HTTP_ADDRESS", "127.0.0.1:9001")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTP.Address != "127.0.0.1:9001" {
		t.Fatalf("environment did not override file: %q", cfg.HTTP.Address)
	}
	if cfg.HTTP.ReadTimeout.String() != "7s" || cfg.Postgres.MaxConnections != 4 {
		t.Fatalf("file values not loaded: %#v", cfg)
	}
	if cfg.Postgres.URL != "postgres://file/database" {
		t.Fatalf("file PostgreSQL URL was lost: %q", cfg.Postgres.URL)
	}
}

func TestLoadFileRejectsUnknownField(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(filename, []byte(`{"Mystery":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SWF_CONFIG_FILE", filename)
	if _, err := Load(); err == nil {
		t.Fatal("unknown field accepted")
	}
}

func TestExplicitOverrideWinsOverEnvironment(t *testing.T) {
	t.Setenv("SWF_HTTP_ADDRESS", "127.0.0.1:9001")
	cfg, err := LoadWithOverrides(Overrides{HTTPAddress: "127.0.0.1:9002"})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTP.Address != "127.0.0.1:9002" {
		t.Fatalf("explicit override lost: %q", cfg.HTTP.Address)
	}
}

func TestLoadAcceptsPostgresURL(t *testing.T) {
	t.Setenv("SWF_POSTGRES_URL", "postgres://user:secret@localhost/app")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Postgres.URL == "" {
		t.Fatal("expected PostgreSQL URL")
	}
}

func TestLoadRejectsInvalidDuration(t *testing.T) {
	t.Setenv("SWF_HTTP_READ_TIMEOUT", "eventually")
	if _, err := Load(); err == nil {
		t.Fatal("expected invalid duration error")
	}
}
