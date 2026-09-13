package oidc

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"io"
	"time"
)

const (
	transactionFormatVersion = 1
	transactionPlainSize     = 3*43 + 8
	transactionSealedSize    = 1 + 12 + transactionPlainSize + 16
)

// TransactionCodec seals pending logins for trusted server-side storage. Use
// one persistent, random 32-byte key per application across all replicas. The
// consumer must atomically remove each stored record on callback; sealing does
// not itself enforce one-use storage.
type TransactionCodec struct {
	aead cipher.AEAD
}

// NewTransactionCodec accepts a separate, persistent 256-bit encryption key.
func NewTransactionCodec(key []byte) (*TransactionCodec, error) {
	if len(key) != 32 {
		return nil, ErrLoginFailed
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, ErrLoginFailed
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, ErrLoginFailed
	}
	return &TransactionCodec{aead: aead}, nil
}

// Seal encrypts a fresh transaction. The returned bytes contain state, nonce,
// and PKCE verifier inside authenticated ciphertext; never log or send them to
// the browser. Store them alongside hashed state and browser binding values.
func (codec *TransactionCodec) Seal(transaction *Transaction) ([]byte, error) {
	if codec == nil || codec.aead == nil || transaction == nil || transaction.owner == nil || transaction.used.Load() ||
		!validTransactionFields(transaction) {
		return nil, ErrLoginFailed
	}
	client := transaction.owner
	if !validTransactionTime(client.now(), transaction.createdAt) {
		return nil, ErrLoginFailed
	}
	plaintext := make([]byte, 0, transactionPlainSize)
	plaintext = append(plaintext, transaction.state...)
	plaintext = append(plaintext, transaction.nonce...)
	plaintext = append(plaintext, transaction.verifier...)
	var timestamp [8]byte
	binary.BigEndian.PutUint64(timestamp[:], uint64(transaction.createdAt.UnixNano()))
	plaintext = append(plaintext, timestamp[:]...)
	nonce := make([]byte, codec.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, ErrLoginFailed
	}
	sealed := []byte{transactionFormatVersion}
	sealed = append(sealed, nonce...)
	sealed = codec.aead.Seal(sealed, nonce, plaintext, transactionAAD(client))
	return sealed, nil
}

// Open restores a transaction for the same LoginClient. The consumer must
// first atomically take the sealed record using both state and browser binding.
// An old, malformed, tampered, or cross-client record fails closed.
func (codec *TransactionCodec) Open(client *LoginClient, sealed []byte) (*Transaction, error) {
	if codec == nil || codec.aead == nil || client == nil || client.configuration == nil ||
		len(sealed) != transactionSealedSize || sealed[0] != transactionFormatVersion {
		return nil, ErrLoginFailed
	}
	nonceSize := codec.aead.NonceSize()
	plaintext, err := codec.aead.Open(nil, sealed[1:1+nonceSize], sealed[1+nonceSize:], transactionAAD(client))
	if err != nil || len(plaintext) != transactionPlainSize {
		return nil, ErrLoginFailed
	}
	transaction := &Transaction{
		owner:     client,
		state:     string(plaintext[:43]),
		nonce:     string(plaintext[43:86]),
		verifier:  string(plaintext[86:129]),
		createdAt: time.Unix(0, int64(binary.BigEndian.Uint64(plaintext[129:]))),
	}
	if !validTransactionFields(transaction) || !validTransactionTime(client.now(), transaction.createdAt) {
		return nil, ErrLoginFailed
	}
	return transaction, nil
}

func validTransactionFields(transaction *Transaction) bool {
	_, stateOK := decodeBinding(transaction.state)
	_, nonceOK := decodeBinding(transaction.nonce)
	_, verifierOK := decodeBinding(transaction.verifier)
	return stateOK && nonceOK && verifierOK
}

func validTransactionTime(now, createdAt time.Time) bool {
	return !createdAt.IsZero() && !createdAt.After(now.Add(clockSkew)) && now.Sub(createdAt) <= transactionLifetime
}

func transactionAAD(client *LoginClient) []byte {
	config := client.configuration
	parts := []string{config.Verifier.issuer, client.clientID, client.redirectURI}
	hash := sha256.New()
	var length [4]byte
	for _, part := range parts {
		binary.BigEndian.PutUint32(length[:], uint32(len(part)))
		_, _ = hash.Write(length[:])
		_, _ = hash.Write([]byte(part))
	}
	return hash.Sum(nil)
}
