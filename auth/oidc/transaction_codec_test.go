package oidc

import (
	"bytes"
	"errors"
	"testing"
	"time"
)

func TestTransactionCodecRestoresOnlyFreshMatchingClient(t *testing.T) {
	client := testLoginClient(t)
	codec, err := NewTransactionCodec(bytes.Repeat([]byte{7}, 32))
	if err != nil {
		t.Fatal(err)
	}
	_, transaction, err := client.Begin()
	if err != nil {
		t.Fatal(err)
	}
	sealed, err := codec.Seal(transaction)
	if err != nil || len(sealed) != transactionSealedSize {
		t.Fatalf("failed to seal transaction: %v", err)
	}
	if bytes.Contains(sealed, []byte(transaction.state)) || bytes.Contains(sealed, []byte(transaction.verifier)) {
		t.Fatal("sealed record exposed transaction secret")
	}
	restarted, err := NewLoginClient(client.configuration, testClient, "rotated-secret", client.redirectURI, nil)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := codec.Open(restarted, sealed)
	if err != nil || restored.State() != transaction.State() || restored.nonce != transaction.nonce || restored.verifier != transaction.verifier || restored.owner != restarted {
		t.Fatalf("failed to restore transaction: %v", err)
	}
	wrongRedirect, err := NewLoginClient(client.configuration, testClient, "secret", "https://access.example.test/other-callback", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := codec.Open(wrongRedirect, sealed); !errors.Is(err, ErrLoginFailed) {
		t.Fatal("accepted transaction for a different callback")
	}
	tampered := bytes.Clone(sealed)
	tampered[len(tampered)-1] ^= 1
	if _, err := codec.Open(restarted, tampered); !errors.Is(err, ErrLoginFailed) {
		t.Fatal("accepted tampered transaction")
	}
	unknownVersion := bytes.Clone(sealed)
	unknownVersion[0]++
	if _, err := codec.Open(restarted, unknownVersion); !errors.Is(err, ErrLoginFailed) {
		t.Fatal("accepted unknown record version")
	}
	wrongKey, _ := NewTransactionCodec(bytes.Repeat([]byte{8}, 32))
	if _, err := wrongKey.Open(restarted, sealed); !errors.Is(err, ErrLoginFailed) {
		t.Fatal("accepted transaction with wrong key")
	}
	restarted.now = func() time.Time { return transaction.createdAt.Add(transactionLifetime + time.Second) }
	if _, err := codec.Open(restarted, sealed); !errors.Is(err, ErrLoginFailed) {
		t.Fatal("accepted expired transaction")
	}
	client.now = restarted.now
	if _, err := codec.Seal(transaction); !errors.Is(err, ErrLoginFailed) {
		t.Fatal("sealed expired transaction")
	}
}

func TestTransactionCodecRejectsInvalidInputs(t *testing.T) {
	client := testLoginClient(t)
	if _, err := NewTransactionCodec(make([]byte, 31)); !errors.Is(err, ErrLoginFailed) {
		t.Fatal("accepted short encryption key")
	}
	codec, _ := NewTransactionCodec(make([]byte, 32))
	if _, err := codec.Seal(nil); !errors.Is(err, ErrLoginFailed) {
		t.Fatal("accepted nil transaction")
	}
	if _, err := codec.Open(client, nil); !errors.Is(err, ErrLoginFailed) {
		t.Fatal("accepted empty record")
	}
	_, transaction, _ := client.Begin()
	transaction.used.Store(true)
	if _, err := codec.Seal(transaction); !errors.Is(err, ErrLoginFailed) {
		t.Fatal("sealed used transaction")
	}
}
