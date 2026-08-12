package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/moasq/go-b2b-starter/internal/db/helpers"
	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
	"github.com/moasq/go-b2b-starter/internal/modules/procurement/domain"
)

func mapRun(row sqlc.ProcurementInquiryRun) *domain.InquiryRun {
	return &domain.InquiryRun{
		ID:                row.ID,
		OrganizationID:    row.OrganizationID,
		Status:            domain.RunStatus(row.Status),
		Source:            row.Source,
		ScheduleRef:       pgInt8Ptr(row.ScheduleRef),
		Nota:              helpers.FromPgTextPtr(row.Nota),
		CreatedByMemberID: helpers.FromPgTextPtr(row.CreatedByMemberID),
		SentAt:            helpers.FromPgTimestamptzPtr(row.SentAt),
		CompletedAt:       helpers.FromPgTimestamptzPtr(row.CompletedAt),
		CreatedAt:         row.CreatedAt.Time,
		UpdatedAt:         row.UpdatedAt.Time,
	}
}

func mapRecipient(row sqlc.ProcurementInquiryRecipient) *domain.InquiryRecipient {
	return &domain.InquiryRecipient{
		ID:                row.ID,
		OrganizationID:    row.OrganizationID,
		RunID:             row.RunID,
		SupplierID:        row.SupplierID,
		ContactID:         row.ContactID,
		DraftedMessage:    helpers.FromPgTextPtr(row.DraftedMessage),
		Status:            domain.RecipientStatus(row.Status),
		ProviderMessageID: helpers.FromPgTextPtr(row.ProviderMessageID),
		SentAt:            helpers.FromPgTimestamptzPtr(row.SentAt),
		AnsweredAt:        helpers.FromPgTimestamptzPtr(row.AnsweredAt),
		FollowupCount:     row.FollowupCount,
		CreatedAt:         row.CreatedAt.Time,
		UpdatedAt:         row.UpdatedAt.Time,
	}
}

func mapResponse(row sqlc.ProcurementInquiryResponse) (*domain.InquiryResponse, error) {
	items := []domain.ResponseItem{}
	if len(row.Extracted) > 0 {
		if err := json.Unmarshal(row.Extracted, &items); err != nil {
			return nil, fmt.Errorf("decode extracted response: %w", err)
		}
	}
	return &domain.InquiryResponse{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
		RecipientID:    row.RecipientID,
		RawMessageID:   row.RawMessageID,
		Items:          items,
		Resumen:        helpers.FromPgText(row.Resumen),
		Confidence:     pgFloat8Ptr(row.Confidence),
		RequiereHumano: row.RequiereHumano,
		CreatedAt:      row.CreatedAt.Time,
	}, nil
}

type runRepository struct {
	store sqlc.Store
}

// NewRunRepository builds the run/recipient/response repository.
func NewRunRepository(store sqlc.Store) domain.InquiryRunRepository {
	return &runRepository{store: store}
}

func (r *runRepository) CreateRun(ctx context.Context, orgID int32, nota *string, memberID string) (*domain.InquiryRun, error) {
	row, err := r.store.CreateInquiryRun(ctx, sqlc.CreateInquiryRunParams{
		OrganizationID:    orgID,
		Nota:              helpers.ToPgTextPtr(nota),
		CreatedByMemberID: helpers.ToPgTextPtr(strPtr(memberID)),
	})
	if err != nil {
		return nil, err
	}
	return mapRun(row), nil
}

func (r *runRepository) GetRun(ctx context.Context, orgID, runID int32) (*domain.InquiryRun, error) {
	row, err := r.store.GetInquiryRun(ctx, sqlc.GetInquiryRunParams{ID: runID, OrganizationID: orgID})
	if err != nil {
		if isNoRows(err) {
			return nil, domain.ErrRunNotFound
		}
		return nil, err
	}
	return mapRun(row), nil
}

func (r *runRepository) ListRuns(ctx context.Context, orgID int32, limit, offset int32) ([]*domain.InquiryRun, error) {
	rows, err := r.store.ListInquiryRunsByOrganization(ctx, sqlc.ListInquiryRunsByOrganizationParams{
		OrganizationID: orgID,
		Limit:          limit,
		Offset:         offset,
	})
	if err != nil {
		return nil, err
	}
	out := make([]*domain.InquiryRun, 0, len(rows))
	for i := range rows {
		out = append(out, mapRun(rows[i]))
	}
	return out, nil
}

