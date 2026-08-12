package mercadopago

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testWebhookSecret = "test_mp_webhook_secret"

func TestParseSubscriptionEventData_ExtractsNumericInvoiceCount(t *testing.T) {
	data, err := ParseSubscriptionEventData(json.RawMessage(`{
		"id": "pre-1",
		"external_reference": "org_1",
		"status": "authorized",
		"metadata": {"invoice_count_max": 50}
	}`))
	require.NoError(t, err)
	assert.Equal(t, "50", data.ProductMetadata["invoice_count"])
}

func TestParseSubscriptionEventData_ExtractsStringInvoiceCount(t *testing.T) {
	data, err := ParseSubscriptionEventData(json.RawMessage(`{
		"id": "pre-1",
		"external_reference": "org_1",
		"status": "authorized",
		"metadata": {"invoice_count_max": "50"}
	}`))
	require.NoError(t, err)
	assert.Equal(t, "50", data.ProductMetadata["invoice_count"])
}

func TestParseSubscriptionEventData_NoQuotaMetadata(t *testing.T) {
	data, err := ParseSubscriptionEventData(json.RawMessage(`{
		"id": "pre-1",
		"external_reference": "org_1",
		"status": "authorized"
	}`))
	require.NoError(t, err)
	assert.Empty(t, data.ProductMetadata)
}

func TestParseSubscriptionEventData_KeepsDerivedPeriodBounds(t *testing.T) {
	data, err := ParseSubscriptionEventData(json.RawMessage(`{
		"id": "pre-1",
		"external_reference": "org_1",
		"status": "authorized",
		"auto_recurring": {"start_date": "2026-01-01T00:00:00Z", "end_date": "2027-01-01T00:00:00Z"},
		"metadata": {"invoice_count_max": "25"}
	}`))
	require.NoError(t, err)
	assert.Equal(t, "25", data.ProductMetadata["invoice_count"])
	assert.False(t, data.CurrentPeriodStart.IsZero())
	assert.False(t, data.CurrentPeriodEnd.IsZero())
}

func TestParsePaymentEventData_NumericPaymentID(t *testing.T) {
	data, err := ParsePaymentEventData(json.RawMessage(`{"id": 456, "status": "approved"}`))
	require.NoError(t, err)
	assert.Equal(t, "456", data.ID)
	assert.Equal(t, "succeeded", data.Status)
}

func TestParsePaymentEventData_StringPaymentID(t *testing.T) {
	data, err := ParsePaymentEventData(json.RawMessage(`{"id": "pay-123", "status": "approved"}`))
	require.NoError(t, err)
	assert.Equal(t, "pay-123", data.ID)
}

func TestParsePaymentEventData_MissingID(t *testing.T) {
	data, err := ParsePaymentEventData(json.RawMessage(`{"status": "approved"}`))
	require.NoError(t, err)
	assert.Equal(t, "", data.ID)
}

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

func TestVerifyWebhookSignature_RawHeaderFallbackRejected(t *testing.T) {
	// A correctly-computed raw HMAC without the ts=,v1= envelope must be
	// rejected: the verifier no longer accepts the legacy raw-header fallback.
	payload := []byte(`{"type":"subscription_authorized"}`)
	mac := hmac.New(sha256.New, []byte(testWebhookSecret))
	mac.Write(payload)

	err := VerifyWebhookSignature(payload, hex.EncodeToString(mac.Sum(nil)), testWebhookSecret)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing ts parameter")
}

func TestVerifyWebhookSignature_ReplayRejected(t *testing.T) {
	payload := []byte(`{"id":"notif-123","request_id":"req-456","type":"subscription_authorized","data":{}}`)
	// Signature is cryptographically valid but its timestamp is outside the
	// 5-minute freshness window — replay must be rejected.
	ts := fmt.Sprintf("%d", time.Now().Add(-10*time.Minute).Unix())
	expectedMAC := hmac.New(sha256.New, []byte(testWebhookSecret))
	expectedMAC.Write([]byte(fmt.Sprintf("id:notif-123;request-id:req-456;ts:%s;%s", ts, payload)))
	sig := fmt.Sprintf("ts=%s,v1=%s", ts, hex.EncodeToString(expectedMAC.Sum(nil)))

	err := VerifyWebhookSignature(payload, sig, testWebhookSecret)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "freshness window")
}

func TestVerifyWebhookSignature_FutureTimestampRejected(t *testing.T) {
	payload := []byte(`{"id":"notif-123","request_id":"req-456","type":"subscription_authorized","data":{}}`)
	ts := fmt.Sprintf("%d", time.Now().Add(10*time.Minute).Unix())
	expectedMAC := hmac.New(sha256.New, []byte(testWebhookSecret))
	expectedMAC.Write([]byte(fmt.Sprintf("id:notif-123;request-id:req-456;ts:%s;%s", ts, payload)))
	sig := fmt.Sprintf("ts=%s,v1=%s", ts, hex.EncodeToString(expectedMAC.Sum(nil)))

	err := VerifyWebhookSignature(payload, sig, testWebhookSecret)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "freshness window")
}

func TestVerifyWebhookSignature_MalformedHeaderRejected(t *testing.T) {
	payload := []byte(`{"id":"notif-123"}`)

	// Missing v1 parameter.
	err := VerifyWebhookSignature(payload, "ts=1234567890", testWebhookSecret)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing v1 parameter")

	// Missing ts parameter.
	err = VerifyWebhookSignature(payload, "v1=deadbeef", testWebhookSecret)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing ts parameter")

	// Non-numeric timestamp.
	err = VerifyWebhookSignature(payload, "ts=not-a-number,v1=deadbeef", testWebhookSecret)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid signature timestamp")

	// Garbage header that is not in ts=,v1= form at all.
	err = VerifyWebhookSignature(payload, "garbage-not-hex", testWebhookSecret)
	require.Error(t, err)
}

func TestMapMPStatus_PendingStaysDistinctNotNone(t *testing.T) {
	// MP "pending" (a trial/billing state) must not collapse to "none" or
	// "active": it stays a distinct status that the status provider resolves
	// through the unknown/inactive path, triggering the paywall lazy guard.
	assert.Equal(t, "pending", MapMPStatus("pending"))
	assert.NotEqual(t, "none", MapMPStatus("pending"))
	assert.NotEqual(t, "active", MapMPStatus("pending"))
}

func TestMapMPStatus_KnownAndRawFallthrough(t *testing.T) {
	assert.Equal(t, "active", MapMPStatus("authorized"))
	assert.Equal(t, "active", MapMPStatus("approved"))
	assert.Equal(t, "past_due", MapMPStatus("paused"))
	assert.Equal(t, "canceled", MapMPStatus("cancelled"))
	// Unmapped raw statuses pass through untouched (the status provider keeps
	// them distinct as unknown-inactive rather than "none").
	assert.Equal(t, "in_process", MapMPStatus("in_process"))
}

