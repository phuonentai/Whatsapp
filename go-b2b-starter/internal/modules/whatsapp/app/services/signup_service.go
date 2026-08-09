package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	ticketsServices "github.com/moasq/go-b2b-starter/internal/modules/tickets/app/services"
	ticketsDomain "github.com/moasq/go-b2b-starter/internal/modules/tickets/domain"
	whatsappDomain "github.com/moasq/go-b2b-starter/internal/modules/whatsapp/domain"
	"github.com/moasq/go-b2b-starter/internal/modules/whatsapp/infra/graphapi"
	"github.com/moasq/go-b2b-starter/internal/platform/logger"
	loggerdomain "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
)

const (
	signupStepExchangeCode          = "exchange_code"
	signupStepResolveBusiness       = "resolve_business"
	signupStepResolveWABA           = "resolve_waba"
	signupStepCreateSystemUser      = "create_system_user"
	signupStepSubscribeWABA         = "subscribe_waba"
	signupStepRegisterSubscriptions = "register_subscriptions"
	signupStepWriteConfig           = "write_config"
	signupStepTestEcho              = "test_echo"

	signupMaxRetries = 3
)

type SignupService interface {
	MetaConfig(ctx context.Context) (*whatsappDomain.MetaConfig, error)
	Exchange(ctx context.Context, orgID int32, code string, actorMemberID string) (*whatsappDomain.SignupResult, error)
	Status(ctx context.Context, orgID int32) (*whatsappDomain.SignupFlow, error)
}

type signupService struct {
	configRepo    whatsappDomain.ConfigRepository
	signupRepo    whatsappDomain.SignupFlowRepository
	graphClient   graphapi.Client
	metaCfg       graphapi.MetaConfig
	appID         string
	callbackURL   string
	ticketService ticketsServices.TicketService
	logger        logger.Logger
	backoff       func(int) time.Duration
}

func defaultBackoff(attempt int) time.Duration {
	return time.Duration(1<<attempt) * time.Second
}

func NewSignupService(
	configRepo whatsappDomain.ConfigRepository,
	signupRepo whatsappDomain.SignupFlowRepository,
	graphClient graphapi.Client,
	metaCfg graphapi.MetaConfig,
	appID string,
	callbackURL string,
	ticketService ticketsServices.TicketService,
	log logger.Logger,
) SignupService {
	return &signupService{
		configRepo:    configRepo,
		signupRepo:    signupRepo,
		graphClient:   graphClient,
		metaCfg:       metaCfg,
		appID:         appID,
		callbackURL:   callbackURL,
		ticketService: ticketService,
		logger:        log,
		backoff:       defaultBackoff,
	}
}

func (s *signupService) MetaConfig(ctx context.Context) (*whatsappDomain.MetaConfig, error) {
	return &whatsappDomain.MetaConfig{
		AppID:       s.metaCfg.AppID,
		ConfigID:    s.metaCfg.ConfigID,
		RedirectURI: s.metaCfg.RedirectURI,
	}, nil
}

func (s *signupService) Status(ctx context.Context, orgID int32) (*whatsappDomain.SignupFlow, error) {
	return s.signupRepo.GetByOrganization(ctx, orgID)
}