func (r *runRepository) TransitionRun(ctx context.Context, orgID, runID int32, from, to domain.RunStatus) (*domain.InquiryRun, error) {
	row, err := r.store.UpdateRunStatusFrom(ctx, sqlc.UpdateRunStatusFromParams{
		ID:             runID,
		OrganizationID: orgID,
		Status:         string(from),
		Status_2:       string(to),
	})
	if isNoRows(err) {
		return nil, domain.ErrInvalidTransition
	}
	if err != nil {
		return nil, err
	}
	return mapRun(row), nil
}

func (r *runRepository) CreateRecipient(ctx context.Context, orgID, runID, supplierID, contactID int32, draftedMessage *string) (*domain.InquiryRecipient, error) {
	row, err := r.store.CreateInquiryRecipient(ctx, sqlc.CreateInquiryRecipientParams{
		OrganizationID: orgID,
		RunID:          runID,
		SupplierID:     supplierID,
		ContactID:      contactID,
		DraftedMessage: helpers.ToPgTextPtr(draftedMessage),
	})
	if err != nil {
		return nil, err
	}
	return mapRecipient(row), nil
}

func (r *runRepository) GetRecipient(ctx context.Context, orgID, recipientID int32) (*domain.InquiryRecipient, error) {
	row, err := r.store.GetInquiryRecipient(ctx, sqlc.GetInquiryRecipientParams{ID: recipientID, OrganizationID: orgID})
	if err != nil {
		if isNoRows(err) {
			return nil, domain.ErrRecipientNotFound
		}
		return nil, err
	}
	return mapRecipient(row), nil
}

func (r *runRepository) ListRunRecipients(ctx context.Context, orgID, runID int32) ([]*domain.InquiryRecipient, error) {
	rows, err := r.store.ListRunRecipients(ctx, sqlc.ListRunRecipientsParams{RunID: runID, OrganizationID: orgID})
	if err != nil {
		return nil, err
	}
	out := make([]*domain.InquiryRecipient, 0, len(rows))
	for i := range rows {
		out = append(out, mapRecipient(rows[i]))
	}
	return out, nil
}

func (r *runRepository) ListSuppliersWithDisplay(ctx context.Context, orgID int32, ids []int32) ([]domain.SupplierWithDisplay, error) {
	if len(ids) == 0 {
		return []domain.SupplierWithDisplay{}, nil
	}
	rows, err := r.store.ListSuppliersWithDisplay(ctx, sqlc.ListSuppliersWithDisplayParams{
		OrganizationID: orgID,
		Column2:        ids,
	})
	if err != nil {
		return nil, err
	}
	out := make([]domain.SupplierWithDisplay, 0, len(rows))
	for i := range rows {
		base := sqlc.ProcurementSupplier{
			ID:             rows[i].ID,
			OrganizationID: rows[i].OrganizationID,
			ContactID:      rows[i].ContactID,
			Nit:            rows[i].Nit,
			DeliveryDays:   rows[i].DeliveryDays,
			MinOrderAmount: rows[i].MinOrderAmount,
			Notes:          rows[i].Notes,
			IsActive:       rows[i].IsActive,
			CreatedAt:      rows[i].CreatedAt,
			UpdatedAt:      rows[i].UpdatedAt,
		}
		out = append(out, domain.SupplierWithDisplay{
			Supplier:    mapSupplier(base),
			DisplayName: rows[i].DisplayName,
		})
	}
	return out, nil
}

func (r *runRepository) ListRunRecipientsWithPhone(ctx context.Context, orgID, runID int32) ([]domain.RecipientWithPhone, error) {
	rows, err := r.store.ListRunRecipientsWithPhone(ctx, sqlc.ListRunRecipientsWithPhoneParams{RunID: runID, OrganizationID: orgID})
	if err != nil {
		return nil, err
	}
	out := make([]domain.RecipientWithPhone, 0, len(rows))
	for i := range rows {
		base := sqlc.ProcurementInquiryRecipient{
			ID:                rows[i].ID,
			OrganizationID:    rows[i].OrganizationID,
			RunID:             rows[i].RunID,
			SupplierID:        rows[i].SupplierID,
			ContactID:         rows[i].ContactID,
			DraftedMessage:    rows[i].DraftedMessage,
			Status:            rows[i].Status,
			ProviderMessageID: rows[i].ProviderMessageID,
			SentAt:            rows[i].SentAt,
			AnsweredAt:        rows[i].AnsweredAt,
			FollowupCount:     rows[i].FollowupCount,
			CreatedAt:         rows[i].CreatedAt,
			UpdatedAt:         rows[i].UpdatedAt,
		}
		out = append(out, domain.RecipientWithPhone{
			Recipient:    mapRecipient(base),
			ContactPhone: rows[i].ContactPhone.String,
		})
	}
	return out, nil
}

