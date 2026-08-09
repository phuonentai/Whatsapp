package siigo

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestVerifyWebhookSignature_Valid(t *testing.T) {
	secret := "whsec_test"
	payload := []byte(`{"id":"inv-1","status":"valid"}`)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	sig := hex.EncodeToString(mac.Sum(nil))

	if err := VerifyWebhookSignature(payload, sig, secret); err != nil {
		t.Fatalf("valid signature rejected: %v", err)
	}
}

func TestVerifyWebhookSignature_Tampered(t *testing.T) {
	secret := "whsec_test"
	payload := []byte(`{"id":"inv-1","status":"valid"}`)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	sig := hex.EncodeToString(mac.Sum(nil))

	if err := VerifyWebhookSignature([]byte(`{"id":"inv-1","status":"invalid"}`), sig, secret); err == nil {
		t.Fatal("tampered payload accepted")
	}
}

func TestVerifyWebhookSignature_MissingSecretOrHeader(t *testing.T) {
	if err := VerifyWebhookSignature([]byte("{}"), "", "whsec"); err == nil {
		t.Fatal("missing signature accepted")
	}
	if err := VerifyWebhookSignature([]byte("{}"), "abc", ""); err == nil {
		t.Fatal("missing secret accepted")
	}
}
