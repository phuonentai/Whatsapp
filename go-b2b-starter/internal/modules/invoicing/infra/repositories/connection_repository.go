package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/moasq/go-b2b-starter/internal/db/helpers"
	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
	"github.com/moasq/go-b2b-starter/internal/modules/invoicing/domain"
)

type connectionRepository struct{ store sqlc.Store }

func NewConnectionRepository(store sqlc.Store) domain.ConnectionRepository {
	return &connectionRepository{store: store}
}

func (r *connectionRepository) Get(ctx context.Context, orgID int32) (*domain.OrgConnection, error) {
	row, err := r.store.GetOrgConnection(ctx, orgID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrConnectionNotFound
		}
		return nil, fmt.Errorf("failed to get org connection: %w", err)
	}
	return mapConnection(&row), nil
}

func (r *connectionRepository) Upsert(ctx context.Context, conn *domain.OrgConnection) (*domain.OrgConnection, error) {
	row, err := r.store.UpsertOrgConnection(ctx, sqlc.UpsertOrgConnectionParams{
		OrganizationID: conn.OrganizationID,
		Provider:       conn.Provider,
		Status:         string(conn.Status),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to upsert org connection: %w", err)
	}
	return mapConnection(&row), nil
}

func (r *connectionRepository) UpdateStatus(ctx context.Context, orgID int32, status domain.ConnectionStatus, lastError string) (*domain.OrgConnection, error) {
	row, err := r.store.UpdateOrgConnectionStatus(ctx, sqlc.UpdateOrgConnectionStatusParams{
		OrganizationID: orgID,
		Status:         string(status),
		LastError:      helpers.ToPgText(lastError),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrConnectionNotFound
		}
		return nil, fmt.Errorf("failed to update org connection status: %w", err)
	}
	return mapConnection(&row), nil
}

func (r *connectionRepository) UpdateCredentials(ctx context.Context, orgID int32, clientIDEnc, clientSecretEnc []byte, nit, companyName string) (*domain.OrgConnection, error) {
	row, err := r.store.UpdateOrgConnectionCredentials(ctx, sqlc.UpdateOrgConnectionCredentialsParams{
		OrganizationID:  orgID,
		ClientIDEnc:     clientIDEnc,
		ClientSecretEnc: clientSecretEnc,
		Nit:             helpers.ToPgText(nit),
		SiigoCompanyName: helpers.ToPgText(companyName),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrConnectionNotFound
		}
		return nil, fmt.Errorf("failed to update org connection credentials: %w", err)
	}
	return mapConnection(&row), nil
}

func (r *connectionRepository) Delete(ctx context.Context, orgID int32) error {
	if err := r.store.DeleteOrgConnection(ctx, orgID); err != nil {
		return fmt.Errorf("failed to delete org connection: %w", err)
	}
	return nil
}

func (r *connectionRepository) ListByStatus(ctx context.Context, provider string, status domain.ConnectionStatus) ([]*domain.OrgConnection, error) {
	rows, err := r.store.ListOrgConnectionsByStatus(ctx, sqlc.ListOrgConnectionsByStatusParams{
		Provider: provider,
		Status:   string(status),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list org connections by status: %w", err)
	}
	connections := make([]*domain.OrgConnection, len(rows))
	for i := range rows {
		connections[i] = mapConnection(&rows[i])
	}
	return connections, nil
}

func mapConnection(row *sqlc.InvoicingOrgConnection) *domain.OrgConnection {
	conn := &domain.OrgConnection{
		OrganizationID:   row.OrganizationID,
		Provider:         row.Provider,
		Status:           domain.ConnectionStatus(row.Status),
		ClientIDEnc:      row.ClientIDEnc,
		ClientSecretEnc:  row.ClientSecretEnc,
		Nit:              helpers.FromPgText(row.Nit),
		SiigoCompanyName: helpers.FromPgText(row.SiigoCompanyName),
		LastError:        helpers.FromPgText(row.LastError),
		CreatedAt:        row.CreatedAt.Time,
		UpdatedAt:        row.UpdatedAt.Time,
	}
	if row.PausedAt.Valid {
		t := row.PausedAt.Time
		conn.PausedAt = &t
	}
	return conn
}
