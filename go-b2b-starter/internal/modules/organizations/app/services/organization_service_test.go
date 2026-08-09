package services_test

import (
	"context"
	"errors"
	"testing"

	"github.com/moasq/go-b2b-starter/internal/modules/organizations/app/services"
	"github.com/moasq/go-b2b-starter/internal/modules/organizations/domain"
)

// --- Fakes ---------------------------------------------------------------

type fakeOrgRepo struct {
	org        *domain.Organization
	updateArgs *domain.Organization
	updateErr  error
}

func (f *fakeOrgRepo) Create(ctx context.Context, org *domain.Organization) (*domain.Organization, error) {
	return org, nil
}
func (f *fakeOrgRepo) GetByID(ctx context.Context, id int32) (*domain.Organization, error) {
	if f.org == nil {
		return nil, domain.ErrOrganizationNotFound
	}
	return f.org, nil
}
func (f *fakeOrgRepo) GetBySlug(ctx context.Context, slug string) (*domain.Organization, error) {
	return f.org, nil
}
func (f *fakeOrgRepo) GetByStytchID(ctx context.Context, stytchOrgID string) (*domain.Organization, error) {
	return f.org, nil
}
func (f *fakeOrgRepo) GetByUserEmail(ctx context.Context, email string) (*domain.Organization, error) {
	return f.org, nil
}
func (f *fakeOrgRepo) Update(ctx context.Context, org *domain.Organization) (*domain.Organization, error) {
	if f.updateErr != nil {
		return nil, f.updateErr
	}
	f.updateArgs = org
	f.org = org
	return org, nil
}
func (f *fakeOrgRepo) UpdateStytchInfo(ctx context.Context, id int32, stytchOrgID, stytchConnectionID, stytchConnectionName string) (*domain.Organization, error) {
	return f.org, nil
}
func (f *fakeOrgRepo) Delete(ctx context.Context, id int32) error {
	return nil
}
func (f *fakeOrgRepo) List(ctx context.Context, limit, offset int32) ([]*domain.Organization, error) {
	return []*domain.Organization{f.org}, nil
}
func (f *fakeOrgRepo) GetStats(ctx context.Context, id int32) (*domain.OrganizationStats, error) {
	return nil, nil
}

type fakeAccountRepo struct {
	accounts   []*domain.Account
	updateArgs *domain.Account
}

func (f *fakeAccountRepo) Create(ctx context.Context, account *domain.Account) (*domain.Account, error) {
	return account, nil
}
func (f *fakeAccountRepo) GetByID(ctx context.Context, orgID, accountID int32) (*domain.Account, error) {
	for _, a := range f.accounts {
		if a.ID == accountID {
			return a, nil
		}
	}
	return nil, domain.ErrAccountNotFound
}
func (f *fakeAccountRepo) GetByEmail(ctx context.Context, orgID int32, email string) (*domain.Account, error) {
	return nil, nil
}
func (f *fakeAccountRepo) ListByOrganization(ctx context.Context, orgID int32) ([]*domain.Account, error) {
	return f.accounts, nil
}
func (f *fakeAccountRepo) Update(ctx context.Context, account *domain.Account) (*domain.Account, error) {
	f.updateArgs = account
	return account, nil
}
func (f *fakeAccountRepo) UpdateStytchInfo(ctx context.Context, orgID, accountID int32, stytchMemberID, stytchRoleID, stytchRoleSlug string, stytchEmailVerified bool) (*domain.Account, error) {
	return nil, nil
}
func (f *fakeAccountRepo) UpdateLastLogin(ctx context.Context, orgID, accountID int32) (*domain.Account, error) {
	return nil, nil
}
func (f *fakeAccountRepo) Delete(ctx context.Context, orgID, accountID int32) error {
	return nil
}
func (f *fakeAccountRepo) GetOrganization(ctx context.Context, accountID int32) (*domain.Organization, error) {
	return nil, nil
}
func (f *fakeAccountRepo) CheckPermission(ctx context.Context, orgID, accountID int32) (*domain.AccountPermission, error) {
	return nil, nil
}
func (f *fakeAccountRepo) GetStats(ctx context.Context, accountID int32) (*domain.AccountStats, error) {
	return nil, nil
}

type fakeAuthOrgRepo struct {
	updateOrgID string
	updateName  string
	err         error
}

func (f *fakeAuthOrgRepo) CreateOrganization(ctx context.Context, req *domain.CreateAuthOrganizationRequest) (*domain.AuthOrganization, error) {
	return nil, nil
}
func (f *fakeAuthOrgRepo) GetOrganization(ctx context.Context, organizationID string) (*domain.AuthOrganization, error) {
	return &domain.AuthOrganization{OrganizationID: organizationID, DisplayName: f.updateName}, nil
}
func (f *fakeAuthOrgRepo) UpdateOrganization(ctx context.Context, organizationID, displayName string) (*domain.AuthOrganization, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.updateOrgID = organizationID
	f.updateName = displayName
	return &domain.AuthOrganization{OrganizationID: organizationID, DisplayName: displayName}, nil
}
func (f *fakeAuthOrgRepo) DeleteOrganization(ctx context.Context, organizationID string) error {
	return nil
}
func (f *fakeAuthOrgRepo) CheckEmailExists(ctx context.Context, email string) (bool, error) {
	return false, nil
}

