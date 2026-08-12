package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	whatsappDomain "github.com/moasq/go-b2b-starter/internal/modules/whatsapp/domain"
	"github.com/moasq/go-b2b-starter/internal/modules/whatsapp/infra/graphapi"
)

// fakeTemplateRepo is an in-memory TemplateRepository.
type fakeTemplateRepo struct {
	whatsappDomain.TemplateRepository
	templates []*whatsappDomain.Template
	nextID    int64
}

func newFakeTemplateRepo() *fakeTemplateRepo {
	return &fakeTemplateRepo{nextID: 1}
}

func (f *fakeTemplateRepo) Create(ctx context.Context, t *whatsappDomain.Template) (*whatsappDomain.Template, error) {
	for _, existing := range f.templates {
		if existing.OrganizationID == t.OrganizationID && existing.Name == t.Name && existing.Language == t.Language {
			return nil, whatsappDomain.ErrTemplateNameConflict
		}
	}
	t.ID = f.nextID
	f.nextID++
	f.templates = append(f.templates, t)
	return t, nil
}

func (f *fakeTemplateRepo) GetByID(ctx context.Context, orgID int32, id int64) (*whatsappDomain.Template, error) {
	for _, t := range f.templates {
		if t.ID == id && t.OrganizationID == orgID {
			return t, nil
		}
	}
	return nil, whatsappDomain.ErrTemplateNotFound
}

func (f *fakeTemplateRepo) ListByOrg(ctx context.Context, orgID int32) ([]*whatsappDomain.Template, error) {
	var out []*whatsappDomain.Template
	for _, t := range f.templates {
		if t.OrganizationID == orgID {
			out = append(out, t)
		}
	}
	return out, nil
}

func (f *fakeTemplateRepo) Update(ctx context.Context, t *whatsappDomain.Template) (*whatsappDomain.Template, error) {
	for i, existing := range f.templates {
		if existing.ID == t.ID && existing.OrganizationID == t.OrganizationID {
			f.templates[i] = t
			return t, nil
		}
	}
	return nil, whatsappDomain.ErrTemplateNotFound
}

func (f *fakeTemplateRepo) UpdateStatus(ctx context.Context, orgID int32, id int64, status whatsappDomain.TemplateStatus, metaTemplateID, rejectionReason *string) (*whatsappDomain.Template, error) {
	for i, t := range f.templates {
		if t.ID == id && t.OrganizationID == orgID {
			if t.Status == status {
				return nil, nil // transaction-isolated no-op
			}
			t.Status = status
			if metaTemplateID != nil {
				t.MetaTemplateID = metaTemplateID
			}
			t.RejectionReason = rejectionReason
			if status == whatsappDomain.TemplateStatusApproved {
				t.IsActive = true
			}
			if status == whatsappDomain.TemplateStatusRejected {
				t.IsActive = false
			}
			f.templates[i] = t
			return t, nil
		}
	}
	return nil, whatsappDomain.ErrTemplateNotFound
}

func (f *fakeTemplateRepo) Delete(ctx context.Context, orgID int32, id int64) error {
	for i, t := range f.templates {
		if t.ID == id && t.OrganizationID == orgID {
			f.templates = append(f.templates[:i], f.templates[i+1:]...)
			return nil
		}
	}
	return whatsappDomain.ErrTemplateNotFound
}

// fakeMetaClient is a mock graphapi.Client for template calls.
type fakeMetaClient struct {
	graphapi.Client
	submitCalls     int
	submitErr       error
	submitResultID  string
	getStatusCalls  int
	getStatusErr    error
	getStatusResult string
}

func (f *fakeMetaClient) SubmitTemplate(ctx context.Context, accessToken, graphAPIURL, apiVersion, phoneNumberID, name, language, category, body string) (string, error) {
	f.submitCalls++
	if f.submitErr != nil {
		return "", f.submitErr
	}
	if f.submitResultID == "" {
		return "meta-1", nil
	}
	return f.submitResultID, nil
}