func (r *runRepository) MarkRecipientSent(ctx context.Context, orgID, recipientID int32, providerMessageID string) (*domain.InquiryRecipient, error) {
	row, err := r.store.MarkRecipientSent(ctx, sqlc.MarkRecipientSentParams{
		ID:                recipientID,
		OrganizationID:    orgID,
		ProviderMessageID: helpers.ToPgText(providerMessageID),
	})
	if isNoRows(err) {
		return nil, domain.ErrRecipientNotFound
	}
	if err != nil {
		return nil, err
	}
	return mapRecipient(row), nil
}

func (r *runRepository) MarkRecipientAnswered(ctx context.Context, orgID, recipientID int32) (*domain.InquiryRecipient, error) {
	row, err := r.store.MarkRecipientAnswered(ctx, sqlc.MarkRecipientAnsweredParams{ID: recipientID, OrganizationID: orgID})
	if isNoRows(err) {
		return nil, domain.ErrRecipientNotFound
	}
	if err != nil {
		return nil, err
	}
	return mapRecipient(row), nil
}

func (r *runRepository) MarkRecipientTimedOut(ctx context.Context, orgID, recipientID int32) (*domain.InquiryRecipient, error) {
	row, err := r.store.MarkRecipientTimedOut(ctx, sqlc.MarkRecipientTimedOutParams{ID: recipientID, OrganizationID: orgID})
	if isNoRows(err) {
		return nil, domain.ErrRecipientNotFound
	}
	if err != nil {
		return nil, err
	}
	return mapRecipient(row), nil
}

func (r *runRepository) MarkRecipientFailed(ctx context.Context, orgID, recipientID int32) (*domain.InquiryRecipient, error) {
	row, err := r.store.MarkRecipientFailed(ctx, sqlc.MarkRecipientFailedParams{ID: recipientID, OrganizationID: orgID})
	if isNoRows(err) {
		return nil, domain.ErrRecipientNotFound
	}
	if err != nil {
		return nil, err
	}
	return mapRecipient(row), nil
}

func (r *runRepository) ListActiveRecipientsByPhone(ctx context.Context, orgID int32, phoneNumber string) ([]*domain.InquiryRecipient, error) {
	rows, err := r.store.ListActiveRecipientsByPhone(ctx, sqlc.ListActiveRecipientsByPhoneParams{
		OrganizationID: orgID,
		PhoneNumber:    helpers.ToPgText(phoneNumber),
	})
	if err != nil {
		return nil, err
	}
	out := make([]*domain.InquiryRecipient, 0, len(rows))
	for i := range rows {
		out = append(out, mapRecipient(rows[i]))
	}
	return out, nil
}

func (r *runRepository) ListAwaitingRecipients(ctx context.Context, orgID, runID int32) ([]*domain.InquiryRecipient, error) {
	rows, err := r.store.ListAwaitingRecipients(ctx, sqlc.ListAwaitingRecipientsParams{RunID: runID, OrganizationID: orgID})
	if err != nil {
		return nil, err
	}
	out := make([]*domain.InquiryRecipient, 0, len(rows))
	for i := range rows {
		out = append(out, mapRecipient(rows[i]))
	}
	return out, nil
}

func (r *runRepository) ListExpiredSentRecipients(ctx context.Context, orgID, runID int32, windowHours int32) ([]*domain.InquiryRecipient, error) {
	rows, err := r.store.ListExpiredSentRecipients(ctx, sqlc.ListExpiredSentRecipientsParams{
		OrganizationID: orgID,
		RunID:          runID,
		Column3:        windowHours,
	})
	if err != nil {
		return nil, err
	}
	out := make([]*domain.InquiryRecipient, 0, len(rows))
	for i := range rows {
		out = append(out, mapRecipient(rows[i]))
	}
	return out, nil
}

func (r *runRepository) SaveResponse(ctx context.Context, resp *domain.InquiryResponse) (*domain.InquiryResponse, error) {
	extracted, err := json.Marshal(resp.Items)
	if err != nil {
		return nil, err
	}
	if resp.Items == nil {
		extracted = []byte(`[]`)
	}
	row, err := r.store.CreateInquiryResponse(ctx, sqlc.CreateInquiryResponseParams{
		OrganizationID: resp.OrganizationID,
		RecipientID:    resp.RecipientID,
		RawMessageID:   resp.RawMessageID,
		Extracted:      extracted,
		Resumen:        helpers.ToPgText(resp.Resumen),
		Confidence:     toPgFloat8(resp.Confidence),
		RequiereHumano: resp.RequiereHumano,
	})
	if isNoRows(err) {
		// ON CONFLICT DO NOTHING on a duplicate returns no row: no-op.
		return nil, domain.ErrDuplicateResponse
	}
	if err != nil {
		return nil, err
	}
	return mapResponse(row)
}

