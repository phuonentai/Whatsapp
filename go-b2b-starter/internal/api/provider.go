package api

import (
	"go.uber.org/dig"

	"github.com/moasq/go-b2b-starter/internal/modules/agent"
	"github.com/moasq/go-b2b-starter/internal/modules/auth"
	"github.com/moasq/go-b2b-starter/internal/modules/billing"
	"github.com/moasq/go-b2b-starter/internal/modules/cognitive"
	"github.com/moasq/go-b2b-starter/internal/modules/crm"
	"github.com/moasq/go-b2b-starter/internal/modules/documents"
	"github.com/moasq/go-b2b-starter/internal/modules/invoicing"
	"github.com/moasq/go-b2b-starter/internal/modules/organizations"
	"github.com/moasq/go-b2b-starter/internal/modules/playbooks"
	"github.com/moasq/go-b2b-starter/internal/modules/registry"
	"github.com/moasq/go-b2b-starter/internal/modules/tickets"
	"github.com/moasq/go-b2b-starter/internal/modules/whatsapp"
	server "github.com/moasq/go-b2b-starter/internal/platform/server/domain"
)

type moduleRoutes struct {
	OrganizationRoutes  *organizations.Routes
	RbacRoutes          *auth.Routes
	SubscriptionHandler *billing.Handler
	InvoicingHandler    *invoicing.Handler
	DocumentsRoutes     *documents.Routes
	CognitiveRoutes     *cognitive.Routes
	WhatsAppRoutes      *whatsapp.Routes
	CRMRoutes           *crm.Routes
	RegistryRoutes      *registry.Routes
	TicketsRoutes       *tickets.Routes
	PlaybooksRoutes     *playbooks.Routes
	AgentRoutes         *agent.Routes
}

func Init(container *dig.Container) error {
	if err := setupDependencies(container); err != nil {
		return err
	}
	if err := registerAPI(container); err != nil {
		return err
	}
	return nil
}

func registerAPI(container *dig.Container) error {
	if err := container.Provide(func(
		organizationRoutes *organizations.Routes,
		rbacRoutes *auth.Routes,
		subscriptionHandler *billing.Handler,
		invoicingHandler *invoicing.Handler,
		documentsRoutes *documents.Routes,
		cognitiveRoutes *cognitive.Routes,
		whatsAppRoutes *whatsapp.Routes,
		crmRoutes *crm.Routes,
		registryRoutes *registry.Routes,
		ticketsRoutes *tickets.Routes,
		playbooksRoutes *playbooks.Routes,
		agentRoutes *agent.Routes,
	) *moduleRoutes {
		return &moduleRoutes{
			OrganizationRoutes:  organizationRoutes,
			RbacRoutes:          rbacRoutes,
			SubscriptionHandler: subscriptionHandler,
			InvoicingHandler:    invoicingHandler,
			DocumentsRoutes:     documentsRoutes,
			CognitiveRoutes:     cognitiveRoutes,
			WhatsAppRoutes:      whatsAppRoutes,
			CRMRoutes:           crmRoutes,
			RegistryRoutes:      registryRoutes,
			TicketsRoutes:       ticketsRoutes,
			PlaybooksRoutes:     playbooksRoutes,
			AgentRoutes:         agentRoutes,
		}
	}); err != nil {
		return err
	}

	return container.Invoke(func(
		srv server.Server,
		modules *moduleRoutes,
	) {
		srv.RegisterRoutes(modules.OrganizationRoutes.Routes, server.ApiPrefix)
		srv.RegisterRoutes(modules.RbacRoutes.Routes, server.ApiPrefix)
		srv.RegisterRoutes(modules.SubscriptionHandler.Routes, server.ApiPrefix)
		srv.RegisterRoutes(modules.InvoicingHandler.Routes, server.ApiPrefix)
		srv.RegisterRoutes(modules.DocumentsRoutes.Routes, server.ApiPrefix)
		srv.RegisterRoutes(modules.CognitiveRoutes.Routes, server.ApiPrefix)
		srv.RegisterRoutes(modules.WhatsAppRoutes.Routes, server.ApiPrefix)
		srv.RegisterRoutes(modules.CRMRoutes.Routes, server.ApiPrefix)
		srv.RegisterRoutes(modules.RegistryRoutes.Routes, server.ApiPrefix)
		srv.RegisterRoutes(modules.TicketsRoutes.Routes, server.ApiPrefix)
		srv.RegisterRoutes(modules.PlaybooksRoutes.Routes, server.ApiPrefix)
		srv.RegisterRoutes(modules.AgentRoutes.Routes, server.ApiPrefix)
	})
}

func setupDependencies(container *dig.Container) error {
	if err := organizations.NewProvider(container).RegisterDependencies(); err != nil {
		return err
	}
	if err := auth.NewProvider(container).RegisterDependencies(); err != nil {
		return err
	}
	if err := billing.RegisterHandlers(container); err != nil {
		return err
	}
	if err := documents.NewProvider(container).RegisterDependencies(); err != nil {
		return err
	}
	if err := cognitive.NewProvider(container).RegisterDependencies(); err != nil {
		return err
	}
	if err := crm.NewProvider(container).RegisterDependencies(); err != nil {
		return err
	}
	// Registry module is registered in bootstrap/init_mods.go before billing
	// (BillingService depends on ModuleService); do NOT re-register here.
	if err := tickets.NewProvider(container).RegisterDependencies(); err != nil {
		return err
	}
	// Playbooks module (registered in bootstrap before billing; kept in sync
	// here for API wiring only — do NOT re-register dependencies here).
	return nil
}
