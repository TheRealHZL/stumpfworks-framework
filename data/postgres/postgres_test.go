package postgres

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestOpenValidatesOptions(t *testing.T) {
	_, err := Open(context.Background(), Options{})
	if err == nil || !strings.Contains(err.Error(), "URL is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOpenDoesNotExposeCredentialsOnParseError(t *testing.T) {
	secret := "do-not-expose"
	_, err := Open(context.Background(), Options{URL: "postgres://user:" + secret + "@%zz", MaxConnections: 1, ConnectTimeout: time.Second, MaxMessageBytes: 1024})
	if err == nil {
		t.Fatal("expected parse error")
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatalf("error exposed credentials: %v", err)
	}
}

func TestOpenRejectsPlaintextByDefault(t *testing.T) {
	_, err := Open(context.Background(), Options{URL: "postgres://localhost/app?sslmode=disable", MaxConnections: 1, ConnectTimeout: time.Second, MaxMessageBytes: 1024})
	if err == nil || !strings.Contains(err.Error(), "TLS is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWithinTransactionRejectsNilCallback(t *testing.T) {
	pool := &Pool{}
	if err := pool.WithinTransaction(context.Background(), nil); err == nil {
		t.Fatal("accepted nil callback")
	}
}
