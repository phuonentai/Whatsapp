// Package secrets provides envelope encryption (AES-256-GCM) for integration
// credentials stored per-organization. The master key comes from the
// environment (SIIGO_MASTER_KEY) and is never persisted. Ciphertext includes
// the nonce and GCM tag, so each encryption is self-contained.
//
// Invariant: secret material must never appear in logs, error messages, or
// API responses. Errors returned by this package contain no key or ciphertext.
package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

const (
	// KeySize is the AES-256 key size in bytes.
	KeySize = 32
	// nonceSize is the AES-GCM standard nonce size in bytes.
	nonceSize = 12
)

var (
	// ErrInvalidKey is returned when the master key is missing or has the
	// wrong size.
	ErrInvalidKey = errors.New("invalid encryption master key")
	// ErrDecrypt is returned when ciphertext cannot be decrypted (tampered,
	// wrong key, or corrupt input).
	ErrDecrypt = errors.New("failed to decrypt secret")
)

// Envelope encrypts and decrypts credentials with a master key from the
// environment. Not safe for concurrent mutation of the key itself (key is
// read once); safe for concurrent use of Encrypt/Decrypt.
type Envelope struct {
	aead cipher.AEAD
}

// NewEnvelope builds an Envelope from a base64-encoded 32-byte master key.
// The key is only ever referenced in memory.
func NewEnvelope(masterKeyB64 string) (*Envelope, error) {
	if masterKeyB64 == "" {
		return nil, fmt.Errorf("%w: SIIGO_MASTER_KEY is empty", ErrInvalidKey)
	}
	raw, err := base64.StdEncoding.DecodeString(masterKeyB64)
	if err != nil {
		return nil, fmt.Errorf("%w: SIIGO_MASTER_KEY is not valid base64", ErrInvalidKey)
	}
	if len(raw) != KeySize {
		return nil, fmt.Errorf("%w: SIIGO_MASTER_KEY must decode to %d bytes, got %d", ErrInvalidKey, KeySize, len(raw))
	}
	block, err := aes.NewCipher(raw)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidKey, err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize GCM: %w", err)
	}
	return &Envelope{aead: aead}, nil
}

// Encrypt seals plaintext into ciphertext+nonce+tag. The returned bytes
// contain no plaintext.
func (e *Envelope) Encrypt(plaintext string) ([]byte, error) {
	if plaintext == "" {
		return nil, errors.New("cannot encrypt empty secret")
	}
	nonce := make([]byte, nonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}
	// Layout: nonce || ciphertext(+tag). GCM appends the tag to the
	// ciphertext, so the blob is self-contained.
	return e.aead.Seal(nonce, nonce, []byte(plaintext), nil), nil
}

// Decrypt opens a blob produced by Encrypt. Errors carry no key or content
// material; a tampered blob yields ErrDecrypt.
func (e *Envelope) Decrypt(blob []byte) (string, error) {
	if len(blob) <= nonceSize {
		return "", fmt.Errorf("%w: truncated blob", ErrDecrypt)
	}
	nonce, ciphertext := blob[:nonceSize], blob[nonceSize:]
	plain, err := e.aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("%w: authentication failed", ErrDecrypt)
	}
	return string(plain), nil
}
