package services

import (
	"context"
	"errors"
	"testing"

	crmDomain "github.com/moasq/go-b2b-starter/internal/modules/crm/domain"

	"github.com/moasq/go-b2b-starter/internal/modules/campaigns/domain"
)

// ---------- mocks ----------

type mockSegmentRepo struct {
	segments map[int32]*domain.Segment
	nextID   int32
}

func newMockSegmentRepo() *mockSegmentRepo {
	return &mockSegmentRepo{segments: map[int32]*domain.Segment{}, nextID: 1}
}

func (m *mockSegmentRepo) Create(ctx context.Context, orgID int32, nombre string, spec []domain.Filter, createdBy string) (*domain.Segment, error) {
	s := &domain.Segment{ID: m.nextID, OrganizationID: orgID, Nombre: nombre, FilterSpec: spec, CreatedBy: createdBy}
	m.nextID++
	m.segments[s.ID] = s
	return s, nil
}
func (m *mockSegmentRepo) Update(ctx context.Context, orgID, id int32, nombre string, spec []domain.Filter) (*domain.Segment, error) {
	s, ok := m.segments[id]
	if !ok || s.OrganizationID != orgID {
		return nil, domain.ErrSegmentNotFound
	}
	s.Nombre, s.FilterSpec = nombre, spec
	return s, nil
}
func (m *mockSegmentRepo) Delete(ctx context.Context, orgID, id int32) error {
	if _, ok := m.segments[id]; !ok {
		return domain.ErrSegmentNotFound
	}
	delete(m.segments, id)
	return nil
}
func (m *mockSegmentRepo) Get(ctx context.Context, orgID, id int32) (*domain.Segment, error) {
	s, ok := m.segments[id]
	if !ok || s.OrganizationID != orgID {
		return nil, domain.ErrSegmentNotFound
	}
	return s, nil
}
func (m *mockSegmentRepo) List(ctx context.Context, orgID int32) ([]*domain.Segment, error) {
	out := make([]*domain.Segment, 0)
	for _, s := range m.segments {
		if s.OrganizationID == orgID {
			out = append(out, s)
		}
	}
	return out, nil
}

type mockCampaignRepo struct {
	campaigns  map[int32]*domain.Campaign
	recipients map[int32][]*domain.CampaignRecipient
	nextID     int32
}

func newMockCampaignRepo() *mockCampaignRepo {
	return &mockCampaignRepo{campaigns: map[int32]*domain.Campaign{}, recipients: map[int32][]*domain.CampaignRecipient{}, nextID: 1}
}

func (m *mockCampaignRepo) Create(ctx context.Context, orgID int32, nombre string, segmentID int32, mensaje string, createdBy string) (*domain.Campaign, error) {
	c := &domain.Campaign{ID: m.nextID, OrganizationID: orgID, Nombre: nombre, SegmentID: segmentID, Status: domain.CampaignDraft, CreatedBy: createdBy}
	if mensaje != "" {
		c.Mensaje = &mensaje
	}
	m.nextID++
	m.campaigns[c.ID] = c
	return c, nil
}
func (m *mockCampaignRepo) Get(ctx context.Context, orgID, id int32) (*domain.Campaign, error) {
	c, ok := m.campaigns[id]
	if !ok || c.OrganizationID != orgID {
		return nil, domain.ErrCampaignNotFound
	}
	return c, nil
}
func (m *mockCampaignRepo) List(ctx context.Context, orgID int32) ([]*domain.Campaign, error) {
	out := make([]*domain.Campaign, 0)
	for _, c := range m.campaigns {
		if c.OrganizationID == orgID {
			out = append(out, c)
		}
	}
	return out, nil
}
func (m *mockCampaignRepo) Launch(ctx context.Context, orgID, id int32, recipientCount int32) (*domain.Campaign, error) {
	c, ok := m.campaigns[id]
	if !ok || c.OrganizationID != orgID {
		return nil, domain.ErrCampaignNotFound
	}
	if c.Status != domain.CampaignDraft {
		return nil, domain.ErrCampaignNotDraft
	}
	c.Status = domain.CampaignReady
	c.RecipientCount = recipientCount
	return c, nil
}
func (m *mockCampaignRepo) SnapshotRecipients(ctx context.Context, campaignID int32, contactIDs []int32) (int64, error) {
	inserted := int64(0)
	seen := map[int32]bool{}
	for _, r := range m.recipients[campaignID] {
		seen[r.ContactID] = true
	}
	for _, id := range contactIDs {
		if seen[id] {
			continue
		}
		seen[id] = true
		m.recipients[campaignID] = append(m.recipients[campaignID], &domain.CampaignRecipient{CampaignID: campaignID, ContactID: id, Status: domain.RecipientPending})
		inserted++
	}
	return inserted, nil
}
func (m *mockCampaignRepo) ListRecipients(ctx context.Context, campaignID int32, limit, offset int32) ([]*domain.CampaignRecipient, error) {
	return m.recipients[campaignID], nil
}