func (s *signupService) Exchange(ctx context.Context, orgID int32, code string, actorMemberID string) (*whatsappDomain.SignupResult, error) {
	existing, err := s.signupRepo.GetByOrganization(ctx, orgID)
	if err == nil {
		switch existing.Status {
		case whatsappDomain.SignupStatusConnected:
			return nil, whatsappDomain.ErrSignupAlreadyConnected
		case whatsappDomain.SignupStatusExchanging, whatsappDomain.SignupStatusRegistering, whatsappDomain.SignupStatusVerifying:
			return nil, whatsappDomain.ErrSignupInProgress
		}
	}
	if code == "" {
		return nil, whatsappDomain.ErrSignupCodeRequired
	}

	metadata := map[string]interface{}{}
	flow, err := s.signupRepo.Upsert(ctx, &whatsappDomain.SignupFlow{
		OrganizationID: orgID,
		Status:         whatsappDomain.SignupStatusExchanging,
		Step:           signupStepExchangeCode,
		RetryCount:     0,
		Metadata:       metadata,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start signup flow: %w", err)
	}

	userToken, err := s.graphClient.ExchangeCode(ctx, code)
	if err != nil {
		return nil, s.failFlow(ctx, flow, "token_exchange_failed", err, actorMemberID)
	}

	flow, err = s.signupRepo.UpdateStatus(ctx, orgID, whatsappDomain.SignupStatusRegistering, signupStepResolveBusiness, "", 0, flow.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to advance signup flow: %w", err)
	}

	businessID, err := withRetries(ctx, s.backoff, func() (string, error) {
		return s.graphClient.ResolveBusiness(ctx, userToken)
	})
	if err != nil {
		return nil, s.failFlow(ctx, flow, "business_resolution_failed", err, actorMemberID)
	}

	flow, err = s.signupRepo.UpdateStatus(ctx, orgID, whatsappDomain.SignupStatusRegistering, signupStepResolveWABA, "", 0, flow.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to advance signup flow: %w", err)
	}

	wabas, err := withRetries(ctx, s.backoff, func() ([]graphapi.WABAInfo, error) {
		return s.graphClient.ResolveWABAAndNumbers(ctx, userToken, businessID)
	})
	if err != nil {
		return nil, s.failFlow(ctx, flow, "waba_resolution_failed", err, actorMemberID)
	}
	if len(wabas) == 0 || len(wabas[0].PhoneNumbers) == 0 {
		return nil, s.failFlow(ctx, flow, "no_waba_or_number", fmt.Errorf("no WABA or phone number resolved"), actorMemberID)
	}
	waba := wabas[0]
	phone := waba.PhoneNumbers[0]
	coexistence := phone.Certificate != ""
	flow.Metadata["waba_id"] = waba.ID
	flow.Metadata["phone_number_id"] = phone.ID
	flow.Metadata["coexistence"] = coexistence

	flow, err = s.signupRepo.UpdateStatus(ctx, orgID, whatsappDomain.SignupStatusRegistering, signupStepCreateSystemUser, "", 0, flow.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to advance signup flow: %w", err)
	}

	sysToken, err := withRetries(ctx, s.backoff, func() (string, error) {
		return s.graphClient.CreateSystemUser(ctx, userToken, businessID, s.appID)
	})
	if err != nil {
		return nil, s.failFlow(ctx, flow, "system_user_failed", err, actorMemberID)
	}

	flow, err = s.signupRepo.UpdateStatus(ctx, orgID, whatsappDomain.SignupStatusRegistering, signupStepSubscribeWABA, "", 0, flow.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to advance signup flow: %w", err)
	}

	if err := withRetriesErr(ctx, s.backoff, func() error {
		return s.graphClient.SubscribeWABA(ctx, userToken, waba.ID)
	}); err != nil {
		return nil, s.failFlow(ctx, flow, "waba_subscribe_failed", err, actorMemberID)
	}

	webhookSecret, err := generateSecret("whsec_")
	if err != nil {
		return nil, s.failFlow(ctx, flow, "secret_generation_failed", err, actorMemberID)
	}
	verifyToken, err := generateSecret("verify_")
	if err != nil {
		return nil, s.failFlow(ctx, flow, "secret_generation_failed", err, actorMemberID)
	}

	flow, err = s.signupRepo.UpdateStatus(ctx, orgID, whatsappDomain.SignupStatusRegistering, signupStepRegisterSubscriptions, "", 0, flow.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to advance signup flow: %w", err)
	}

	if err := withRetriesErr(ctx, s.backoff, func() error {
		return s.graphClient.RegisterAppSubscriptions(ctx, userToken, s.appID, s.callbackURL, verifyToken)
	}); err != nil {
		return nil, s.failFlow(ctx, flow, "subscription_registration_failed", err, actorMemberID)
	}

	flow, err = s.signupRepo.UpdateStatus(ctx, orgID, whatsappDomain.SignupStatusRegistering, signupStepWriteConfig, "", 0, flow.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to advance signup flow: %w", err)
	}

	cfg, err := s.writeConfig(ctx, orgID, &whatsappDomain.WhatsAppConfig{
		PhoneNumberID: phone.ID,
		BusinessPhone: phone.DisplayPhoneNumber,
		WebhookSecret: webhookSecret,
		VerifyToken:   verifyToken,
		AppID:         s.appID,
		WABAID:        waba.ID,
		AccessToken:   sysToken,
		IsActive:      true,
		Metadata:      map[string]interface{}{"coexistence": coexistence, "signup_flow": "embedded"},
	})
	if err != nil {
		return nil, s.failFlow(ctx, flow, "config_write_failed", err, actorMemberID)
	}

	flow, err = s.signupRepo.UpdateStatus(ctx, orgID, whatsappDomain.SignupStatusVerifying, signupStepTestEcho, "", 0, flow.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to advance signup flow: %w", err)
	}

	apiVersion := cfg.APIVersion
	if apiVersion == "" {
		apiVersion = "v21.0"
	}
	graphAPIURL := cfg.GraphAPIURL
	if graphAPIURL == "" {
		graphAPIURL = "https://graph.facebook.com"
	}
	if err := withRetriesErr(ctx, s.backoff, func() error {
		return s.graphClient.SendTestMessage(ctx, sysToken, graphAPIURL, apiVersion, phone.ID, cfg.BusinessPhone)
	}); err != nil {
		return nil, s.failFlow(ctx, flow, "test_echo_failed", err, actorMemberID)
	}

	flow, err = s.signupRepo.UpdateStatus(ctx, orgID, whatsappDomain.SignupStatusConnected, "", "", 0, flow.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to finalize signup flow: %w", err)
	}

	cfg.WebhookSecret = MaskSecret(cfg.WebhookSecret)
	cfg.VerifyToken = MaskSecret(cfg.VerifyToken)
	cfg.AccessToken = MaskSecret(cfg.AccessToken)

	return &whatsappDomain.SignupResult{Status: whatsappDomain.SignupStatusConnected, Config: cfg}, nil
}

// writeConfig creates or updates the org's WhatsApp config with provisioned values.
func (s *signupService) writeConfig(ctx context.Context, orgID int32, input *whatsappDomain.WhatsAppConfig) (*whatsappDomain.WhatsAppConfig, error) {
	existing, err := s.configRepo.GetByOrganizationID(ctx, orgID)
	if err == nil {
		existing.PhoneNumberID = input.PhoneNumberID
		existing.BusinessPhone = input.BusinessPhone
		existing.WebhookSecret = input.WebhookSecret
		existing.VerifyToken = input.VerifyToken
		existing.AppID = input.AppID
		existing.WABAID = input.WABAID
		existing.AccessToken = input.AccessToken
		existing.IsActive = true
		if input.Metadata != nil {
			existing.Metadata = input.Metadata
		}
		return s.configRepo.Update(ctx, existing)
	}

	input.OrganizationID = orgID
	if err := input.Validate(); err != nil {
		return nil, err
	}
	return s.configRepo.Create(ctx, input)
}

// failFlow marks the flow failed, records the error code, creates a HITL ticket
// (best-effort), and returns a SignupFailedError.
func (s *signupService) failFlow(ctx context.Context, flow *whatsappDomain.SignupFlow, errorCode string, cause error, actorMemberID string) error {
	flow.Metadata["error_code"] = errorCode
	flow.RetryCount++
	updated, err := s.signupRepo.UpdateStatus(ctx, flow.OrganizationID, whatsappDomain.SignupStatusFailed, flow.Step, errorCode, flow.RetryCount, flow.Metadata)
	if err != nil {
		s.logger.Error("failed to record signup failure", loggerdomain.Fields{"org_id": flow.OrganizationID, "error": err.Error()})
	} else {
		flow = updated
	}

	s.escalate(ctx, flow, errorCode, cause, actorMemberID)
	return &whatsappDomain.SignupFailedError{Code: errorCode, Err: cause}
}

// escalate creates a high-priority HITL ticket; failures are logged, never fatal.
func (s *signupService) escalate(ctx context.Context, flow *whatsappDomain.SignupFlow, errorCode string, cause error, actorMemberID string) {
	if s.ticketService == nil {
		s.logger.Warn("signup failed without ticket service", loggerdomain.Fields{"org_id": flow.OrganizationID, "error_code": errorCode})
		return
	}
	ticket, err := s.ticketService.Create(ctx, flow.OrganizationID, &ticketsServices.CreateTicketRequest{
		Title:       fmt.Sprintf("WhatsApp signup failed (org %d)", flow.OrganizationID),
		Description: fmt.Sprintf("Embedded signup failed with error_code=%s at step=%s after %d retries. Cause: %v", errorCode, flow.Step, flow.RetryCount, cause),
		Priority:    ticketsDomain.PriorityHigh,
		Tags:        []string{"whatsapp", "signup"},
	}, actorMemberID)
	if err != nil {
		s.logger.Error("failed to create HITL ticket for signup failure", loggerdomain.Fields{
			"org_id": flow.OrganizationID, "error_code": errorCode, "error": err.Error(),
		})
		return
	}
	flow.Metadata["ticket_id"] = ticket.ID
	if _, err := s.signupRepo.UpdateStatus(ctx, flow.OrganizationID, flow.Status, flow.Step, errorCode, flow.RetryCount, flow.Metadata); err != nil {
		s.logger.Warn("failed to record ticket id on signup flow", loggerdomain.Fields{"org_id": flow.OrganizationID, "error": err.Error()})
	}
}

// withRetriesErr runs an error-only fn up to signupMaxRetries times with the given backoff.
func withRetriesErr(ctx context.Context, backoff func(int) time.Duration, fn func() error) error {
	_, err := withRetries(ctx, backoff, func() (struct{}, error) {
		return struct{}{}, fn()
	})
	return err
}

// withRetries runs fn up to signupMaxRetries times with the given backoff.
func withRetries[T any](ctx context.Context, backoff func(int) time.Duration, fn func() (T, error)) (T, error) {
	var zero T
	var err error
	for attempt := 0; attempt < signupMaxRetries; attempt++ {
		zero, err = fn()
		if err == nil {
			return zero, nil
		}
		if attempt < signupMaxRetries-1 {
			delay := backoff(attempt)
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return zero, ctx.Err()
			}
		}
	}
	return zero, err
}

// generateSecret returns a cryptographically random string with the given prefix.
func generateSecret(prefix string) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate secret: %w", err)
	}
	return prefix + base64.RawURLEncoding.EncodeToString(b), nil
}
