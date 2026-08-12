package cmd

import (
	"context"
	"fmt"

	"go.uber.org/dig"

	"github.com/moasq/go-b2b-starter/internal/modules/agent"
	"github.com/moasq/go-b2b-starter/internal/modules/agent/app/services"
	igEvents "github.com/moasq/go-b2b-starter/internal/modules/instagram/domain/events"
	whatsappEvents "github.com/moasq/go-b2b-starter/internal/modules/whatsapp/domain/events"
	"github.com/moasq/go-b2b-starter/internal/platform/eventbus"
	"github.com/moasq/go-b2b-starter/pkg/whatsapp"
)

// subscriptionParams collects the optional ActiveInquiryChecker provided by
// the procurement module (registered before this module at bootstrap). When
// absent (e.g., agent-only wiring), the skip check is disabled.
type subscriptionParams struct {
	dig.In

	Checker agent.ActiveInquiryChecker `optional:"true"`
}

// Init registers agent dependencies and subscribes the agent pipeline to
// inbound WhatsApp message events. Messages consumed by an active supplier
// inquiry run skip the analysis pipeline entirely (no flow, no suggestion);
// the procurement subscriber processes them independently.
func Init(container *dig.Container) error {
	module := agent.NewModule(container)
	if err := module.RegisterDependencies(); err != nil {
		return fmt.Errorf("failed to register agent dependencies: %w", err)
	}

	provider := agent.NewProvider(container)
	if err := provider.RegisterDependencies(); err != nil {
		return fmt.Errorf("failed to register agent provider: %w", err)
	}

	if err := container.Invoke(func(
		bus eventbus.EventBus,
		agentService services.AgentService,
		params subscriptionParams,
	) error {
		if err := bus.Subscribe(whatsappEvents.MessageReceivedEventType, func(ctx context.Context, event eventbus.Event) error {
			msgEvent, ok := event.(*whatsappEvents.MessageReceived)
			if !ok {
				return fmt.Errorf("unexpected event type: %T", event)
			}
			if params.Checker != nil {
				skip, err := shouldSkipInquiry(ctx, params.Checker, msgEvent)
				if err != nil {
					// Fail-safe: on lookup errors do NOT skip; the procurement
					// subscriber still extracts the reply independently.
					return agentService.HandleMessageReceived(ctx, msgEvent)
				}
				if skip {
					return nil
				}
			}
			return agentService.HandleMessageReceived(ctx, msgEvent)
		}); err != nil {
			return err
		}
		return bus.Subscribe(igEvents.MessageReceivedEventType, func(ctx context.Context, event eventbus.Event) error {
			msgEvent, ok := event.(*igEvents.MessageReceived)
			if !ok {
				return fmt.Errorf("unexpected event type: %T", event)
			}
			return agentService.HandleInstagramMessageReceived(ctx, msgEvent)
		})
	}); err != nil {
		return fmt.Errorf("failed to subscribe to inbound message events: %w", err)
	}

	return nil
}

// shouldSkipInquiry resolves the sender phone (tenant-scoped) against active
// inquiry-run recipients.
func shouldSkipInquiry(ctx context.Context, checker agent.ActiveInquiryChecker, event *whatsappEvents.MessageReceived) (bool, error) {
	if event.MessageType != "text" || event.Content == "" {
		return false, nil
	}
	phone, err := whatsapp.CanonicalizeE164(event.From)
	if err != nil {
		return false, nil
	}
	return checker.IsActiveRecipientByPhone(ctx, event.OrganizationID, phone)
}