type mockEvaluator struct {
	count      *domain.EvalResult
	contactIDs []int32
}

func (m *mockEvaluator) Count(ctx context.Context, orgID int32, spec []domain.Filter) (*domain.EvalResult, error) {
	if m.count == nil {
		return &domain.EvalResult{Total: int64(len(m.contactIDs))}, nil
	}
	return m.count, nil
}
func (m *mockEvaluator) ContactIDs(ctx context.Context, orgID int32, spec []domain.Filter) ([]int32, error) {
	return m.contactIDs, nil
}

type mockActivityRepo struct {
	crmDomain.ActivityRepository
	created int
}

func (m *mockActivityRepo) Create(ctx context.Context, a *crmDomain.Activity) (*crmDomain.Activity, error) {
	m.created++
	return a, nil
}

// ---------- tests ----------

func TestLaunchCampaignSnapshotsAudienceAndTransitions(t *testing.T) {
	campaignRepo := newMockCampaignRepo()
	segmentRepo := newMockSegmentRepo()
	evaluator := &mockEvaluator{contactIDs: []int32{1, 2, 3}}
	activityRepo := &mockActivityRepo{}

	segment, _ := segmentRepo.Create(context.Background(), 42, "Clientes", []domain.Filter{{Field: domain.FieldLeadStatus, Op: domain.OpEq, Value: "cliente"}}, "m1")
	campaign, _ := campaignRepo.Create(context.Background(), 42, "Promo", segment.ID, "", "m1")

	svc := NewCampaignService(campaignRepo, segmentRepo, evaluator, activityRepo)
	launched, err := svc.Launch(context.Background(), 42, campaign.ID, "m1")
	if err != nil {
		t.Fatalf("launch failed: %v", err)
	}
	if launched.Status != domain.CampaignReady {
		t.Fatalf("expected ready, got %s", launched.Status)
	}
	if launched.RecipientCount != 3 {
		t.Fatalf("expected recipient_count 3, got %d", launched.RecipientCount)
	}
	if activityRepo.created != 1 {
		t.Fatalf("expected audit activity, got %d", activityRepo.created)
	}
}

func TestLaunchCampaignEmptyAudience(t *testing.T) {
	campaignRepo := newMockCampaignRepo()
	segmentRepo := newMockSegmentRepo()
	activityRepo := &mockActivityRepo{}
	evaluator := &mockEvaluator{contactIDs: []int32{}}

	segment, _ := segmentRepo.Create(context.Background(), 42, "Vacíos", nil, "m1")
	// Overwrite spec to valid one; Create validates only via service, repo mock accepts.
	campaign, _ := campaignRepo.Create(context.Background(), 42, "Promo", segment.ID, "", "m1")

	svc := NewCampaignService(campaignRepo, segmentRepo, evaluator, activityRepo)
	launched, err := svc.Launch(context.Background(), 42, campaign.ID, "m1")
	if err != nil {
		t.Fatalf("launch failed: %v", err)
	}
	if launched.RecipientCount != 0 {
		t.Fatalf("expected 0 recipients, got %d", launched.RecipientCount)
	}
}

func TestRelaunchReturnsErrCampaignNotDraft(t *testing.T) {
	campaignRepo := newMockCampaignRepo()
	segmentRepo := newMockSegmentRepo()
	activityRepo := &mockActivityRepo{}
	evaluator := &mockEvaluator{contactIDs: []int32{1}}

	segment, _ := segmentRepo.Create(context.Background(), 42, "S", []domain.Filter{{Field: domain.FieldLeadStatus, Op: domain.OpEq, Value: "cliente"}}, "m1")
	campaign, _ := campaignRepo.Create(context.Background(), 42, "Promo", segment.ID, "", "m1")

	svc := NewCampaignService(campaignRepo, segmentRepo, evaluator, activityRepo)
	if _, err := svc.Launch(context.Background(), 42, campaign.ID, "m1"); err != nil {
		t.Fatalf("first launch failed: %v", err)
	}
	_, err := svc.Launch(context.Background(), 42, campaign.ID, "m1")
	if !errors.Is(err, domain.ErrCampaignNotDraft) {
		t.Fatalf("expected ErrCampaignNotDraft, got %v", err)
	}
}

