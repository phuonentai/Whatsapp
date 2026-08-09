package cmd

import (
	"go.uber.org/dig"

	"github.com/moasq/go-b2b-starter/internal/platform/llm/domain"
	"github.com/moasq/go-b2b-starter/internal/platform/llm/infra"
	loggerDomain "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
)

// meteredClientParams collects the optional TokenLedger. When the billing
// module is initialized (it always is in the real app), the ledger is present
// and the client is wrapped; otherwise the raw client is used.
type meteredClientParams struct {
	dig.In

	Ledger domain.TokenLedger `optional:"true"`
}

func Init(container *dig.Container) error {
	// Register LLMClient (which includes LLMService), wrapped with usage
	// metering when a TokenLedger is available.
	if err := container.Provide(func(p meteredClientParams, logger loggerDomain.Logger) (domain.LLMClient, error) {
		config := infra.NewLLMConfig()
		client, err := infra.NewOpenAIClient(config, logger)
		if err != nil {
			return nil, err
		}
		if p.Ledger != nil {
			return infra.NewMeteredLLMClient(client, p.Ledger, logger), nil
		}
		return client, nil
	}); err != nil {
		return err
	}

	// Also register LLMService for backward compatibility
	return container.Provide(func(client domain.LLMClient) domain.LLMService {
		return client
	})
}
