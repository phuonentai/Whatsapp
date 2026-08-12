package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/moasq/go-b2b-starter/internal/db/helpers"
	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
	"github.com/moasq/go-b2b-starter/internal/modules/procurement/domain"
)

func mapOrder(row sqlc.ProcurementOrder) (*domain.Order, error) {
	items := []domain.OrderItem{}
	if len(row.Items) > 0 {
		if err := json.Unmarshal(row.Items, &items); err != nil {
			return nil, fmt.Errorf("decode order items: %w", err)
		}
	}
	return &domain.Order{
		ID:                row.ID,
		OrganizationID:    row.OrganizationID,
		RunID:             row.RunID,
		SupplierID:        row.SupplierID,
		ContactID:         row.ContactID,
		NegocioID:         helpers.FromPgInt4Ptr(row.NegocioID),
		Status:            domain.OrderStatus(row.Status),
		Items:             items,
		Notes:             helpers.FromPgTextPtr(row.Notes),
		ConfirmMessage:    helpers.FromPgTextPtr(row.ConfirmMessage),
		BlockedReason:     helpers.FromPgTextPtr(row.BlockedReason),
		CreatedByMemberID: helpers.FromPgTextPtr(row.CreatedByMemberID),
		CreatedAt:         row.CreatedAt.Time,
		UpdatedAt:         row.UpdatedAt.Time,
	}, nil
}

func orderParams(order *domain.Order) (sqlc.CreateOrderParams, error) {
	items, err := domain.MarshalOrderItems(order.Items)
	if err != nil {
		return sqlc.CreateOrderParams{}, err
	}
	return sqlc.CreateOrderParams{
		OrganizationID:    order.OrganizationID,
		RunID:             order.RunID,
		SupplierID:        order.SupplierID,
		ContactID:         order.ContactID,
		Items:             items,
		Notes:             helpers.ToPgTextPtr(order.Notes),
		ConfirmMessage:    helpers.ToPgTextPtr(order.ConfirmMessage),
		Status:            string(order.Status),
		CreatedByMemberID: helpers.ToPgTextPtr(order.CreatedByMemberID),
		BlockedReason:     helpers.ToPgTextPtr(order.BlockedReason),
	}, nil
}

type orderRepository struct {
	store sqlc.Store
}

// NewOrderRepository builds the order repository.
func NewOrderRepository(store sqlc.Store) domain.OrderRepository {
	return &orderRepository{store: store}
}

// PlaceOrderTx runs the atomic order-placement transaction (D13): order
// marker + (confirmation outbox event | blocked audit) + negocio + actividad
// + order_placed audit + negocio link. Any failure rolls everything back.
func (r *orderRepository) PlaceOrderTx(ctx context.Context, in domain.PlaceOrderTxParams) (*domain.Order, error) {
	order := in.Order
	var created *domain.Order

	err := inTx(ctx, r.store, func(s sqlc.Store) error {
		params, err := orderParams(order)
		if err != nil {
			return err
		}
		row, err := s.CreateOrder(ctx, params)
		if isNoRows(err) {
			return domain.ErrOrderAlreadyPlaced
		}
		if err != nil {
			return err
		}
		mapped, err := mapOrder(row)
		if err != nil {
			return err
		}
		created = mapped

		blocked := order.Status == domain.OrderSendBlocked
		if in.ConfirmEvent != nil && !blocked {
			if _, err := s.InsertOutboxEvent(ctx, sqlc.InsertOutboxEventParams{
				EventType:      in.ConfirmEvent.EventType,
				Payload:        in.ConfirmEvent.Payload,
				OrganizationID: helpers.ToPgInt4Ptr(&order.OrganizationID),
			}); err != nil {
				return fmt.Errorf("enqueue order confirm: %w", err)
			}
		}

		pipelineID, err := s.GetDefaultPipelineID(ctx, order.OrganizationID)
		if isNoRows(err) {
			return domain.ErrDefaultPipelineMissing
		}
		if err != nil {
			return err
		}

		deal, err := s.CreateDeal(ctx, sqlc.CreateDealParams{
			OrganizationID: order.OrganizationID,
			Nombre:         in.DealNombre,
			ContactID:      helpers.ToPgInt4Ptr(&order.ContactID),
			CompanyID:      pgtype.Int4{},
			PipelineID:     pipelineID,
			StageID:        pgtype.Int4{},
			Monto:          pgtype.Numeric{},
			Moneda:         "COP",
			FechaCierreEsperada: pgtype.Date{},
			Estado:         "abierto",
			Probabilidad:   pgtype.Int4{},
			Notas:          pgtype.Text{},
			Metadata:       []byte(`{}`),
			AssignedTo:     pgtype.Int4{},
		})
		if err != nil {
			return fmt.Errorf("create negocio: %w", err)
		}
		dealID := deal.ID

		if _, err := s.CreateActivity(ctx, sqlc.CreateActivityParams{
			OrganizationID: order.OrganizationID,
			ContactID:      helpers.ToPgInt4Ptr(&order.ContactID),
			CompanyID:      pgtype.Int4{},
			DealID:         helpers.ToPgInt4Ptr(&dealID),
			ConversationID: pgtype.Int4{},
			Tipo:           "sistema",
			Asunto:         helpers.ToPgTextPtr(&in.ActividadAsunto),
			Contenido:      helpers.ToPgTextPtr(&in.ActividadContenido),
			Estado:         pgtype.Text{},
			FechaVencimiento: pgtype.Timestamptz{},
			RealizadaPor:   pgtype.Int4{},
			RealizadaEn:    pgTimestamptzNow(),
			Metadata:       []byte(`{}`),
		}); err != nil {
			return fmt.Errorf("create actividad: %w", err)
		}

		if _, err := s.UpdateOrderNegocioID(ctx, sqlc.UpdateOrderNegocioIDParams{
			ID:             order.ID,
			OrganizationID: order.OrganizationID,
			NegocioID:      helpers.ToPgInt4Ptr(&dealID),
		}); err != nil {
			return err
		}
		created.NegocioID = &dealID

		decision := "allow"
		var reason *string
		action := "order_placed"
		if blocked {
			decision = "skip"
			action = "order_send_blocked"
			reason = order.BlockedReason
		}
		if _, err := s.InsertProcurementAudit(ctx, sqlc.InsertProcurementAuditParams{
			OrganizationID: order.OrganizationID,
			EntityType:     "order",
			EntityID:       helpers.ToPgInt4Ptr(&created.ID),
			Action:         action,
			Decision:       decision,
			Reason:         helpers.ToPgTextPtr(reason),
			MemberID:       helpers.ToPgTextPtr(order.CreatedByMemberID),
			Metadata:       []byte(`{}`),
		}); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.SQLState() == "23505" && pgErr.ConstraintName == "orders_run_supplier_unique" {
			return nil, domain.ErrOrderAlreadyPlaced
		}
		return nil, err
	}
	return created, nil
}

func pgTimestamptzNow() pgtype.Timestamptz {
	return helpers.ToPgTimestamptz(time.Now())
}

// CreateOrder is idempotent on (run_id, supplier_id): a duplicate POST maps
// to domain.ErrOrderAlreadyPlaced (the handler then returns the existing
// order instead of creating a second one).
func (r *orderRepository) CreateOrder(ctx context.Context, order *domain.Order) (*domain.Order, error) {
	params, err := orderParams(order)
	if err != nil {
		return nil, err
	}
	row, err := r.store.CreateOrder(ctx, params)
	if isNoRows(err) {
		return nil, domain.ErrOrderAlreadyPlaced
	}
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.SQLState() == "23505" && pgErr.ConstraintName == "orders_run_supplier_unique" {
			return nil, domain.ErrOrderAlreadyPlaced
		}
		return nil, err
	}
	return mapOrder(row)
}