func TestLaunchUnknownCampaign(t *testing.T) {
	svc := NewCampaignService(newMockCampaignRepo(), newMockSegmentRepo(), &mockEvaluator{}, &mockActivityRepo{})
	_, err := svc.Launch(context.Background(), 42, 999, "m1")
	if !errors.Is(err, domain.ErrCampaignNotFound) {
		t.Fatalf("expected ErrCampaignNotFound, got %v", err)
	}
}

func TestCreateCampaignRejectsUnknownSegment(t *testing.T) {
	svc := NewCampaignService(newMockCampaignRepo(), newMockSegmentRepo(), &mockEvaluator{}, &mockActivityRepo{})
	_, err := svc.Create(context.Background(), 42, "Promo", 999, "", "m1")
	if err == nil {
		t.Fatalf("expected error for unknown segment")
	}
}

func TestCreateCampaignPersistsMensaje(t *testing.T) {
	campaignRepo := newMockCampaignRepo()
	segmentRepo := newMockSegmentRepo()
	segment, _ := segmentRepo.Create(context.Background(), 42, "Clientes", []domain.Filter{{Field: domain.FieldLeadStatus, Op: domain.OpEq, Value: "cliente"}}, "m1")

	svc := NewCampaignService(campaignRepo, segmentRepo, &mockEvaluator{}, &mockActivityRepo{})
	campaign, err := svc.Create(context.Background(), 42, "Promo", segment.ID, "¡Hola! Oferta especial esta semana. Responde SÍ para más info.", "m1")
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if campaign.Mensaje == nil || *campaign.Mensaje != "¡Hola! Oferta especial esta semana. Responde SÍ para más info." {
		t.Fatalf("expected mensaje persisted, got %v", campaign.Mensaje)
	}
}

func TestCreateCampaignWithoutMensaje(t *testing.T) {
	campaignRepo := newMockCampaignRepo()
	segmentRepo := newMockSegmentRepo()
	segment, _ := segmentRepo.Create(context.Background(), 42, "Clientes", []domain.Filter{{Field: domain.FieldLeadStatus, Op: domain.OpEq, Value: "cliente"}}, "m1")

	svc := NewCampaignService(campaignRepo, segmentRepo, &mockEvaluator{}, &mockActivityRepo{})
	campaign, err := svc.Create(context.Background(), 42, "Promo", segment.ID, "", "m1")
	if err != nil {
		t.Fatalf("create without mensaje failed: %v", err)
	}
	if campaign.Mensaje != nil {
		t.Fatalf("expected nil mensaje for old clients, got %q", *campaign.Mensaje)
	}
	if campaign.Nombre != "Promo" || campaign.SegmentID != segment.ID {
		t.Fatalf("unexpected campaign: %+v", campaign)
	}
}

func TestSegmentServiceRejectsInvalidSpec(t *testing.T) {
	svc := NewSegmentService(newMockSegmentRepo(), &mockEvaluator{}, nil)
	_, err := svc.Create(context.Background(), 42, "Mal", []domain.Filter{{Field: "nope", Op: "eq", Value: "x"}}, "m1")
	if !errors.Is(err, domain.ErrInvalidFilterSpec) {
		t.Fatalf("expected ErrInvalidFilterSpec, got %v", err)
	}
}

func TestSegmentServicePreview(t *testing.T) {
	svc := NewSegmentService(newMockSegmentRepo(), &mockEvaluator{count: &domain.EvalResult{Total: 10, ExcludedByGates: 3}}, nil)
	res, err := svc.Preview(context.Background(), 42, []domain.Filter{{Field: domain.FieldLeadStatus, Op: domain.OpEq, Value: "cliente"}})
	if err != nil {
		t.Fatalf("preview failed: %v", err)
	}
	if res.Total != 10 || res.ExcludedByGates != 3 {
		t.Fatalf("unexpected preview result: %+v", res)
	}
}
