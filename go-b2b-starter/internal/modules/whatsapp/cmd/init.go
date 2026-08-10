package cmd

import (
	"encoding/json"
	"fmt"

	"go.uber.org/dig"

	"github.com/moasq/go-b2b-starter/internal/modules/whatsapp"
	"github.com/moasq/go-b2b-starter/internal/modules/whatsapp/domain/events"
	"github.com/moasq/go-b2b-starter/internal/modules/whatsapp/infra/graphapi"
	"github.com/moasq/go-b2b-starter/internal/platform/eventbus"
	"github.com/moasq/go-b2b-starter/internal/platform/outbox"
)

func Init(container *dig.Container) error {
	// Validated environment invariants: missing required Meta vars fail loudly
	// at startup so misconfiguration never reaches the signup flow.
	cfg, metaCfg, err := graphapi.FromEnv()
	if err != nil {
		return fmt.Errorf("whatsapp graph api config: %w", err)
	}

	if err := container.Provide(func() graphapi.Client {
		return graphapi.NewClient(cfg, nil)
	}); err != nil {
		return fmt.Errorf("failed to provide graph api client: %w", err)
	}
	if err := container.Provide(func() graphapi.MetaConfig {
		return metaCfg
	}); err != nil {
		return fmt.Errorf("failed to provide meta config: %w", err)
	}

	module := whatsapp.NewModule(container)
	if err := module.RegisterDependencies(); err != nil {
		return fmt.Errorf("failed to register whatsapp dependencies: %w", err)
	}

	provider := whatsapp.NewProvider(container)
	if err := provider.RegisterDependencies(); err != nil {
		return fmt.Errorf("failed to register whatsapp provider: %w", err)
	}

	if err := registerOutboxCodecs(container); err != nil {
		return fmt.Errorf("failed to register whatsapp outbox codecs: %w", err)
	}

	return nil
}

// registerOutboxCodecs binds deserializers so the outbox dispatcher can
// reconstruct typed WhatsApp events from stored payloads.
func registerOutboxCodecs(container *dig.Container) error {
	return container.Invoke(func(registry *outbox.Registry) {
		registry.Register(events.MessageReceivedEventType, func(payload json.RawMessage) (eventbus.Event, error) {
			var e events.MessageReceived
			if err := json.Unmarshal(payload, &e); err != nil {
				return nil, fmt.Errorf("decode %s: %w", events.MessageReceivedEventType, err)
			}
			return &e, nil
		})
		registry.Register(events.MessageEchoEventType, func(payload json.RawMessage) (eventbus.Event, error) {
			var e events.MessageEcho
			if err := json.Unmarshal(payload, &e); err != nil {
				return nil, fmt.Errorf("decode %s: %w", events.MessageEchoEventType, err)
			}
			return &e, nil
		})
		registry.Register(events.MessageSendEventType, func(payload json.RawMessage) (eventbus.Event, error) {
			var e events.MessageSend
			if err := json.Unmarshal(payload, &e); err != nil {
				return nil, fmt.Errorf("decode %s: %w", events.MessageSendEventType, err)
			}
			return &e, nil
		})
	})
}
