package invoicing

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/moasq/go-b2b-starter/internal/modules/invoicing/domain"
	"github.com/moasq/go-b2b-starter/internal/modules/invoicing/infra/siigo"
	loggerDomain "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
)

type stubInvoicingService struct {
	processed [][]byte
	lastErr   error
}

func (s *stubInvoicingService) CreateForDeal(ctx context.Context, orgID, dealID int32) (*domain.Invoice, error) {
	return nil, nil
}

func (s *stubInvoicingService) ProcessWebhookEvent(ctx context.Context, rawBody []byte) error {
	s.processed = append(s.processed, rawBody)
	return s.lastErr
}

func (s *stubInvoicingService) PollPending(ctx context.Context) (int, error) { return 0, nil }

func sign(payload, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

func newTestRouter(svc *stubInvoicingService, secret string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := &Handler{
		invoicingService: svc,
		webhookSecret:    secret,
		logger:           nopLog{},
	}
	r := gin.New()
	r.POST("/api/v1/webhooks/siigo", h.ProcessSiigoWebhook)
	return r
}

type nopLog struct{}

func (nopLog) Debug(msg string, fields ...loggerDomain.Fields) {}
func (nopLog) Info(msg string, fields ...loggerDomain.Fields)  {}
func (nopLog) Warn(msg string, fields ...loggerDomain.Fields)  {}
func (nopLog) Error(msg string, fields ...loggerDomain.Fields) {}
func (nopLog) Fatal(msg string, fields ...loggerDomain.Fields) {}
func (nopLog) WithFields(fields loggerDomain.Fields) loggerDomain.Logger {
	return nopLog{}
}

func TestProcessSiigoWebhook_ValidSignature(t *testing.T) {
	svc := &stubInvoicingService{}
	router := newTestRouter(svc, "whsec_test")

	payload, _ := json.Marshal(map[string]any{"id": "inv-1", "status": "valid"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/siigo", bytes.NewReader(payload))
	req.Header.Set(siigo.WebhookSignatureHeader, sign(string(payload), "whsec_test"))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if len(svc.processed) != 1 {
		t.Fatalf("expected 1 processed event, got %d", len(svc.processed))
	}
}

func TestProcessSiigoWebhook_InvalidSignature(t *testing.T) {
	svc := &stubInvoicingService{}
	router := newTestRouter(svc, "whsec_test")

	payload := []byte(`{"id":"inv-1","status":"valid"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/siigo", bytes.NewReader(payload))
	req.Header.Set(siigo.WebhookSignatureHeader, "deadbeef")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
	if len(svc.processed) != 0 {
		t.Fatal("webhook processed despite invalid signature")
	}
}

func TestProcessSiigoWebhook_MissingSignature(t *testing.T) {
	svc := &stubInvoicingService{}
	router := newTestRouter(svc, "whsec_test")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/siigo", bytes.NewReader([]byte(`{}`)))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for missing signature, got %d", w.Code)
	}
}