func (r *orderRepository) GetOrderByRunSupplier(ctx context.Context, runID, supplierID int32) (*domain.Order, error) {
	row, err := r.store.GetOrderByRunSupplier(ctx, sqlc.GetOrderByRunSupplierParams{RunID: runID, SupplierID: supplierID})
	if isNoRows(err) {
		return nil, domain.ErrOrderNotFound
	}
	if err != nil {
		return nil, err
	}
	return mapOrder(row)
}

func (r *orderRepository) GetOrder(ctx context.Context, orgID, orderID int32) (*domain.Order, error) {
	row, err := r.store.GetOrder(ctx, sqlc.GetOrderParams{ID: orderID, OrganizationID: orgID})
	if isNoRows(err) {
		return nil, domain.ErrOrderNotFound
	}
	if err != nil {
		return nil, err
	}
	return mapOrder(row)
}

func (r *orderRepository) MarkOrderConfirmSent(ctx context.Context, orgID, orderID int32) (*domain.Order, error) {
	row, err := r.store.UpdateOrderConfirmSent(ctx, sqlc.UpdateOrderConfirmSentParams{ID: orderID, OrganizationID: orgID})
	if isNoRows(err) {
		return nil, domain.ErrOrderNotFound
	}
	if err != nil {
		return nil, err
	}
	return mapOrder(row)
}

func (r *orderRepository) MarkOrderSendBlocked(ctx context.Context, orgID, orderID int32, reason string) (*domain.Order, error) {
	row, err := r.store.UpdateOrderSendBlocked(ctx, sqlc.UpdateOrderSendBlockedParams{
		ID:             orderID,
		OrganizationID: orgID,
		BlockedReason:  helpers.ToPgTextPtr(&reason),
	})
	if err != nil {
		return nil, err
	}
	return mapOrder(row)
}

func (r *orderRepository) MarkOrderConfirmFailed(ctx context.Context, orgID, orderID int32) (*domain.Order, error) {
	row, err := r.store.UpdateOrderConfirmFailed(ctx, sqlc.UpdateOrderConfirmFailedParams{ID: orderID, OrganizationID: orgID})
	if err != nil {
		return nil, err
	}
	return mapOrder(row)
}

func (r *orderRepository) ListRunOrders(ctx context.Context, orgID, runID int32) ([]*domain.Order, error) {
	rows, err := r.store.ListRunOrders(ctx, sqlc.ListRunOrdersParams{RunID: runID, OrganizationID: orgID})
	if err != nil {
		return nil, err
	}
	out := make([]*domain.Order, 0, len(rows))
	for i := range rows {
		o, err := mapOrder(rows[i])
		if err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, nil
}
