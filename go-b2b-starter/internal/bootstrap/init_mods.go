package bootstrap

import (
	"context"
	"fmt"

	"go.uber.org/dig"

	"github.com/moasq/go-b2b-starter/internal/api"
	"github.com/moasq/go-b2b-starter/internal/modules/agent/cmd"
	"github.com/moasq/go-b2b-starter/internal/modules/auth"
	authCmd "github.com/moasq/go-b2b-starter/internal/modules/auth/cmd"
	billing "github.com/moasq/go-b2b-starter/internal/modules/billing/cmd"
	cognitive "github.com/moasq/go-b2b-starter/internal/modules/cognitive/cmd"
	crm "github.com/moasq/go-b2b-starter/internal/modules/crm/cmd"
	db "github.com/moasq/go-b2b-starter/internal/db/cmd"
	docs "github.com/moasq/go-b2b-starter/internal/docs/cmd"
	documents "github.com/moasq/go-b2b-starter/internal/modules/documents/cmd"
	eventbus "github.com/moasq/go-b2b-starter/internal/platform/eventbus/cmd"
	files "github.com/moasq/go-b2b-starter/internal/modules/files/cmd"
	invoicing "github.com/moasq/go-b2b-starter/internal/modules/invoicing/cmd"
	llm "github.com/moasq/go-b2b-starter/internal/platform/llm/cmd"
	logger "github.com/moasq/go-b2b-starter/internal/platform/logger/cmd"
	mercadopago "github.com/moasq/go-b2b-starter/internal/platform/mercadopago/cmd"
	ocr "github.com/moasq/go-b2b-starter/internal/platform/ocr/cmd"
	orgDomain "github.com/moasq/go-b2b-starter/internal/modules/organizations/domain"
	organizations "github.com/moasq/go-b2b-starter/internal/modules/organizations/cmd"
	paywall "github.com/moasq/go-b2b-starter/internal/modules/paywall/cmd"
	playbooks "github.com/moasq/go-b2b-starter/internal/modules/playbooks"
	registry "github.com/moasq/go-b2b-starter/internal/modules/registry"
	polar "github.com/moasq/go-b2b-starter/internal/platform/polar/cmd"
	redisCmd "github.com/moasq/go-b2b-starter/internal/platform/redis/cmd"
	server "github.com/moasq/go-b2b-starter/internal/platform/server/cmd"
	stytchCmd "github.com/moasq/go-b2b-starter/internal/platform/stytch/cmd"
	whatsapp "github.com/moasq/go-b2b-starter/internal/modules/whatsapp/cmd"
)

// orgLookupAdapter adapts orgDomain.OrganizationRepository to auth.OrganizationLookup
type orgLookupAdapter struct {
	repo orgDomain.OrganizationRepository
}

func (a *orgLookupAdapter) GetByStytchID(ctx context.Context, stytchOrgID string) (auth.OrganizationEntity, error) {
	return a.repo.GetByStytchID(ctx, stytchOrgID)
}

// accLookupAdapter adapts orgDomain.AccountRepository to auth.AccountLookup
type accLookupAdapter struct {
	repo orgDomain.AccountRepository
}

func (a *accLookupAdapter) GetByEmail(ctx context.Context, orgID int32, email string) (auth.AccountEntity, error) {
	return a.repo.GetByEmail(ctx, orgID, email)
}

func InitMods(container *dig.Container) {

	// pkg
	server.Init(container)
	logger.Init(container)
	db.Init(container)
	files.Init(container)
	if err := eventbus.Init(container); err != nil {
		panic(err)
	}
	if err := llm.Init(container); err != nil {
		panic(err)
	}

	// Polar package must be initialized before payment module (payment depends on Polar client)
	if err := polar.Init(container); err != nil {
		panic(err)
	}

	// MercadoPago package must be initialized before billing module (billing depends on MP client)
	if err := mercadopago.Init(container); err != nil {
		panic(err)
	}

	// Redis must be initialized before auth (Stytch repositories rely on Redis-backed clients upstream)
	if err := redisCmd.Init(container); err != nil {
		panic(err)
	}

	// Stytch client package must be initialized before app/auth (for organization/member management)
	// This provides: stytch.Config, stytch.Client, stytch.RBACPolicyService
	if err := stytchCmd.ProvideStytchDependencies(container); err != nil {
		panic(err)
	}

	// Auth package (pkg/auth) must be initialized before app/auth
	// This provides: auth.AuthProvider (authentication/authorization)
	if err := authCmd.Init(container); err != nil {
		panic(err)
	}

	// docs
	docs.Init(container)

	// app
	if err := organizations.Init(container); err != nil {
		panic(err)
	}

	// Register auth resolvers (bridges organizations domain to auth package)
	if err := auth.ProvideResolvers(container,
		func(repo orgDomain.OrganizationRepository) auth.OrganizationResolver {
			return auth.NewOrganizationResolver(&orgLookupAdapter{repo: repo})
		},
		func(repo orgDomain.AccountRepository) auth.AccountResolver {
			return auth.NewAccountResolver(&accLookupAdapter{repo: repo})
		},
	); err != nil {
		panic(err)
	}

	// Initialize auth middleware (requires resolvers to be registered)
	if err := authCmd.InitMiddleware(container); err != nil {
		panic(err)
	}

	// Register auth middleware as named middlewares for use in routes
	if err := auth.RegisterNamedMiddlewares(container); err != nil {
		panic(err)
	}

	// Billing module (subscription lifecycle, quotas, webhooks).
	// Registry module services must be registered before billing because
	// BillingService depends on ModuleService.
	if err := registry.NewProvider(container).RegisterDependencies(); err != nil {
		panic(err)
	}
	// Playbooks depend on registry ModuleService (config preset validation).
	if err := playbooks.NewProvider(container).RegisterDependencies(); err != nil {
		panic(err)
	}
	if err := billing.Init(container); err != nil {
		panic(err)
	}

	// Paywall middleware (access gating based on subscription status)
	if err := paywall.SetupMiddleware(container); err != nil {
		panic(err)
	}
	if err := paywall.RegisterNamedMiddlewares(container); err != nil {
		panic(err)
	}

	// OCR service (Mistral API for document text extraction)
	// Must be initialized before documents module (documents depends on OCR)
	if err := ocr.Init(container); err != nil {
		panic(err)
	}

	// Documents module (PDF upload and text extraction)
	if err := documents.Init(container); err != nil {
		panic(err)
	}

	// Cognitive module (AI/RAG with embeddings and vector search)
	// Note: This also wires the event listener for DocumentUploaded events
	if err := cognitive.Init(container); err != nil {
		panic(err)
	}

	// WhatsApp module (webhook ingress + config). Must precede crm: the CRM
	// outbound service depends on the WhatsApp config repository.
	if err := whatsapp.Init(container); err != nil {
		panic(err)
	}

	// CRM module (contacts/conversations/messages + outbound). Must precede
	// agent: the agent pipeline depends on the CRM outbound service. This
	// also wires the whatsapp.message.received CRM listener.
	if err := crm.Init(container); err != nil {
		panic(err)
	}

	// Agent module (agentic WhatsApp assistant). Subscribes to the same
	// whatsapp.message.received event alongside the CRM listener.
	if err := cmd.Init(container); err != nil {
		panic(err)
	}

	// Invoicing module (Siigo electronic invoicing). Depends on CRM repos and
	// the outbound send seam; subscribes to crm deal stage changes.
	if err := invoicing.Init(container); err != nil {
		panic(err)
	}

	// api
	if err := api.Init(container); err != nil {
		panic(fmt.Sprintf("api init failed: %v", err))
	}
}