func (r *runRepository) GetResponseByRecipientMessage(ctx context.Context, recipientID int32, rawMessageID string) (*domain.InquiryResponse, error) {
	row, err := r.store.GetResponseByRecipientMessage(ctx, sqlc.GetResponseByRecipientMessageParams{
		RecipientID:  recipientID,
		RawMessageID: rawMessageID,
	})
	if isNoRows(err) {
		return nil, domain.ErrResponseNotFound
	}
	if err != nil {
		return nil, err
	}
	return mapResponse(row)
}

func (r *runRepository) ListRunResponses(ctx context.Context, orgID, runID int32) ([]*domain.InquiryResponse, error) {
	rows, err := r.store.ListRunResponses(ctx, sqlc.ListRunResponsesParams{RunID: runID, OrganizationID: orgID})
	if err != nil {
		return nil, err
	}
	out := make([]*domain.InquiryResponse, 0, len(rows))
	for i := range rows {
		resp, err := mapResponse(rows[i])
		if err != nil {
			return nil, err
		}
		out = append(out, resp)
	}
	return out, nil
}

func (r *runRepository) RunBoardRows(ctx context.Context, orgID, runID int32) ([]domain.BoardRow, error) {
	rows, err := r.store.GetRunBoardRows(ctx, sqlc.GetRunBoardRowsParams{RunID: runID, OrganizationID: orgID})
	if err != nil {
		return nil, err
	}
	out := make([]domain.BoardRow, 0, len(rows))
	for i := range rows {
		row := rows[i]
		br := domain.BoardRow{
			RecipientID:       row.RecipientID,
			RecipientStatus:   domain.RecipientStatus(row.RecipientStatus),
			SentAt:            helpers.FromPgTimestamptzPtr(row.SentAt),
			AnsweredAt:        helpers.FromPgTimestamptzPtr(row.AnsweredAt),
			ProviderMessageID: helpers.FromPgTextPtr(row.ProviderMessageID),
			SupplierID:        row.SupplierID,
			NIT:               row.Nit,
			DeliveryDays:      helpers.FromPgInt4Ptr(row.DeliveryDays),
			MinOrderAmount:    helpers.FromPgNumeric(row.MinOrderAmount),
			ContactID:         row.ContactID,
			DisplayName:       row.DisplayName,
			PhoneNumber:       row.PhoneNumber.String,
		}
		if row.Extracted != nil {
			items := []domain.ResponseItem{}
			if len(row.Extracted) > 0 {
				if err := json.Unmarshal(row.Extracted, &items); err != nil {
					return nil, fmt.Errorf("decode board response: %w", err)
				}
			}
			br.Response = &domain.InquiryResponse{
				Items:          items,
				Resumen:        helpers.FromPgText(row.Resumen),
				Confidence:     pgFloat8Ptr(row.Confidence),
				RequiereHumano: row.RequiereHumano,
			}
		}
		out = append(out, br)
	}
	return out, nil
}

func (r *runRepository) SendFanOut(ctx context.Context, orgID, runID int32, events []domain.OutboxEventInput) (*domain.InquiryRun, error) {
	var run *domain.InquiryRun
	err := inTx(ctx, r.store, func(s sqlc.Store) error {
		row, err := s.UpdateRunStatusFrom(ctx, sqlc.UpdateRunStatusFromParams{
			ID:             runID,
			OrganizationID: orgID,
			Status:         string(domain.RunDraft),
			Status_2:       string(domain.RunSending),
		})
		if isNoRows(err) {
			return domain.ErrInvalidTransition
		}
		if err != nil {
			return err
		}
		run = mapRun(row)

		for _, ev := range events {
			if _, err := s.InsertOutboxEvent(ctx, sqlc.InsertOutboxEventParams{
				EventType:      ev.EventType,
				Payload:        ev.Payload,
				OrganizationID: helpers.ToPgInt4Ptr(&orgID),
			}); err != nil {
				return fmt.Errorf("enqueue %s: %w", ev.EventType, err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return run, nil
}

// errRecipientGuard is an internal sentinel for "already transitioned".
var _ = errors.New