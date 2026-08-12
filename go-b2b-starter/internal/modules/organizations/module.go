package organizations

import (
	"go.uber.org/dig"

	"github.com/moasq/go-b2b-starter/internal/modules/organizations/app/services"
	"github.com/moasq/go-b2b-starter/internal/modules/organizations/domain"
	"github.com/moasq/go-b2b-starter/internal/modules/organizations/infra/repositories"
	loggerDomain "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
	stytchcfg "github.com/moasq/go-b2b-starter/internal/platform/stytch"
)

// Module provides organization module dependencies
type Module struct {
	container *dig.Container
}

func NewModule(container *dig.Container) *Module {
	return &Module{
		container: container,
	}
}

// RegisterDependencies registers all organization module dependencies
// Note: Repository implementations are registered in internal/db/inject.go
func (m *Module) RegisterDependencies() error {
	// Register auth provider repositories (Stytch implementation)
	if err := m.container.Provide(func(
		client *stytchcfg.Client,
		logger loggerDomain.Logger,
		localOrgRepo domain.OrganizationRepository,
	) domain.AuthOrganizationRepository {
		return repositories.NewStytchOrganizationRepository(client, logger, localOrgRepo)
	}); err != nil {
		return err
	}

	if err := m.container.Provide(func(
		client *stytchcfg.Client,
		cfg *stytchcfg.Config,
		logger loggerDomain.Logger,
	) domain.AuthMemberRepository {
		return repositories.NewStytchMemberRepository(client, *cfg, logger)
	}); err != nil {
		return err
	}

	// Session revocation (member deactivation): the same Stytch member
	// repository implements the SessionRevoker domain contract. The shared
	// *stytchcfg.Client singleton carries the circuit breaker.
	if err := m.container.Provide(func(
		client *stytchcfg.Client,
		cfg *stytchcfg.Config,
		logger loggerDomain.Logger,
	) domain.SessionRevoker {
		return repositories.NewStytchMemberRepository(client, *cfg, logger)
	}); err != nil {
		return err
	}

	if err := m.container.Provide(func(
		client *stytchcfg.Client,
		logger loggerDomain.Logger,
	) domain.AuthRoleRepository {
		return repositories.NewStytchRoleRepository(client, logger)
	}); err != nil {
		return err
	}

	// MFA policy updates: the same Stytch organization repository implements
	// the MfaPolicyUpdater domain contract. The shared *stytchcfg.Client
	// singleton carries the circuit breaker.
	if err := m.container.Provide(func(
		client *stytchcfg.Client,
		logger loggerDomain.Logger,
		localOrgRepo domain.OrganizationRepository,
	) domain.MfaPolicyUpdater {
		return repositories.NewStytchOrganizationRepository(client, logger, localOrgRepo)
	}); err != nil {
		return err
	}

	// Org auth policy (JIT / allowed auth methods / SSO JIT): the same Stytch
	// organization repository implements the OrgAuthPolicyUpdater domain
	// contract. The shared *stytchcfg.Client singleton carries the circuit
	// breaker.
	if err := m.container.Provide(func(
		client *stytchcfg.Client,
		logger loggerDomain.Logger,
		localOrgRepo domain.OrganizationRepository,
	) domain.OrgAuthPolicyUpdater {
		return repositories.NewStytchOrganizationRepository(client, logger, localOrgRepo)
	}); err != nil {
		return err
	}

	// Register organization service
	if err := m.container.Provide(func(
		orgRepo domain.OrganizationRepository,
		accountRepo domain.AccountRepository,
		authOrgRepo domain.AuthOrganizationRepository,
		authMemberRepo domain.AuthMemberRepository,
		sessionRevoker domain.SessionRevoker,
		mfaPolicyUpdater domain.MfaPolicyUpdater,
		authPolicyUpdater domain.OrgAuthPolicyUpdater,
		logger loggerDomain.Logger,
	) services.OrganizationService {
		return services.NewOrganizationService(orgRepo, accountRepo, authOrgRepo, authMemberRepo, sessionRevoker, mfaPolicyUpdater, authPolicyUpdater, logger)
	}); err != nil {
		return err
	}

	// Register member service (for auth member operations)
	if err := m.container.Provide(func(
		authOrgRepo domain.AuthOrganizationRepository,
		authMemberRepo domain.AuthMemberRepository,
		authRoleRepo domain.AuthRoleRepository,
		localOrgRepo domain.OrganizationRepository,
		localAccountRepo domain.AccountRepository,
		trialSeeder domain.TrialSeeder,
		logger loggerDomain.Logger,
	) services.MemberService {
		return services.NewMemberService(
			authOrgRepo,
			authMemberRepo,
			authRoleRepo,
			localOrgRepo,
			localAccountRepo,
			trialSeeder,
			logger,
		)
	}); err != nil {
		return err
	}

	return nil
}
