package polar

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSecret = "whsec_test_secret_key_12345678901234567890"

func signPayload(payload []byte, msgID, timestamp string, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(fmt.Sprintf("%s.%s.%s", msgID, timestamp, payload)))
	return "v1," + base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func TestVerifyWebhookSignature_Valid(t *testing.T) {
	payload := []byte(`{"type":"subscription.created","data":{}}`)
	msgID := "msg_test_1"
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	sig := signPayload(payload, msgID, timestamp, testSecret)

	err := VerifyWebhookSignature(payload, msgID, timestamp, sig, testSecret)
	require.NoError(t, err)
}

func TestVerifyWebhookSignature_TamperedBody(t *testing.T) {
	payload := []byte(`{"type":"subscription.created","data":{}}`)
	msgID := "msg_test_1"
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	sig := signPayload(payload, msgID, timestamp, testSecret)

	tampered := []byte(`{"type":"subscription.created","data":{"evil":true}}`)
	err := VerifyWebhookSignature(tampered, msgID, timestamp, sig, testSecret)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "signature mismatch")
}

func TestVerifyWebhookSignature_ExpiredTimestamp(t *testing.T) {
	payload := []byte(`{"type":"subscription.created","data":{}}`)
	msgID := "msg_test_1"
	old := strconv.FormatInt(time.Now().Add(-10*time.Minute).Unix(), 10)
	sig := signPayload(payload, msgID, old, testSecret)

	err := VerifyWebhookSignature(payload, msgID, old, sig, testSecret)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tolerance")
}

func TestVerifyWebhookSignature_MissingHeaders(t *testing.T) {
	payload := []byte(`{"type":"subscription.created","data":{}}`)
	ts := strconv.FormatInt(time.Now().Unix(), 10)

	err := VerifyWebhookSignature(payload, "", ts, "v1,abc", testSecret)
	require.Error(t, err)

	err = VerifyWebhookSignature(payload, "msg_1", ts, "", testSecret)
	require.Error(t, err)

	err = VerifyWebhookSignature(payload, "msg_1", "", "v1,abc", testSecret)
	require.Error(t, err)

	err = VerifyWebhookSignature(payload, "msg_1", ts, "v1,abc", "")
	require.Error(t, err)
}

func TestVerifyWebhookSignature_UnknownVersionIgnored(t *testing.T) {
	payload := []byte(`{"type":"subscription.created","data":{}}`)
	msgID := "msg_test_1"
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	validSig := signPayload(payload, msgID, timestamp, testSecret)

	err := VerifyWebhookSignature(payload, msgID, timestamp, "v2,"+validSig[len("v1,"):], testSecret)
	require.Error(t, err)
}
