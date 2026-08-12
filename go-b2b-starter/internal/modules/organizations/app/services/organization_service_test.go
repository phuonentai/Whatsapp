package services_test

import (
	"context"
	"errors"
	"testing"

	"github.com/moasq/go-b2b-starter/internal/modules/organizations/app/services"
	"github.com/moasq/go-b2b-starter/internal/modules/organizations/domain"
	logger "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
)

type noopLogger struct{}

func (noopLogger) Debug(msg string, fields ...logger.Fields) {}
func (noopLogger) Info(msg string, fields ...logger.Fields)  {}
func (noopLogger) Warn(msg string, fields ...logger.Fields)  {}
func (noopLogger) Error(msg string, fields ...logger.Fields) {}
func (noopLogger) Fatal(msg string, fields ...logger.Fields) {}
func (noopLogger) WithFields(fields logger.Fields) logger.Logger {
	return noopLogger{}
}

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

type fakeSessionRevoker struct {
	orgID    string
	memberID string
	calls    int
	err      error
}

func (f *fakeSessionRevoker) RevokeMemberSessions(ctx context.Context, stytchOrgID, stytchMemberID string) error {
	f.calls++
	f.orgID = stytchOrgID
	f.memberID = stytchMemberID
	return f.err
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

type fakeMfaPolicyUpdater struct {
	err        error
	lastOrgID  string
	lastPolicy domain.MfaPolicy
	lastCalls  int
}

func (f *fakeMfaPolicyUpdater) UpdateMfaPolicy(
	ctx context.Context,
	orgID string,
	policy domain.MfaPolicy,
	methods domain.MfaMethods,
	allowedMethods []domain.MfaMethod,
) error {
	f.lastCalls++
	f.lastOrgID = orgID
	f.lastPolicy = policy
	return f.err
}

// fakeAuthPolicyUpdater satisfies domain.OrgAuthPolicyUpdater for service
// tests. It records the last update call and can inject errors.
type fakeAuthPolicyUpdater struct {
	err          error
	lastOrgID    string
	lastEmailJIT domain.JitPolicy
	lastCalls    int
	policy       *domain.AuthPolicy
}

func (f *fakeAuthPolicyUpdater) GetAuthPolicy(ctx context.Context, orgID string) (*domain.AuthPolicy, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.policy, nil
}

func (f *fakeAuthPolicyUpdater) UpdateAuthPolicy(
	ctx context.Context,
	orgID string,
	emailJitPolicy domain.JitPolicy,
	allowedDomains []string,
	allowedAuthMethods []domain.AllowedAuthMethod,
	ssoJitPolicy domain.SsoJitPolicy,
	ssoJitAllowedConnections []string,
	ssoDefaultConnectionID string,
) error {
	f.lastCalls++
	f.lastOrgID = orgID
	f.lastEmailJIT = emailJitPolicy
	return f.err
}

func newService(org *fakeOrgRepo, acc *fakeAccountRepo, authOrg *fakeAuthOrgRepo, authMember *fakeAuthMemberRepo) services.OrganizationService {
	return newServiceWithDeps(org, acc, authOrg, authMember, &fakeSessionRevoker{}, &fakeMfaPolicyUpdater{}, &fakeAuthPolicyUpdater{})
}

func newServiceWithRevoker(org *fakeOrgRepo, acc *fakeAccountRepo, authOrg *fakeAuthOrgRepo, authMember *fakeAuthMemberRepo, revoker *fakeSessionRevoker) services.OrganizationService {
	return newServiceWithDeps(org, acc, authOrg, authMember, revoker, &fakeMfaPolicyUpdater{}, &fakeAuthPolicyUpdater{})
}

func newServiceWithDeps(
	org *fakeOrgRepo,
	acc *fakeAccountRepo,
	authOrg *fakeAuthOrgRepo,
	authMember *fakeAuthMemberRepo,
	revoker *fakeSessionRevoker,
	updater *fakeMfaPolicyUpdater,
	authPolicyUpdater *fakeAuthPolicyUpdater,
) services.OrganizationService {
	return services.NewOrganizationService(org, acc, authOrg, authMember, revoker, updater, authPolicyUpdater, noopLogger{})
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

// --- Deactivation revocation tests ------------------------------------------

func TestUpdateAccountDeactivationRevokesSessionsAfterStatusUpdate(t *testing.T) {
	orgRepo := &fakeOrgRepo{org: &domain.Organization{ID: 1, Name: "Acme", Status: "active", StytchOrgID: "stytch-org-1"}}
	accRepo := &fakeAccountRepo{accounts: []*domain.Account{
		{ID: 1, OrganizationID: 1, Email: "a@x.com", FullName: "A", StytchMemberID: "stytch-member-1", Role: "member", Status: "active"},
	}}
	revoker := &fakeSessionRevoker{}
	svc := newServiceWithRevoker(orgRepo, accRepo, &fakeAuthOrgRepo{}, &fakeAuthMemberRepo{}, revoker)

	updated, err := svc.UpdateAccount(context.Background(), 1, 1, &services.UpdateAccountRequest{
		FullName: "A", Role: "member", Status: "inactive",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Local status update must complete first...
	if accRepo.updateArgs == nil || accRepo.updateArgs.Status != "inactive" {
		t.Fatalf("local status update not applied: %+v", accRepo.updateArgs)
	}
	if updated.Status != "inactive" {
		t.Fatalf("expected inactive status, got %q", updated.Status)
	}
	// ...then the Stytch session revocation runs with the org+member IDs.
	if revoker.calls != 1 {
		t.Fatalf("expected exactly 1 revocation call, got %d", revoker.calls)
	}
	if revoker.orgID != "stytch-org-1" || revoker.memberID != "stytch-member-1" {
		t.Fatalf("unexpected revocation target: org=%q member=%q", revoker.orgID, revoker.memberID)
	}
	if updated.SessionRevocationPending {
		t.Fatal("revocation succeeded; must not carry pending notice")
	}
}

func TestUpdateAccountRevocationFailureCompletesDeactivationWithNotice(t *testing.T) {
	orgRepo := &fakeOrgRepo{org: &domain.Organization{ID: 1, Name: "Acme", Status: "active", StytchOrgID: "stytch-org-1"}}
	accRepo := &fakeAccountRepo{accounts: []*domain.Account{
		{ID: 1, OrganizationID: 1, Email: "a@x.com", FullName: "A", StytchMemberID: "stytch-member-1", Role: "member", Status: "active"},
	}}
	revoker := &fakeSessionRevoker{err: errors.New("stytch circuit breaker open")}
	svc := newServiceWithRevoker(orgRepo, accRepo, &fakeAuthOrgRepo{}, &fakeAuthMemberRepo{}, revoker)

	updated, err := svc.UpdateAccount(context.Background(), 1, 1, &services.UpdateAccountRequest{
		FullName: "A", Role: "member", Status: "inactive",
	})
	if err != nil {
		t.Fatalf("deactivation must not fail when revocation fails: %v", err)
	}

	if accRepo.updateArgs == nil || accRepo.updateArgs.Status != "inactive" {
		t.Fatalf("local deactivation must still be applied: %+v", accRepo.updateArgs)
	}
	if revoker.calls != 1 {
		t.Fatalf("expected revocation attempt, got %d calls", revoker.calls)
	}
	if !updated.SessionRevocationPending {
		t.Fatal("expected session_revocation_pending notice when revocation fails")
	}
}

func TestUpdateAccountActiveRoleChangeSkipsRevocation(t *testing.T) {
	orgRepo := &fakeOrgRepo{org: &domain.Organization{ID: 1, Name: "Acme", Status: "active", StytchOrgID: "stytch-org-1"}}
	accRepo := &fakeAccountRepo{accounts: []*domain.Account{
		{ID: 1, OrganizationID: 1, Email: "a@x.com", FullName: "A", StytchMemberID: "stytch-member-1", Role: "member", Status: "active"},
	}}
	revoker := &fakeSessionRevoker{}
	svc := newServiceWithRevoker(orgRepo, accRepo, &fakeAuthOrgRepo{}, &fakeAuthMemberRepo{}, revoker)

	_, err := svc.UpdateAccount(context.Background(), 1, 1, &services.UpdateAccountRequest{
		FullName: "A", Role: "approver", Status: "active",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if revoker.calls != 0 {
		t.Fatalf("expected no revocation for a non-deactivation update, got %d calls", revoker.calls)
	}
}

func TestUpdateAccountDeactivationWithoutStytchMemberSkipsRevocation(t *testing.T) {
	orgRepo := &fakeOrgRepo{org: &domain.Organization{ID: 1, Name: "Acme", Status: "active", StytchOrgID: "stytch-org-1"}}
	accRepo := &fakeAccountRepo{accounts: []*domain.Account{
		{ID: 1, OrganizationID: 1, Email: "a@x.com", FullName: "A", Role: "member", Status: "active"},
	}}
	revoker := &fakeSessionRevoker{}
	svc := newServiceWithRevoker(orgRepo, accRepo, &fakeAuthOrgRepo{}, &fakeAuthMemberRepo{}, revoker)

	updated, err := svc.UpdateAccount(context.Background(), 1, 1, &services.UpdateAccountRequest{
		FullName: "A", Role: "member", Status: "inactive",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if revoker.calls != 0 {
		t.Fatalf("expected no revocation without a Stytch member id, got %d calls", revoker.calls)
	}
	if updated.SessionRevocationPending {
		t.Fatal("must not carry pending notice when no revocation was attempted")
	}
}

// --- UpdateMfaPolicy tests --------------------------------------------------

func TestUpdateMfaPolicyDelegatesToUpdater(t *testing.T) {
	orgRepo := &fakeOrgRepo{org: &domain.Organization{ID: 1, Name: "Acme", Status: "active"}}
	accRepo := &fakeAccountRepo{}
	authOrg := &fakeAuthOrgRepo{}
	authMember := &fakeAuthMemberRepo{}
	updater := &fakeMfaPolicyUpdater{}
	svc := newServiceWithDeps(orgRepo, accRepo, authOrg, authMember, &fakeSessionRevoker{}, updater, &fakeAuthPolicyUpdater{})

	err := svc.UpdateMfaPolicy(
		context.Background(),
		"stytch-org-1",
		domain.MfaPolicyRequiredForAll,
		domain.MfaMethodsRestricted,
		[]domain.MfaMethod{domain.MfaMethodTOTP},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updater.lastCalls != 1 {
		t.Fatalf("expected 1 updater call, got %d", updater.lastCalls)
	}
	if updater.lastOrgID != "stytch-org-1" {
		t.Fatalf("expected org id stytch-org-1, got %q", updater.lastOrgID)
	}
	if updater.lastPolicy != domain.MfaPolicyRequiredForAll {
		t.Fatalf("expected REQUIRED_FOR_ALL policy, got %q", updater.lastPolicy)
	}
}

func TestUpdateMfaPolicyValidationRejectsBadPayload(t *testing.T) {
	orgRepo := &fakeOrgRepo{org: &domain.Organization{ID: 1, Name: "Acme", Status: "active"}}
	accRepo := &fakeAccountRepo{}
	updater := &fakeMfaPolicyUpdater{}
	svc := newServiceWithDeps(orgRepo, accRepo, &fakeAuthOrgRepo{}, &fakeAuthMemberRepo{}, &fakeSessionRevoker{}, updater, &fakeAuthPolicyUpdater{})

	if err := svc.UpdateMfaPolicy(context.Background(), "", domain.MfaPolicyOptional, domain.MfaMethodsAllAllowed, nil); !errors.Is(err, domain.ErrAuthOrganizationIDRequired) {
		t.Fatalf("expected ErrAuthOrganizationIDRequired, got: %v", err)
	}
	if err := svc.UpdateMfaPolicy(context.Background(), "org-1", domain.MfaPolicy("NEVER"), domain.MfaMethodsAllAllowed, nil); !errors.Is(err, domain.ErrInvalidMfaPolicy) {
		t.Fatalf("expected ErrInvalidMfaPolicy, got: %v", err)
	}
	if updater.lastCalls != 0 {
		t.Fatalf("expected no updater calls on invalid payload, got %d", updater.lastCalls)
	}
}

func TestUpdateMfaPolicyUpdaterErrorPropagates(t *testing.T) {
	orgRepo := &fakeOrgRepo{org: &domain.Organization{ID: 1, Name: "Acme", Status: "active"}}
	accRepo := &fakeAccountRepo{}
	updater := &fakeMfaPolicyUpdater{err: domain.ErrMfaPolicyUnavailable}
	svc := newServiceWithDeps(orgRepo, accRepo, &fakeAuthOrgRepo{}, &fakeAuthMemberRepo{}, &fakeSessionRevoker{}, updater, &fakeAuthPolicyUpdater{})

	err := svc.UpdateMfaPolicy(context.Background(), "org-1", domain.MfaPolicyOptional, domain.MfaMethodsAllAllowed, nil)
	if !errors.Is(err, domain.ErrMfaPolicyUnavailable) {
		t.Fatalf("expected ErrMfaPolicyUnavailable to propagate, got: %v", err)
	}
}
