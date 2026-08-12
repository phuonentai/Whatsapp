package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/moasq/go-b2b-starter/internal/modules/organizations/domain"
	loggerDomain "github.com/moasq/go-b2b-starter/internal/platform/logger"
)

type organizationService struct {
	orgRepo           domain.OrganizationRepository
	accountRepo       domain.AccountRepository
	authOrgRepo       domain.AuthOrganizationRepository
	authMemberRepo    domain.AuthMemberRepository
	sessionRevoker    domain.SessionRevoker
	mfaPolicyUpdater  domain.MfaPolicyUpdater
	authPolicyUpdater domain.OrgAuthPolicyUpdater
	logger            loggerDomain.Logger
}

func NewOrganizationService(
	orgRepo domain.OrganizationRepository,
	accountRepo domain.AccountRepository,
	authOrgRepo domain.AuthOrganizationRepository,
	authMemberRepo domain.AuthMemberRepository,
	sessionRevoker domain.SessionRevoker,
	mfaPolicyUpdater domain.MfaPolicyUpdater,
	authPolicyUpdater domain.OrgAuthPolicyUpdater,
	logger loggerDomain.Logger,
) OrganizationService {
	return &organizationService{
		orgRepo:           orgRepo,
		accountRepo:       accountRepo,
		authOrgRepo:       authOrgRepo,
		authMemberRepo:    authMemberRepo,
		sessionRevoker:    sessionRevoker,
		mfaPolicyUpdater:  mfaPolicyUpdater,
		authPolicyUpdater: authPolicyUpdater,
		logger:            logger,
	}
}

func (s *organizationService) CreateOrganization(ctx context.Context, req *CreateOrganizationRequest) (*domain.Organization, error) {
	// Create organization
	org := &domain.Organization{
		Slug:                 req.Slug,
		Name:                 req.Name,
		Status:               "active",
		StytchOrgID:          req.StytchOrgID,
		StytchConnectionID:   req.StytchConnectionID,
		StytchConnectionName: req.StytchConnectionName,
	}

	createdOrg, err := s.orgRepo.Create(ctx, org)
	if err != nil {
		return nil, fmt.Errorf("failed to create organization: %w", err)
	}

	if req.StytchOrgID != "" || req.StytchConnectionID != "" || req.StytchConnectionName != "" {
		createdOrg.StytchOrgID = req.StytchOrgID
		createdOrg.StytchConnectionID = req.StytchConnectionID
		createdOrg.StytchConnectionName = req.StytchConnectionName
		createdOrg, err = s.orgRepo.Update(ctx, createdOrg)
		if err != nil {
			return nil, fmt.Errorf("failed to persist organization Stytch metadata: %w", err)
		}
	}

	// Create admin account (primary admin user)
	adminAccount := &domain.Account{
		OrganizationID: createdOrg.ID,
		Email:          req.OwnerEmail,
		FullName:       req.OwnerName,
		Role:           "admin",
		Status:         "active",
	}

	_, err = s.accountRepo.Create(ctx, adminAccount)
	if err != nil {
		return nil, fmt.Errorf("failed to create admin account: %w", err)
	}

	return createdOrg, nil
}

func (s *organizationService) GetOrganization(ctx context.Context, orgID int32) (*domain.Organization, error) {
	return s.orgRepo.GetByID(ctx, orgID)
}

func (s *organizationService) GetOrganizationBySlug(ctx context.Context, slug string) (*domain.Organization, error) {
	return s.orgRepo.GetBySlug(ctx, slug)
}

func (s *organizationService) GetOrganizationByStytchID(ctx context.Context, stytchOrgID string) (*domain.Organization, error) {
	return s.orgRepo.GetByStytchID(ctx, stytchOrgID)
}

func (s *organizationService) GetOrganizationByUserEmail(ctx context.Context, email string) (*domain.Organization, error) {
	return s.orgRepo.GetByUserEmail(ctx, email)
}

func (s *organizationService) UpdateOrganization(ctx context.Context, orgID int32, req *UpdateOrganizationRequest) (*domain.Organization, error) {
	// Get existing organization
	org, err := s.orgRepo.GetByID(ctx, orgID)
	if err != nil {
		return nil, err
	}

	// Keep both SSOTs in phase: sync the display name to Stytch BEFORE the local write.
	// On Stytch failure, reject the update so the local row never drifts from the auth provider.
	if org.StytchOrgID != "" && req.Name != org.Name {
		if _, err := s.authOrgRepo.UpdateOrganization(ctx, org.StytchOrgID, req.Name); err != nil {
			// Development mode: the Stytch client is nil (placeholder credentials).
			// Skip the auth-provider sync; the local row is authoritative in mock mode.
			if !errors.Is(err, domain.ErrAuthConnection) {
				return nil, fmt.Errorf("failed to sync organization display name to auth provider: %w", err)
			}
		}
	}

	// Update fields
	org.Name = req.Name
	org.Status = req.Status
	if req.StytchOrgID != "" {
		org.StytchOrgID = req.StytchOrgID
	}
	if req.StytchConnectionID != "" {
		org.StytchConnectionID = req.StytchConnectionID
	}
	if req.StytchConnectionName != "" {
		org.StytchConnectionName = req.StytchConnectionName
	}

	return s.orgRepo.Update(ctx, org)
}

