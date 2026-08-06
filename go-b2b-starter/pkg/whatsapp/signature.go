package whatsapp

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

const SignaturePrefix = "sha256="

func VerifySignature(secret string, body []byte, signatureHeader string) error {
	if secret == "" {
		return fmt.Errorf("webhook secret is not configured")
	}
	if signatureHeader == "" {
		return fmt.Errorf("signature header is missing")
	}

	payloadSignature := strings.TrimPrefix(signatureHeader, SignaturePrefix)
	if payloadSignature == signatureHeader {
		return fmt.Errorf("invalid signature format: expected 'sha256=' prefix")
	}

	rawHMAC := hmac.New(sha256.New, []byte(secret))
	rawHMAC.Write(body)
	expected := hex.EncodeToString(rawHMAC.Sum(nil))

	if !hmac.Equal([]byte(payloadSignature), []byte(expected)) {
		return fmt.Errorf("signature mismatch")
	}

	return nil
}
