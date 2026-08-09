package mercadopago

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testWebhookSecret = "test_mp_webhook_secret"

func TestVerifyWebhookSignature_MercadoPagoFormat(t *testing.T) {
	payload := []byte(`{"id":"notif-123","request_id":"req-456","type":"subscription_authorized","data":{}}`)
	ts := fmt.Sprintf("%d", time.Now().Unix())
	expectedMAC := hmac.New(sha256.New, []byte(testWebhookSecret))
	expectedMAC.Write([]byte(fmt.Sprintf("id:notif-123;request-id:req-456;ts:%s;%s", ts, payload)))
	sig := fmt.Sprintf("ts=%s,v1=%s", ts, hex.EncodeToString(expectedMAC.Sum(nil)))

	err := VerifyWebhookSignature(payload, sig, testWebhookSecret)
	require.NoError(t, err)
}

func TestVerifyWebhookSignature_MercadoPagoFormatTampered(t *testing.T) {
	payload := []byte(`{"id":"notif-123","request_id":"req-456","type":"subscription_authorized","data":{}}`)
	ts := fmt.Sprintf("%d", time.Now().Unix())
	expectedMAC := hmac.New(sha256.New, []byte(testWebhookSecret))
	expectedMAC.Write([]byte(fmt.Sprintf("id:notif-123;request-id:req-456;ts:%s;%s", ts, payload)))
	sig := fmt.Sprintf("ts=%s,v1=%s", ts, hex.EncodeToString(expectedMAC.Sum(nil)))

	tampered := []byte(`{"id":"notif-123","request_id":"req-456","type":"subscription_authorized","data":{"evil":true}}`)
	err := VerifyWebhookSignature(tampered, sig, testWebhookSecret)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "signature mismatch")
}

func TestVerifyWebhookSignature_MissingOrInvalidHeader(t *testing.T) {
	payload := []byte(`{"id":"notif-123"}`)

	err := VerifyWebhookSignature(payload, "", testWebhookSecret)
	require.Error(t, err)

	err = VerifyWebhookSignature(payload, "garbage-not-hex", testWebhookSecret)
	require.Error(t, err)

	err = VerifyWebhookSignature(payload, "v1=deadbeef", "")
	require.Error(t, err)
}

func TestVerifyWebhookSignature_LegacyRawHex(t *testing.T) {
	payload := []byte(`{"type":"subscription_authorized"}`)
	mac := hmac.New(sha256.New, []byte(testWebhookSecret))
	mac.Write(payload)

	err := VerifyWebhookSignature(payload, hex.EncodeToString(mac.Sum(nil)), testWebhookSecret)
	require.NoError(t, err)

	err = VerifyWebhookSignature([]byte(`tampered`), hex.EncodeToString(mac.Sum(nil)), testWebhookSecret)
	require.Error(t, err)
}
