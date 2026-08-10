package repositories

import (
	"context"
	"fmt"

	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
	"github.com/moasq/go-b2b-starter/internal/modules/campaigns/domain"
)

// segmentEvaluator evaluates filter specs via SQL. Hard gates (consent =
// granted, valid E.164 phone) are baked into the queries and can never be
// removed by filter_spec content.
type segmentEvaluator struct {
	store sqlc.Store
}

func NewSegmentEvaluator(store sqlc.Store) domain.SegmentEvaluator {
	return &segmentEvaluator{store: store}
}

// evalParams converts a validated filter spec into the shared query
// parameters. tag_ids uses any-of semantics (ANY(...)).
func evalParams(orgID int32, spec []domain.Filter) sqlc.CountSegmentContactsParams {
	p := sqlc.CountSegmentContactsParams{OrganizationID: orgID}
	for _, f := range spec {
		switch f.Field {
		case domain.FieldSource:
			p.Column2 = textOrEmpty(f.Value)
		case domain.FieldLeadStatus:
			p.Column3 = textOrEmpty(f.Value)
		case domain.FieldCompanyID:
			p.Column4 = intOrZero(f.Value)
		case domain.FieldAssignedTo:
			p.Column5 = intOrZero(f.Value)
		case domain.FieldTagIDs:
			if arr, ok := toInt32Array(f.Value); ok {
				p.Column6 = arr
			}
		case domain.FieldRecencyDays:
			p.Column7 = intOrZero(f.Value)
		case domain.FieldSearch:
			p.Column8 = textOrEmpty(f.Value)
		}
	}
	return p
}

func (e *segmentEvaluator) Count(ctx context.Context, orgID int32, spec []domain.Filter) (*domain.EvalResult, error) {
	row, err := e.store.CountSegmentContacts(ctx, evalParams(orgID, spec))
	if err != nil {
		return nil, fmt.Errorf("failed to count segment contacts: %w", err)
	}
	return &domain.EvalResult{
		Total:           row.Total,
		ExcludedByGates: row.ExcludedByGates,
	}, nil
}

func (e *segmentEvaluator) ContactIDs(ctx context.Context, orgID int32, spec []domain.Filter) ([]int32, error) {
	params := evalParams(orgID, spec)
	// Page through the full match set (SMB scale; 10k rows per page).
	ids := make([]int32, 0)
	const pageSize = 10000
	for offset := 0; ; offset += pageSize {
		rows, err := e.store.ListSegmentContacts(ctx, sqlc.ListSegmentContactsParams{
			OrganizationID: params.OrganizationID,
			Column2:        params.Column2,
			Column3:        params.Column3,
			Column4:        params.Column4,
			Column5:        params.Column5,
			Column6:        params.Column6,
			Column7:        params.Column7,
			Column8:        params.Column8,
			Limit:          pageSize,
			Offset:         int32(offset),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list segment contacts: %w", err)
		}
		for i := range rows {
			ids = append(ids, rows[i].ID)
		}
		if len(rows) < pageSize {
			break
		}
	}
	return ids, nil
}

func textOrEmpty(value any) string {
	s, ok := value.(string)
	if !ok {
		return ""
	}
	return s
}

func intOrZero(value any) int32 {
	switch v := value.(type) {
	case float64:
		return int32(v)
	case int:
		return int32(v)
	case int32:
		return v
	case int64:
		return int32(v)
	}
	return 0
}

func toInt32Array(value any) ([]int32, bool) {
	switch v := value.(type) {
	case []int32:
		return v, true
	case []any:
		out := make([]int32, 0, len(v))
		for _, item := range v {
			n, ok := intOrZeroOk(item)
			if !ok {
				return nil, false
			}
			out = append(out, n)
		}
		return out, true
	}
	return nil, false
}

func intOrZeroOk(value any) (int32, bool) {
	switch v := value.(type) {
	case float64:
		if v != float64(int32(v)) {
			return 0, false
		}
		return int32(v), true
	case int:
		return int32(v), true
	case int32:
		return v, true
	case int64:
		return int32(v), true
	}
	return 0, false
}
