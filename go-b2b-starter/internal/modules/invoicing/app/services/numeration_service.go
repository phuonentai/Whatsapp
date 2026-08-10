package services

import (
	"context"
	"fmt"

	"github.com/moasq/go-b2b-starter/internal/modules/invoicing/domain"
	loggerDomain "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
)

// NumerationService handles the numeration continuity step: reading the live
// numeration from the provider and persisting the human-confirmed snapshot,
// which advances the connection from connected to numeracion_ok.
type NumerationService interface {
	GetLive(ctx context.Context, orgID int32) (*domain.NumerationInfo, error)
	Confirm(ctx context.Context, orgID int32) (*domain.NumerationSnapshot, error)
}

type numerationService struct {
	reader   domain.NumerationReader
	repo     domain.NumerationRepository
	connSvc  ConnectionService
	logger   loggerDomain.Logger
}

func NewNumerationService(
	reader domain.NumerationReader,
	repo domain.NumerationRepository,
	connSvc ConnectionService,
	logger loggerDomain.Logger,
) NumerationService {
	return &numerationService{reader: reader, repo: repo, connSvc: connSvc, logger: logger}
}

// GetLive reads the numeration as the provider reports it today. No writes.
func (s *numerationService) GetLive(ctx context.Context, orgID int32) (*domain.NumerationInfo, error) {
	info, err := s.reader.GetNumeration(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to read numeration: %w", err)
	}
	return &info, nil
}

// Confirm stores the snapshot and advances connected → numeracion_ok. A
// failed read leaves the state unchanged and stores nothing.
func (s *numerationService) Confirm(ctx context.Context, orgID int32) (*domain.NumerationSnapshot, error) {
	info, err := s.reader.GetNumeration(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to read numeration before confirmation: %w", err)
	}

	snapshot, err := s.repo.UpsertConfirmed(ctx, &domain.NumerationSnapshot{
		OrganizationID: orgID,
		Mode:           info.Mode,
		ResolutionID:   info.ResolutionID,
		Prefix:         info.Prefix,
		NextNumber:     info.NextNumber,
	})
	if err != nil {
		return nil, err
	}

	if _, err := s.connSvc.ConfirmNumeration(ctx, orgID); err != nil {
		return nil, fmt.Errorf("failed to advance connection after numeration confirmation: %w", err)
	}
	return snapshot, nil
}
