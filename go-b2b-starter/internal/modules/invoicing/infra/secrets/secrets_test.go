package secrets

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"
)

func testKey(t *testing.T) string {
	t.Helper()
	raw := make([]byte, KeySize)
	for i := range raw {
		raw[i] = byte(i + 1)
	}
	return base64.StdEncoding.EncodeToString(raw)
}

func TestEnvelope_RoundTrip(t *testing.T) {
	env, err := NewEnvelope(testKey(t))
	if err != nil {
		t.Fatalf("NewEnvelope failed: %v", err)
	}
	blob, err := env.Encrypt("client_secret_super_secreto")
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}
	if strings.Contains(string(blob), "client_secret") {
		t.Fatal("ciphertext leaked plaintext")
	}
	plain, err := env.Decrypt(blob)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}
	if plain != "client_secret_super_secreto" {
		t.Fatalf("round trip mismatch: %q", plain)
	}
}

func TestEnvelope_DistinctCiphertexts(t *testing.T) {
	env, _ := NewEnvelope(testKey(t))
	a, err := env.Encrypt("mismo valor")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := env.Encrypt("mismo valor")
	if string(a) == string(b) {
		t.Fatal("nonce reuse detected: identical ciphertexts")
	}
}

func TestEnvelope_TamperDetected(t *testing.T) {
	env, _ := NewEnvelope(testKey(t))
	blob, err := env.Encrypt("secreto")
	if err != nil {
		t.Fatal(err)
	}
	blob[len(blob)-1] ^= 0xFF
	_, err = env.Decrypt(blob)
	if !errors.Is(err, ErrDecrypt) {
		t.Fatalf("expected ErrDecrypt, got %v", err)
	}
	if strings.Contains(err.Error(), "secreto") {
		t.Fatal("error leaked plaintext material")
	}
}

func TestEnvelope_InvalidKeys(t *testing.T) {
	if _, err := NewEnvelope(""); !errors.Is(err, ErrInvalidKey) {
		t.Fatalf("expected ErrInvalidKey for empty key, got %v", err)
	}
	if _, err := NewEnvelope("not-base64!!"); !errors.Is(err, ErrInvalidKey) {
		t.Fatalf("expected ErrInvalidKey for non-base64, got %v", err)
	}
	short := base64.StdEncoding.EncodeToString(make([]byte, 16))
	if _, err := NewEnvelope(short); !errors.Is(err, ErrInvalidKey) {
		t.Fatalf("expected ErrInvalidKey for short key, got %v", err)
	}
}

func TestEnvelope_WrongKeyFails(t *testing.T) {
	env1, _ := NewEnvelope(testKey(t))
	blob, _ := env1.Encrypt("secreto")

	raw := make([]byte, KeySize)
	for i := range raw {
		raw[i] = byte(200 - i)
	}
	env2, _ := NewEnvelope(base64.StdEncoding.EncodeToString(raw))
	if _, err := env2.Decrypt(blob); !errors.Is(err, ErrDecrypt) {
		t.Fatalf("expected ErrDecrypt with wrong key, got %v", err)
	}
}

func TestEnvelope_TruncatedBlob(t *testing.T) {
	env, _ := NewEnvelope(testKey(t))
	if _, err := env.Decrypt([]byte("short")); !errors.Is(err, ErrDecrypt) {
		t.Fatalf("expected ErrDecrypt for truncated blob, got %v", err)
	}
}

func TestEnvelope_EmptyPlaintextRejected(t *testing.T) {
	env, _ := NewEnvelope(testKey(t))
	if _, err := env.Encrypt(""); err == nil {
		t.Fatal("expected error for empty plaintext")
	}
}
