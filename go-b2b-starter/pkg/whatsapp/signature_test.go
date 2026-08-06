package whatsapp

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestVerifySignature(t *testing.T) {
	secret := "test-secret-123"
	body := []byte(`{"test": "payload"}`)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	validSignature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	tests := []struct {
		name            string
		secret          string
		body            []byte
		signatureHeader string
		wantErr         bool
	}{
		{
			name:            "valid signature",
			secret:          secret,
			body:            body,
			signatureHeader: validSignature,
			wantErr:         false,
		},
		{
			name:            "empty secret",
			secret:          "",
			body:            body,
			signatureHeader: validSignature,
			wantErr:         true,
		},
		{
			name:            "missing signature header",
			secret:          secret,
			body:            body,
			signatureHeader: "",
			wantErr:         true,
		},
		{
			name:            "invalid signature format (no sha256= prefix)",
			secret:          secret,
			body:            body,
			signatureHeader: "invalid",
			wantErr:         true,
		},
		{
			name:            "tampered body",
			secret:          secret,
			body:            []byte(`{"test": "tampered"}`),
			signatureHeader: validSignature,
			wantErr:         true,
		},
		{
			name:            "wrong secret",
			secret:          "wrong-secret",
			body:            body,
			signatureHeader: validSignature,
			wantErr:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := VerifySignature(tt.secret, tt.body, tt.signatureHeader)
			if (err != nil) != tt.wantErr {
				t.Errorf("VerifySignature() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}
