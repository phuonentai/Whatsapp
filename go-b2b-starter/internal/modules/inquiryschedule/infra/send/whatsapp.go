// Package send implements the follow-up outbound seam over the
// circuit-breakered, rate-limited WhatsApp client (pkg/whatsapp), mirroring
// the supplier-inquiries send path (dispatch-time config resolution + mock
// e2e fallback).
package send

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/moasq/go-b2b-starter/internal/modules/inquiryschedule/domain"
	whatsappDomain "github.com/moasq/go-b2b-starter/internal/modules/whatsapp/domain"
	"github.com/moasq/go-b2b-starter/pkg/whatsapp"
)

// textSender is the outbound Meta seam (pkg/whatsapp client with breaker).
type textSender interface {
	SendTextMessage(ctx context.Context, accessToken, graphAPIURL, apiVersion, phoneNumberID, to, body string) (string, error)
}

// RateLimiter enforces the per-org send pacing (10 msgs / 10s). The
// production wiring adapts the shared procurement pacer so follow-up sends
// share the same bucket as inquiry sends.
type RateLimiter interface {
	Allow(orgID int32) bool
}

// Options lets tests inject the Meta seam and the limiter.
type Options struct {
	Sender  textSender
	Limiter RateLimiter
}

// WhatsAppSender dispatches follow-up messages through the circuit-breakered
// WhatsApp client respecting the shared per-org rate limit.
type WhatsAppSender struct {
	configs whatsappDomain.ConfigRepository
	sender  textSender
	limiter RateLimiter
}

// NewWhatsAppSender builds the follow-up sender. By default the
// circuit-breakered client and the shared procurement pacer are used; tests
// can override both via Options.
func NewWhatsAppSender(configs whatsappDomain.ConfigRepository, limiter RateLimiter, opts ...Options) *WhatsAppSender {
	sender := textSender(whatsapp.NewClientWithBreaker(5, 30*time.Second, 2))
	if len(opts) > 0 && opts[0].Sender != nil {
		sender = opts[0].Sender
	}
	return &WhatsAppSender{configs: configs, sender: sender, limiter: limiter}
}

// SendFollowUp implements domain.FollowUpSender: resolves the org WhatsApp
// config and sends through the rate-limited, circuit-breakered client,
// returning the provider message id.
func (s *WhatsAppSender) SendFollowUp(ctx context.Context, orgID int32, send *domain.FollowUpSend) (string, error) {
	if s.limiter != nil && !s.limiter.Allow(orgID) {
		return "", fmt.Errorf("follow-up rate limit (10 msgs / 10s) exceeded for org %d", orgID)
	}
	config, err := s.configs.GetByOrganizationID(ctx, orgID)
	if err != nil {
		return "", fmt.Errorf("whatsapp_not_configured: %w", err)
	}
	if !config.IsActive {
		return "", fmt.Errorf("whatsapp_not_configured: config inactive")
	}
	if config.AccessToken == "" {
		return "", fmt.Errorf("whatsapp_no_access_token: access token missing")
	}
	apiVersion := config.APIVersion
	if apiVersion == "" {
		apiVersion = "v21.0"
	}
	graphAPIURL := config.GraphAPIURL
	if graphAPIURL == "" {
		graphAPIURL = "https://graph.facebook.com"
	}
	if os.Getenv("AUTH_MOCK_ENABLED") == "true" {
		// Mock-auth e2e mode: never call the real Meta Graph API.
		return fmt.Sprintf("wamid.mock.followup.%d", time.Now().UnixNano()), nil
	}
	return s.sender.SendTextMessage(ctx, config.AccessToken, graphAPIURL, apiVersion, config.PhoneNumberID, send.ContactPhone, send.Message)
}