type fakeAuthMemberRepo struct {
	assignReq *domain.AssignAuthRolesRequest
	err       error
}

func (f *fakeAuthMemberRepo) CreateMember(ctx context.Context, req *domain.CreateAuthMemberRequest) (*domain.AuthMember, error) {
	return nil, nil
}
func (f *fakeAuthMemberRepo) UpdateMember(ctx context.Context, req *domain.UpdateAuthMemberRequest) (*domain.AuthMember, error) {
	return nil, nil
}
func (f *fakeAuthMemberRepo) GetMember(ctx context.Context, organizationID, memberID string) (*domain.AuthMember, error) {
	return nil, nil
}
func (f *fakeAuthMemberRepo) GetMemberByEmail(ctx context.Context, organizationID, email string) (*domain.AuthMember, error) {
	return nil, nil
}
func (f *fakeAuthMemberRepo) ListMembers(ctx context.Context, organizationID string, limit, offset int) ([]*domain.AuthMember, error) {
	return nil, nil
}
func (f *fakeAuthMemberRepo) RemoveMembers(ctx context.Context, req *domain.RemoveAuthMembersRequest) error {
	return nil
}
func (f *fakeAuthMemberRepo) AssignRoles(ctx context.Context, req *domain.AssignAuthRolesRequest) error {
	if f.err != nil {
		return f.err
	}
	f.assignReq = req
	return nil
}
func (f *fakeAuthMemberRepo) SendMagicLink(ctx context.Context, req *domain.SendMagicLinkRequest) error {
	return nil
}

func newService(org *fakeOrgRepo, acc *fakeAccountRepo, authOrg *fakeAuthOrgRepo, authMember *fakeAuthMemberRepo) services.OrganizationService {
	return services.NewOrganizationService(org, acc, authOrg, authMember)
}

// --- UpdateOrganization tests ---------------------------------------------