func (f *fakeMetaClient) GetTemplateStatus(ctx context.Context, accessToken, graphAPIURL, apiVersion, phoneNumberID, metaTemplateID string) (string, error) {
	f.getStatusCalls++
	if f.getStatusErr != nil {
		return "", f.getStatusErr
	}
	return f.getStatusResult, nil
}

func newTemplateTestService(repo *fakeTemplateRepo, meta *fakeMetaClient, cfgRepo whatsappDomain.ConfigRepository) TemplateService {
	return NewTemplateService(repo, cfgRepo, meta, noopLogger{})
}

func seedDraft(repo *fakeTemplateRepo, orgID int32, name string) *whatsappDomain.Template {
	t, err := repo.Create(context.Background(), &whatsappDomain.Template{
		OrganizationID: orgID,
		Name:           name,
		Category:       "UTILITY",
		Language:       "es",
		Body:           "Hola {{1}}, tu pedido {{2}} fue confirmado.",
		ParamCount:     2,
		Status:         whatsappDomain.TemplateStatusDraft,
		IsActive:       true,
	})
	if err != nil {
		panic(err)
	}
	return t
}

func cfgRepoWith(config *whatsappDomain.WhatsAppConfig) whatsappDomain.ConfigRepository {
	repo := newFakeConfigRepo()
	repo.configs[config.OrganizationID] = config
	return repo
}