func (s *organizationService) ListOrganizations(ctx context.Context, req *ListOrganizationsRequest) (*ListOrganizationsResponse, error) {
	organizations, err := s.orgRepo.List(ctx, req.Limit, req.Offset)
	if err != nil {
		return nil, err
	}

	// For simplicity, we're not implementing total count yet
	// In production, you'd want a separate query for total count
	total := int32(len(organizations))

	return &ListOrganizationsResponse{
		Organizations: organizations,
		Total:         total,
		Limit:         req.Limit,
		Offset:        req.Offset,
	}, nil
}

func (s *organizationService) GetOrganizationStats(ctx context.Context, orgID int32) (*domain.OrganizationStats, error) {
	return s.orgRepo.GetStats(ctx, orgID)
}

// UpdateMfaPolicy validates the policy payload and delegates to the auth
// provider adapter. The adapter routes the call through the shared circuit
// breaker; breaker-open/5xx errors surface as domain.ErrMfaPolicyUnavailable
// (503 at the API boundary) and leave the organization's policy unchanged.
func (s *organizationService) UpdateMfaPolicy(
	ctx context.Context,
	orgID string,
	policy domain.MfaPolicy,
	methods domain.MfaMethods,
	allowedMethods []domain.MfaMethod,
) error {
	if orgID == "" {
		return domain.ErrAuthOrganizationIDRequired
	}
	if err := domain.ValidateMfaPolicyUpdate(policy, methods, allowedMethods); err != nil {
		return err
	}
	if err := s.mfaPolicyUpdater.UpdateMfaPolicy(ctx, orgID, policy, methods, allowedMethods); err != nil {
		return fmt.Errorf("failed to update MFA policy: %w", err)
	}
	s.logger.Info("organization MFA policy updated", loggerDomain.Fields{
		"org_id":     orgID,
		"mfa_policy": string(policy),
	})
	return nil
}

// GetAuthPolicy reads the organization's auth policy mirror from the auth
// provider (display-only; Stytch enforces at authentication time). Breaker-
// open/5xx errors surface as domain.ErrAuthPolicyUnavailable (503 at the API
// boundary).
func (s *organizationService) GetAuthPolicy(ctx context.Context, orgID string) (*domain.AuthPolicy, error) {
	if orgID == "" {
		return nil, domain.ErrAuthOrganizationIDRequired
	}
	policy, err := s.authPolicyUpdater.GetAuthPolicy(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get auth policy: %w", err)
	}
	return policy, nil
}

// UpdateAuthPolicy delegates the validated policy payload to the auth provider
// adapter (SSOT). The adapter routes the read-for-preservation and the write
// through the shared circuit breaker; breaker-open/5xx errors surface as
// domain.ErrAuthPolicyUnavailable (503 at the API boundary) and leave the
// organization's policy unchanged.
func (s *organizationService) UpdateAuthPolicy(
	ctx context.Context,
	orgID string,
	emailJitPolicy domain.JitPolicy,
	allowedDomains []string,
	allowedAuthMethods []domain.AllowedAuthMethod,
	ssoJitPolicy domain.SsoJitPolicy,
	ssoJitAllowedConnections []string,
	ssoDefaultConnectionID string,
) error {
	if orgID == "" {
		return domain.ErrAuthOrganizationIDRequired
	}
	if err := s.authPolicyUpdater.UpdateAuthPolicy(
		ctx,
		orgID,
		emailJitPolicy,
		allowedDomains,
		allowedAuthMethods,
		ssoJitPolicy,
		ssoJitAllowedConnections,
		ssoDefaultConnectionID,
	); err != nil {
		return fmt.Errorf("failed to update auth policy: %w", err)
	}
	s.logger.Info("organization auth policy updated", loggerDomain.Fields{
		"org_id":    orgID,
		"email_jit": string(emailJitPolicy),
		"sso_jit":   string(ssoJitPolicy),
	})
	return nil
}

func (s *organizationService) CreateAccount(ctx context.Context, orgID int32, req *CreateAccountRequest) (*domain.Account, error) {
	// Verify organization exists
	_, err := s.orgRepo.GetByID(ctx, orgID)
	if err != nil {
		return nil, err
	}

	account := &domain.Account{
		OrganizationID:      orgID,
		Email:               req.Email,
		FullName:            req.FullName,
		StytchMemberID:      req.StytchMemberID,
		StytchRoleID:        req.StytchRoleID,
		StytchRoleSlug:      req.StytchRoleSlug,
		StytchEmailVerified: req.StytchEmailVerified,
		Role:                req.Role,
		Status:              "active",
	}

	return s.accountRepo.Create(ctx, account)
}

func (s *organizationService) GetAccount(ctx context.Context, orgID, accountID int32) (*domain.Account, error) {
	return s.accountRepo.GetByID(ctx, orgID, accountID)
}