func TestUpdateOrganizationSyncsStytchThenLocal(t *testing.T) {
	orgRepo := &fakeOrgRepo{org: &domain.Organization{ID: 1, Name: "Old", Status: "active", StytchOrgID: "stytch-org-1"}}
	accRepo := &fakeAccountRepo{}
	authOrg := &fakeAuthOrgRepo{}
	authMember := &fakeAuthMemberRepo{}
	svc := newService(orgRepo, accRepo, authOrg, authMember)

	ctx := context.Background()
	_, err := svc.UpdateOrganization(ctx, 1, &services.UpdateOrganizationRequest{Name: "New Name", Status: "active"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if authOrg.updateOrgID != "stytch-org-1" || authOrg.updateName != "New Name" {
		t.Fatalf("stytch org not synced: got orgID=%q name=%q", authOrg.updateOrgID, authOrg.updateName)
	}
	if orgRepo.updateArgs == nil || orgRepo.updateArgs.Name != "New Name" {
		t.Fatalf("local org not updated: %+v", orgRepo.updateArgs)
	}
}

func TestUpdateOrganizationStytchFailureSkipsLocalWrite(t *testing.T) {
	orgRepo := &fakeOrgRepo{org: &domain.Organization{ID: 1, Name: "Old", Status: "active", StytchOrgID: "stytch-org-1"}}
	authOrg := &fakeAuthOrgRepo{err: errors.New("stytch down")}
	svc := newService(orgRepo, &fakeAccountRepo{}, authOrg, &fakeAuthMemberRepo{})

	_, err := svc.UpdateOrganization(context.Background(), 1, &services.UpdateOrganizationRequest{Name: "New", Status: "active"})
	if err == nil {
		t.Fatal("expected error when Stytch update fails")
	}
	if orgRepo.updateArgs != nil {
		t.Fatal("local org write must be skipped when Stytch sync fails")
	}
}

func TestUpdateOrganizationNoStytchIDSkipsSync(t *testing.T) {
	orgRepo := &fakeOrgRepo{org: &domain.Organization{ID: 1, Name: "Old", Status: "active"}}
	authOrg := &fakeAuthOrgRepo{}
	svc := newService(orgRepo, &fakeAccountRepo{}, authOrg, &fakeAuthMemberRepo{})

	_, err := svc.UpdateOrganization(context.Background(), 1, &services.UpdateOrganizationRequest{Name: "New", Status: "active"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if authOrg.updateOrgID != "" {
		t.Fatal("should not call Stytch when no stytch_org_id present")
	}
	if orgRepo.updateArgs == nil || orgRepo.updateArgs.Name != "New" {
		t.Fatalf("local org not updated: %+v", orgRepo.updateArgs)
	}
}

// --- UpdateAccount tests ----------------------------------------------------

func TestUpdateAccountRoleChangeSyncsStytch(t *testing.T) {
	orgRepo := &fakeOrgRepo{org: &domain.Organization{ID: 1, Name: "Acme", Status: "active", StytchOrgID: "stytch-org-1"}}
	accRepo := &fakeAccountRepo{accounts: []*domain.Account{
		{ID: 1, OrganizationID: 1, Email: "a@x.com", FullName: "A", StytchMemberID: "stytch-member-1", Role: "member", Status: "active"},
		{ID: 2, OrganizationID: 1, Email: "b@x.com", FullName: "B", StytchMemberID: "stytch-member-2", Role: "admin", Status: "active"},
	}}
	authMember := &fakeAuthMemberRepo{}
	svc := newService(orgRepo, accRepo, &fakeAuthOrgRepo{}, authMember)

	_, err := svc.UpdateAccount(context.Background(), 1, 1, &services.UpdateAccountRequest{
		FullName: "A", Role: "approver", Status: "active",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if authMember.assignReq == nil {
		t.Fatal("expected Stytch role assignment")
	}
	if authMember.assignReq.OrganizationID != "stytch-org-1" || authMember.assignReq.MemberID != "stytch-member-1" {
		t.Fatalf("unexpected assign request: %+v", authMember.assignReq)
	}
	if len(authMember.assignReq.Roles) != 1 || authMember.assignReq.Roles[0] != "approver" {
		t.Fatalf("unexpected roles: %v", authMember.assignReq.Roles)
	}
	if accRepo.updateArgs == nil || accRepo.updateArgs.Role != "approver" || accRepo.updateArgs.StytchRoleSlug != "approver" {
		t.Fatalf("local account not updated: %+v", accRepo.updateArgs)
	}
}

func TestUpdateAccountStytchFailureSkipsLocalWrite(t *testing.T) {
	orgRepo := &fakeOrgRepo{org: &domain.Organization{ID: 1, Name: "Acme", Status: "active", StytchOrgID: "stytch-org-1"}}
	accRepo := &fakeAccountRepo{accounts: []*domain.Account{
		{ID: 1, OrganizationID: 1, Email: "a@x.com", FullName: "A", StytchMemberID: "stytch-member-1", Role: "member", Status: "active"},
		{ID: 2, OrganizationID: 1, Email: "b@x.com", FullName: "B", StytchMemberID: "stytch-member-2", Role: "admin", Status: "active"},
	}}
	authMember := &fakeAuthMemberRepo{err: errors.New("stytch down")}
	svc := newService(orgRepo, accRepo, &fakeAuthOrgRepo{}, authMember)

	_, err := svc.UpdateAccount(context.Background(), 1, 1, &services.UpdateAccountRequest{
		FullName: "A", Role: "approver", Status: "active",
	})
	if err == nil {
		t.Fatal("expected error when Stytch role sync fails")
	}
	if accRepo.updateArgs != nil {
		t.Fatal("local account write must be skipped when Stytch sync fails")
	}
}

func TestUpdateAccountLastAdminDemotionRejected(t *testing.T) {
	orgRepo := &fakeOrgRepo{org: &domain.Organization{ID: 1, Name: "Acme", Status: "active", StytchOrgID: "stytch-org-1"}}
	accRepo := &fakeAccountRepo{accounts: []*domain.Account{
		{ID: 1, OrganizationID: 1, Email: "a@x.com", FullName: "A", StytchMemberID: "stytch-member-1", Role: "admin", Status: "active"},
	}}
	authMember := &fakeAuthMemberRepo{}
	svc := newService(orgRepo, accRepo, &fakeAuthOrgRepo{}, authMember)

	_, err := svc.UpdateAccount(context.Background(), 1, 1, &services.UpdateAccountRequest{
		FullName: "A", Role: "member", Status: "active",
	})
	if !errors.Is(err, domain.ErrLastAdminDemotion) {
		t.Fatalf("expected ErrLastAdminDemotion, got %v", err)
	}
	if authMember.assignReq != nil {
		t.Fatal("must not assign roles when demotion is rejected")
	}
	if accRepo.updateArgs != nil {
		t.Fatal("must not write locally when demotion is rejected")
	}
}

func TestUpdateAccountAdminDemotionAllowedWithSecondAdmin(t *testing.T) {
	orgRepo := &fakeOrgRepo{org: &domain.Organization{ID: 1, Name: "Acme", Status: "active", StytchOrgID: "stytch-org-1"}}
	accRepo := &fakeAccountRepo{accounts: []*domain.Account{
		{ID: 1, OrganizationID: 1, Email: "a@x.com", FullName: "A", StytchMemberID: "stytch-member-1", Role: "admin", Status: "active"},
		{ID: 2, OrganizationID: 1, Email: "b@x.com", FullName: "B", StytchMemberID: "stytch-member-2", Role: "admin", Status: "active"},
	}}
	authMember := &fakeAuthMemberRepo{}
	svc := newService(orgRepo, accRepo, &fakeAuthOrgRepo{}, authMember)

	_, err := svc.UpdateAccount(context.Background(), 1, 1, &services.UpdateAccountRequest{
		FullName: "A", Role: "member", Status: "active",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if authMember.assignReq == nil {
		t.Fatal("expected Stytch role assignment")
	}
	if accRepo.updateArgs == nil || accRepo.updateArgs.Role != "member" {
		t.Fatalf("local account not updated: %+v", accRepo.updateArgs)
	}
}