func TestCreateTemplate_ValidationSpanishMessages(t *testing.T) {
	repo := newFakeTemplateRepo()
	svc := newTemplateTestService(repo, &fakeMetaClient{}, newFakeConfigRepo())

	_, err := svc.CreateTemplate(context.Background(), 42, &TemplateInput{Name: "", Category: "UTILITY", Language: "es", Body: "x"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "El nombre de la plantilla es obligatorio")

	_, err = svc.CreateTemplate(context.Background(), 42, &TemplateInput{Name: "n", Category: "", Language: "es", Body: "x"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "La categoría de la plantilla es obligatoria")

	_, err = svc.CreateTemplate(context.Background(), 42, &TemplateInput{Name: "n", Category: "UTILITY", Language: "", Body: "x"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "El idioma de la plantilla es obligatorio")

	_, err = svc.CreateTemplate(context.Background(), 42, &TemplateInput{Name: "n", Category: "UTILITY", Language: "es", Body: ""})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "El cuerpo de la plantilla es obligatorio")
}

func TestCreateTemplate_ComputesParamCount(t *testing.T) {
	repo := newFakeTemplateRepo()
	svc := newTemplateTestService(repo, &fakeMetaClient{}, newFakeConfigRepo())

	created, err := svc.CreateTemplate(context.Background(), 42, &TemplateInput{
		Name: "confirmacion_pedido", Category: "UTILITY", Language: "es",
		Body: "Hola {{1}}, tu pedido {{2}} fue confirmado.",
	})
	require.NoError(t, err)
	assert.Equal(t, whatsappDomain.TemplateStatusDraft, created.Status)
	assert.Equal(t, 2, created.ParamCount)
	assert.True(t, created.IsActive)
}

func TestCreateTemplate_DuplicateNameLanguageConflict(t *testing.T) {
	repo := newFakeTemplateRepo()
	svc := newTemplateTestService(repo, &fakeMetaClient{}, newFakeConfigRepo())
	seedDraft(repo, 42, "confirmacion_pedido")

	_, err := svc.CreateTemplate(context.Background(), 42, &TemplateInput{
		Name: "confirmacion_pedido", Category: "UTILITY", Language: "es", Body: "otra",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, whatsappDomain.ErrTemplateNameConflict)
}

func TestListTemplates_OrgScoped(t *testing.T) {
	repo := newFakeTemplateRepo()
	svc := newTemplateTestService(repo, &fakeMetaClient{}, newFakeConfigRepo())
	seedDraft(repo, 42, "a")
	seedDraft(repo, 42, "b")
	seedDraft(repo, 7, "c")

	list, err := svc.ListTemplates(context.Background(), 42)
	require.NoError(t, err)
	require.Len(t, list, 2)
	for _, tmpl := range list {
		assert.Equal(t, int32(42), tmpl.OrganizationID)
	}
}

func TestUpdateTemplate_DraftOnlyGuard(t *testing.T) {
	repo := newFakeTemplateRepo()
	svc := newTemplateTestService(repo, &fakeMetaClient{}, newFakeConfigRepo())
	tmpl := seedDraft(repo, 42, "confirmacion_pedido")

	// Submitted templates cannot be edited.
	_, err := repo.UpdateStatus(context.Background(), 42, tmpl.ID, whatsappDomain.TemplateStatusSubmitted, nil, nil)
	require.NoError(t, err)
	_, err = svc.UpdateTemplate(context.Background(), 42, tmpl.ID, &TemplateInput{
		Name: "renamed", Category: "UTILITY", Language: "es", Body: "nuevo",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, whatsappDomain.ErrTemplateNotDraft)

	// Draft update recomputes param_count.
	draft := seedDraft(repo, 42, "otra")
	updated, err := svc.UpdateTemplate(context.Background(), 42, draft.ID, &TemplateInput{
		Name: "otra", Category: "UTILITY", Language: "es", Body: "Hola {{1}}",
	})
	require.NoError(t, err)
	assert.Equal(t, 1, updated.ParamCount)
}

func TestDeleteTemplate_DraftOnlyGuard(t *testing.T) {
	repo := newFakeTemplateRepo()
	svc := newTemplateTestService(repo, &fakeMetaClient{}, newFakeConfigRepo())
	tmpl := seedDraft(repo, 42, "a")

	_, err := repo.UpdateStatus(context.Background(), 42, tmpl.ID, whatsappDomain.TemplateStatusApproved, nil, nil)
	require.NoError(t, err)
	err = svc.DeleteTemplate(context.Background(), 42, tmpl.ID)
	require.Error(t, err)
	assert.ErrorIs(t, err, whatsappDomain.ErrTemplateNotDraft)

	// Still exists (pause instead of delete).
	_, err = repo.GetByID(context.Background(), 42, tmpl.ID)
	require.NoError(t, err)
}

func TestSubmitTemplate_SuccessStoresMetaID(t *testing.T) {
	repo := newFakeTemplateRepo()
	meta := &fakeMetaClient{submitResultID: "meta-1045559864261146"}
	svc := newTemplateTestService(repo, meta, cfgRepoWith(&whatsappDomain.WhatsAppConfig{
		OrganizationID: 42, PhoneNumberID: "12345", AccessToken: "tok", IsActive: true,
	}))
	tmpl := seedDraft(repo, 42, "confirmacion_pedido")

	updated, err := svc.SubmitTemplate(context.Background(), 42, tmpl.ID)
	require.NoError(t, err)
	assert.Equal(t, whatsappDomain.TemplateStatusSubmitted, updated.Status)
	require.NotNil(t, updated.MetaTemplateID)
	assert.Equal(t, "meta-1045559864261146", *updated.MetaTemplateID)
	assert.Equal(t, 1, meta.submitCalls)
}

func TestSubmitTemplate_IdempotentNoSecondMetaCall(t *testing.T) {
	repo := newFakeTemplateRepo()
	meta := &fakeMetaClient{}
	svc := newTemplateTestService(repo, meta, cfgRepoWith(&whatsappDomain.WhatsAppConfig{
		OrganizationID: 42, PhoneNumberID: "12345", AccessToken: "tok", IsActive: true,
	}))
	tmpl := seedDraft(repo, 42, "confirmacion_pedido")
	_, err := repo.UpdateStatus(context.Background(), 42, tmpl.ID, whatsappDomain.TemplateStatusSubmitted, ptr("meta-1"), nil)
	require.NoError(t, err)

	updated, err := svc.SubmitTemplate(context.Background(), 42, tmpl.ID)
	require.NoError(t, err)
	assert.Equal(t, whatsappDomain.TemplateStatusSubmitted, updated.Status)
	assert.Equal(t, 0, meta.submitCalls, "already-submitted template must not call Meta again")
}

func TestSubmitTemplate_ConfigMissingError(t *testing.T) {
	repo := newFakeTemplateRepo()
	svc := newTemplateTestService(repo, &fakeMetaClient{}, newFakeConfigRepo())
	tmpl := seedDraft(repo, 42, "a")

	_, err := svc.SubmitTemplate(context.Background(), 42, tmpl.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "whatsapp_not_configured")
}

func TestSubmitTemplate_MetaErrorLeavesLocalStateUntouched(t *testing.T) {
	repo := newFakeTemplateRepo()
	meta := &fakeMetaClient{submitErr: errors.New("graph api error (code 100): bad")}
	svc := newTemplateTestService(repo, meta, cfgRepoWith(&whatsappDomain.WhatsAppConfig{
		OrganizationID: 42, PhoneNumberID: "12345", AccessToken: "tok", IsActive: true,
	}))
	tmpl := seedDraft(repo, 42, "a")

	_, err := svc.SubmitTemplate(context.Background(), 42, tmpl.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "whatsapp_api_error")

	// Local state unchanged: still draft, no meta id.
	loaded, err := repo.GetByID(context.Background(), 42, tmpl.ID)
	require.NoError(t, err)
	assert.Equal(t, whatsappDomain.TemplateStatusDraft, loaded.Status)
	assert.Nil(t, loaded.MetaTemplateID)
}

func TestRefreshTemplateStatus_ReconcilesApproved(t *testing.T) {
	repo := newFakeTemplateRepo()
	meta := &fakeMetaClient{getStatusResult: "APPROVED"}
	svc := newTemplateTestService(repo, meta, cfgRepoWith(&whatsappDomain.WhatsAppConfig{
		OrganizationID: 42, PhoneNumberID: "12345", AccessToken: "tok", IsActive: true,
	}))
	tmpl := seedDraft(repo, 42, "a")
	_, err := repo.UpdateStatus(context.Background(), 42, tmpl.ID, whatsappDomain.TemplateStatusSubmitted, ptr("meta-1"), nil)
	require.NoError(t, err)

	updated, err := svc.RefreshTemplateStatus(context.Background(), 42, tmpl.ID)
	require.NoError(t, err)
	assert.Equal(t, whatsappDomain.TemplateStatusApproved, updated.Status)
	assert.True(t, updated.IsActive)
	assert.Equal(t, 1, meta.getStatusCalls)
}

func TestRefreshTemplateStatus_NotFoundAtMeta(t *testing.T) {
	repo := newFakeTemplateRepo()
	meta := &fakeMetaClient{getStatusErr: &graphapi.GraphError{Code: 100, Message: "not found"}}
	svc := newTemplateTestService(repo, meta, cfgRepoWith(&whatsappDomain.WhatsAppConfig{
		OrganizationID: 42, PhoneNumberID: "12345", AccessToken: "tok", IsActive: true,
	}))
	tmpl := seedDraft(repo, 42, "a")
	_, err := repo.UpdateStatus(context.Background(), 42, tmpl.ID, whatsappDomain.TemplateStatusSubmitted, ptr("meta-1"), nil)
	require.NoError(t, err)

	_, err = svc.RefreshTemplateStatus(context.Background(), 42, tmpl.ID)
	require.Error(t, err)
	assert.ErrorIs(t, err, whatsappDomain.ErrTemplateNotFoundAtMeta)
}

func ptr(s string) *string { return &s }