func (s *organizationService) GetAccountByEmail(ctx context.Context, orgID int32, email string) (*domain.Account, error) {
	return s.accountRepo.GetByEmail(ctx, orgID, email)
}

func (s *organizationService) ListAccounts(ctx context.Context, orgID int32) ([]*domain.Account, error) {
	// Verify organization exists
	_, err := s.orgRepo.GetByID(ctx, orgID)
	if err != nil {
		return nil, err
	}

	return s.accountRepo.ListByOrganization(ctx, orgID)
}

func (s *organizationService) UpdateAccount(ctx context.Context, orgID, accountID int32, req *UpdateAccountRequest) (*domain.Account, error) {
	// Get existing account
	account, err := s.accountRepo.GetByID(ctx, orgID, accountID)
	if err != nil {
		return nil, err
	}
	previousStatus := account.Status

	// Last-admin guard: never demote the only remaining admin of the organization.
	if account.Role == "admin" && req.Role != "admin" {
		admins, err := s.countAdmins(ctx, orgID)
		if err != nil {
			return nil, err
		}
		if admins <= 1 {
			return nil, domain.ErrLastAdminDemotion
		}
	}

	// Keep both SSOTs in phase: sync the member role to Stytch BEFORE the local write.
	// On Stytch failure, reject the update so the local row never drifts from the auth provider.
	var stytchOrgID string
	if account.StytchMemberID != "" && orgID > 0 {
		org, orgErr := s.orgRepo.GetByID(ctx, orgID)
		if orgErr != nil {
			return nil, orgErr
		}
		stytchOrgID = org.StytchOrgID
		if stytchOrgID != "" && req.Role != account.Role {
			if err := s.authMemberRepo.AssignRoles(ctx, &domain.AssignAuthRolesRequest{
				OrganizationID: stytchOrgID,
				MemberID:       account.StytchMemberID,
				Roles:          []string{req.Role},
			}); err != nil {
				return nil, fmt.Errorf("failed to sync member role to auth provider: %w", err)
			}
			account.StytchRoleSlug = req.Role
		}
	}

	// Update fields
	account.FullName = req.FullName
	account.Role = req.Role
	account.Status = req.Status
	if req.StytchRoleID != "" {
		account.StytchRoleID = req.StytchRoleID
	}
	if req.StytchRoleSlug != "" {
		account.StytchRoleSlug = req.StytchRoleSlug
	}
	if req.StytchEmailVerified != nil {
		account.StytchEmailVerified = *req.StytchEmailVerified
	}

	updated, err := s.accountRepo.Update(ctx, account)
	if err != nil {
		return nil, err
	}

	// Deactivation revocation: after the local status update succeeds, revoke
	// the member's active Stytch sessions so deactivation takes effect at the
	// identity layer within the JWT window. Best-effort by design: the local
	// status change is authoritative for app access, so a revocation failure
	// logs a warning and carries a pending-revocation notice on the result
	// instead of failing the deactivation.
	if isDeactivationTransition(previousStatus, updated.Status) &&
		stytchOrgID != "" && updated.StytchMemberID != "" {
		if revokeErr := s.sessionRevoker.RevokeMemberSessions(ctx, stytchOrgID, updated.StytchMemberID); revokeErr != nil {
			s.logger.Warn("member deactivated; session revocation deferred", loggerDomain.Fields{
				"org_id":           orgID,
				"account_id":       accountID,
				"stytch_org_id":    stytchOrgID,
				"stytch_member_id": updated.StytchMemberID,
				"error":            revokeErr.Error(),
			})
			updated.SessionRevocationPending = true
		}
	}

	return updated, nil
}

// isDeactivationTransition reports whether an account status change transitions
// an active member into a deactivated (inactive/suspended) state.
func isDeactivationTransition(previousStatus, nextStatus string) bool {
	return previousStatus == "active" && nextStatus != "active"
}

func (s *organizationService) countAdmins(ctx context.Context, orgID int32) (int, error) {
	accounts, err := s.accountRepo.ListByOrganization(ctx, orgID)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, acc := range accounts {
		if acc.Role == "admin" && acc.Status == "active" {
			count++
		}
	}
	return count, nil
}

func (s *organizationService) DeleteAccount(ctx context.Context, orgID, accountID int32) error {
	return s.accountRepo.Delete(ctx, orgID, accountID)
}

func (s *organizationService) UpdateAccountLastLogin(ctx context.Context, orgID, accountID int32) (*domain.Account, error) {
	return s.accountRepo.UpdateLastLogin(ctx, orgID, accountID)
}

func (s *organizationService) CheckAccountPermission(ctx context.Context, orgID, accountID int32) (*domain.AccountPermission, error) {
	return s.accountRepo.CheckPermission(ctx, orgID, accountID)
}

func (s *organizationService) GetAccountStats(ctx context.Context, accountID int32) (*domain.AccountStats, error) {
	return s.accountRepo.GetStats(ctx, accountID)
}
