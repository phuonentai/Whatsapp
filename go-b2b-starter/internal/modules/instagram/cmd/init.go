package cmd

import (
	"encoding/json"
	"fmt"

	"go.uber.org/dig"

	"github.com/moasq/go-b2b-starter/internal/modules/instagram"
	"github.com/moasq/go-b2b-starter/internal/modules/instagram/domain/events"
	"github.com/moasq/go-b2b-starter/internal/modules/instagram/infra/graphapi"
	"github.com/moasq/go-b2b-starter/internal/platform/eventbus"
	"github.com/moasq/go-b2b-starter/internal/platform/outbox"
)

func Init(container *dig.Container) error {
	cfg := graphapi.FromEnv()

	if err := container.Provide(func() graphapi.ClientConfig {
		return cfg
	}); err != nil {
		return fmt.Errorf("failed to provide instagram graph api config: %w", err)
	}
	if err := container.Provide(func() graphapi.IGClient {
		return graphapi.NewIGClient(cfg, nil)
	}); err != nil {
		return fmt.Errorf("failed to provide instagram graph api client: %w", err)
	}

	module := instagram.NewModule(container)
	if err := module.RegisterDependencies(); err != nil {
		return fmt.Errorf("failed to register instagram dependencies: %w", err)
	}

	provider := instagram.NewProvider(container)
	if err := provider.RegisterDependencies(); err != nil {
		return fmt.Errorf("failed to register instagram provider: %w", err)
	}

	if err := registerOutboxCodecs(container); err != nil {
		return fmt.Errorf("failed to register instagram outbox codecs: %w", err)
	}

	return nil
}

// registerOutboxCodecs binds deserializers so the outbox dispatcher can
// reconstruct typed Instagram events from stored payloads.
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
		registry.Register(events.ProfileBackfillEventType, func(payload json.RawMessage) (eventbus.Event, error) {
			var e events.ProfileBackfill
			if err := json.Unmarshal(payload, &e); err != nil {
				return nil, fmt.Errorf("decode %s: %w", events.ProfileBackfillEventType, err)
			}
			return &e, nil
		})
	})
}
