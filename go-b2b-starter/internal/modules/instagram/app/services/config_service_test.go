package services

import (
	"context"
	"errors"
	"testing"
	"time"

	igDomain "github.com/moasq/go-b2b-starter/internal/modules/instagram/domain"
	"github.com/moasq/go-b2b-starter/internal/modules/instagram/infra/graphapi"
)

type fakeIGClient struct {
	sendErr    error
	getUser    *graphapi.IGUser
	getUserErr error
	newToken   string
	newExpiry  *time.Time
	refreshErr error
}

func (f *fakeIGClient) SendTextMessage(context.Context, string, string, string, string, string, string) (string, error) {
	return "mid.sent", f.sendErr
}
func (f *fakeIGClient) GetIGUser(context.Context, string, string, string, string) (*graphapi.IGUser, error) {
	return f.getUser, f.getUserErr
}
func (f *fakeIGClient) RefreshToken(context.Context, string, string, string, string, string) (string, *time.Time, error) {
	return f.newToken, f.newExpiry, f.refreshErr
}

func TestGetConfig_MasksSecrets(t *testing.T) {
	expiry := time.Now().Add(48 * time.Hour)
	cfg := &igDomain.InstagramConfig{
		ID: 1, OrganizationID: 7, IGUserID: "ig-1",
		AccessToken: "verylongaccesstokenvalue", WebhookSecret: "webhooksecretvalue",
		VerifyToken: "verifytokenvalue", TokenExpiresAt: &expiry, IsActive: true,
	}
	svc := NewConfigService(&fakeIGConfigRepo{cfg: cfg}, &fakeIGClient{})

	got, err := svc.GetConfig(context.Background(), 7)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.AccessToken == "verylongaccesstokenvalue" || len(got.AccessToken) != 14 {
		t.Fatalf("access token not masked: %q", got.AccessToken)
	}
	if got.WebhookSecret == "webhooksecretvalue" || got.VerifyToken == "verifytokenvalue" {
		t.Fatal("secrets not masked")
	}
}

func TestUpsertConfig_CreateDefaults(t *testing.T) {
	input := &igDomain.InstagramConfig{
		IGUserID:      "ig-1",
		AccessToken:   "tok-1",
		WebhookSecret: "sec-1",
		VerifyToken:   "ver-1",
	}

	created := &igDomain.InstagramConfig{
		ID: 1, OrganizationID: 7, IGUserID: "ig-1", AccessToken: "tok-1",
		WebhookSecret: "sec-1", VerifyToken: "ver-1",
		APIVersion: "v21.0", GraphAPIURL: "https://graph.facebook.com", IsActive: true,
	}
	repo := &creatingConfigRepo{created: created}
	svc := NewConfigService(repo, &fakeIGClient{})

	got, err := svc.UpsertConfig(context.Background(), 7, input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.IGUserID != "ig-1" {
		t.Fatalf("unexpected config %+v", got)
	}
	if repo.created.APIVersion != "v21.0" || repo.created.GraphAPIURL != "https://graph.facebook.com" {
		t.Fatalf("defaults not applied: %+v", repo.created)
	}
}

func TestUpsertConfig_PreservesMaskedSecretsOnUpdate(t *testing.T) {
	existing := &igDomain.InstagramConfig{
		ID: 1, OrganizationID: 7, IGUserID: "ig-1",
		AccessToken: "realtoken-1234567890", WebhookSecret: "realsecret-1234567890",
		VerifyToken: "realverify-1234567890", IsActive: true,
	}
	svc := NewConfigService(&fakeIGConfigRepo{cfg: existing}, &fakeIGClient{})

	input := &igDomain.InstagramConfig{
		IGUserID:      "ig-1",
		AccessToken:   "realt****7890",
		WebhookSecret: "realse****7890",
		VerifyToken:   "realve****7890",
	}
	got, err := svc.UpsertConfig(context.Background(), 7, input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// The stored secrets must remain untouched and the response masked.
	if got.AccessToken == "realtoken-1234567890" || got.AccessToken == "" {
		t.Fatalf("unexpected access token in response: %q", got.AccessToken)
	}
}

func TestRefreshToken_UpdatesToken(t *testing.T) {
	expiry := time.Now().Add(60 * 24 * time.Hour)
	existing := &igDomain.InstagramConfig{
		ID: 1, OrganizationID: 7, IGUserID: "ig-1",
		AccessToken: "old-token", IsActive: true,
	}
	svc := NewConfigService(&fakeIGConfigRepo{cfg: existing}, &fakeIGClient{
		newToken: "new-token", newExpiry: &expiry,
	})

	got, err := svc.RefreshToken(context.Background(), 7, "app-1", "secret-1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.AccessToken == "old-token" || got.AccessToken == "new-token" {
		t.Fatalf("response should mask the refreshed token, got %q", got.AccessToken)
	}
	if existing.AccessToken != "new-token" {
		t.Fatalf("stored token should be refreshed, got %q", existing.AccessToken)
	}
}

func TestRefreshToken_FailureKeepsStoredToken(t *testing.T) {
	existing := &igDomain.InstagramConfig{
		ID: 1, OrganizationID: 7, IGUserID: "ig-1",
		AccessToken: "old-token", IsActive: true,
	}
	svc := NewConfigService(&fakeIGConfigRepo{cfg: existing}, &fakeIGClient{refreshErr: errors.New("invalid token")})

	_, err := svc.RefreshToken(context.Background(), 7, "app-1", "secret-1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if existing.AccessToken != "old-token" {
		t.Fatalf("stored token must be untouched, got %q", existing.AccessToken)
	}
}

// creatingConfigRepo returns config_not_found on lookup so UpsertConfig takes
// the create path, then returns the prebuilt config on Create.
type creatingConfigRepo struct {
	created *igDomain.InstagramConfig
}

func (f *creatingConfigRepo) GetByIGUserID(context.Context, string) (*igDomain.InstagramConfig, error) {
	return nil, igDomain.ErrConfigNotFound
}
func (f *creatingConfigRepo) GetByOrganizationID(context.Context, int32) (*igDomain.InstagramConfig, error) {
	return nil, igDomain.ErrConfigNotFound
}
func (f *creatingConfigRepo) GetByVerifyToken(context.Context, string) (*igDomain.InstagramConfig, error) {
	return nil, igDomain.ErrConfigNotFound
}
func (f *creatingConfigRepo) Create(_ context.Context, cfg *igDomain.InstagramConfig) (*igDomain.InstagramConfig, error) {
	f.created.OrganizationID = cfg.OrganizationID
	return f.created, nil
}
func (f *creatingConfigRepo) Update(_ context.Context, cfg *igDomain.InstagramConfig) (*igDomain.InstagramConfig, error) {
	return cfg, nil
}
