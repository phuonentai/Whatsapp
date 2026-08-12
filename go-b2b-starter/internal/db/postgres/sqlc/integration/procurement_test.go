//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/moasq/go-b2b-starter/internal/db/helpers"
	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
)

// Procurement schema: migration integrity, tenant-safe composite FKs, and
// idempotent transitions (add-supplier-inquiry-agent).

// TestProcurementDownMigrationDropsSchema verifies 000037's down migration
// fully drops the procurement schema (mirroring campaign conventions) and
// that the up migration restores it.
func TestProcurementDownMigrationDropsSchema(t *testing.T) {
	ctx := context.Background()

	body, err := readMigration("000037_create_procurement_schema.down.sql")
	if err != nil {
		t.Fatalf("read down migration: %v", err)
	}
	if _, err := testPool.Exec(ctx, string(body)); err != nil {
		t.Fatalf("apply down migration: %v", err)
	}

	var exists bool
	if err := testPool.QueryRow(ctx,
		"SELECT EXISTS (SELECT 1 FROM information_schema.schemata WHERE schema_name='procurement')",
	).Scan(&exists); err != nil {
		t.Fatalf("check schema: %v", err)
	}
	if exists {
		t.Fatalf("expected procurement schema to be dropped")
	}

	up, err := readMigration("000037_create_procurement_schema.up.sql")
	if err != nil {
		t.Fatalf("read up migration: %v", err)
	}
	if _, err := testPool.Exec(ctx, string(up)); err != nil {
		t.Fatalf("re-apply up migration: %v", err)
	}
}

// TestProcurementCrossTenantContactRejected verifies the composite tenant FK
// rejects a supplier pointing at another org's contact (23503).
func TestProcurementCrossTenantContactRejected(t *testing.T) {
	ctx := context.Background()
	q := testStore
	orgA, _ := createOrgWithAccount(t, ctx, q)
	orgB, _ := createOrgWithAccount(t, ctx, q)

	contact, err := q.CreateSupplierContact(ctx, sqlc.CreateSupplierContactParams{
		OrganizationID:  orgA,
		PhoneNumber:     helpers.ToPgText("+573002222001"),
		DisplayName:     helpers.ToPgText("Proveedor A"),
		NumeroDocumento: helpers.ToPgText("900111222"),
	})
	if err != nil {
		t.Fatalf("seed contact A: %v", err)
	}

	// Same NIT under orgB is legal (per-org unique)...
	if _, err := q.CreateSupplierContact(ctx, sqlc.CreateSupplierContactParams{
		OrganizationID:  orgB,
		PhoneNumber:     helpers.ToPgText("+573002222002"),
		DisplayName:     helpers.ToPgText("Proveedor B"),
		NumeroDocumento: helpers.ToPgText("900111222"),
	}); err != nil {
		t.Fatalf("seed contact B: %v", err)
	}

	// ...but a supplier row in orgB referencing orgA's contact must fail.
	_, err = q.CreateSupplier(ctx, sqlc.CreateSupplierParams{
		OrganizationID: orgB,
		ContactID:      contact.ID,
		Nit:            "900111222",
		DeliveryDays:   pgtype.Int4{},
		MinOrderAmount: pgtype.Numeric{},
		Notes:          pgtype.Text{},
	})
	if !isPgError(err, "23503") {
		t.Fatalf("expected cross-tenant FK violation (23503), got: %v", err)
	}
}

// TestProcurementDuplicateNitRejected verifies the per-org NIT uniqueness.
func TestProcurementDuplicateNitRejected(t *testing.T) {
	ctx := context.Background()
	q := testStore
	org, _ := createOrgWithAccount(t, ctx, q)

	contact, err := q.CreateSupplierContact(ctx, sqlc.CreateSupplierContactParams{
		OrganizationID:  org,
		PhoneNumber:     helpers.ToPgText("+573002222010"),
		DisplayName:     helpers.ToPgText("Prov"),
		NumeroDocumento: helpers.ToPgText("900222333"),
	})
	if err != nil {
		t.Fatalf("seed contact: %v", err)
	}
	if _, err := q.CreateSupplier(ctx, sqlc.CreateSupplierParams{
		OrganizationID: org,
		ContactID:      contact.ID,
		Nit:            "900222333",
		DeliveryDays:   pgtype.Int4{},
		MinOrderAmount: pgtype.Numeric{},
		Notes:          pgtype.Text{},
	}); err != nil {
		t.Fatalf("create supplier: %v", err)
	}

	other, err := q.CreateSupplierContact(ctx, sqlc.CreateSupplierContactParams{
		OrganizationID:  org,
		PhoneNumber:     helpers.ToPgText("+573002222011"),
		DisplayName:     helpers.ToPgText("Prov2"),
		NumeroDocumento: helpers.ToPgText("900222333"),
	})
	if err != nil {
		t.Fatalf("seed second contact: %v", err)
	}
	_, err = q.CreateSupplier(ctx, sqlc.CreateSupplierParams{
		OrganizationID: org,
		ContactID:      other.ID,
		Nit:            "900222333",
		DeliveryDays:   pgtype.Int4{},
		MinOrderAmount: pgtype.Numeric{},
		Notes:          pgtype.Text{},
	})
	if !isPgError(err, "23505") {
		t.Fatalf("expected unique violation (23505) for duplicate NIT, got: %v", err)
	}
}

// TestProcurementRedeliveredSendNoDoubleTransition verifies the guarded
// pending → sent transition is a no-op on a second dispatch (no row, no
// double transition).
func TestProcurementRedeliveredSendNoDoubleTransition(t *testing.T) {
	ctx := context.Background()
	q := testStore
	org, _ := createOrgWithAccount(t, ctx, q)

	run, err := q.CreateInquiryRun(ctx, sqlc.CreateInquiryRunParams{
		OrganizationID:    org,
		Nota:              helpers.ToPgTextPtr(nil),
		CreatedByMemberID: helpers.ToPgText("member-1"),
	})
	if err != nil {
		t.Fatalf("create run: %v", err)
	}

	contact, err := q.CreateSupplierContact(ctx, sqlc.CreateSupplierContactParams{
		OrganizationID:  org,
		PhoneNumber:     helpers.ToPgText("+573002222020"),
		DisplayName:     helpers.ToPgText("Prov"),
		NumeroDocumento: helpers.ToPgText("900444555"),
	})
	if err != nil {
		t.Fatalf("seed contact: %v", err)
	}
	supplier, err := q.CreateSupplier(ctx, sqlc.CreateSupplierParams{
		OrganizationID: org,
		ContactID:      contact.ID,
		Nit:            "900444555",
		DeliveryDays:   pgtype.Int4{},
		MinOrderAmount: pgtype.Numeric{},
		Notes:          pgtype.Text{},
	})
	if err != nil {
		t.Fatalf("create supplier: %v", err)
	}
	recipient, err := q.CreateInquiryRecipient(ctx, sqlc.CreateInquiryRecipientParams{
		OrganizationID: org,
		RunID:          run.ID,
		SupplierID:     supplier.ID,
		ContactID:      contact.ID,
		DraftedMessage: helpers.ToPgText("Mensaje"),
	})
	if err != nil {
		t.Fatalf("create recipient: %v", err)
	}

	sent, err := q.MarkRecipientSent(ctx, sqlc.MarkRecipientSentParams{
		ID:                recipient.ID,
		OrganizationID:    org,
		ProviderMessageID: helpers.ToPgText("wamid.1"),
	})
	if err != nil {
		t.Fatalf("first dispatch: %v", err)
	}
	if sent.Status != "sent" {
		t.Fatalf("expected sent status, got %q", sent.Status)
	}

	// Redelivery: the guarded update matches no 'pending' row → ErrNoRows.
	_, err = q.MarkRecipientSent(ctx, sqlc.MarkRecipientSentParams{
		ID:                recipient.ID,
		OrganizationID:    org,
		ProviderMessageID: helpers.ToPgText("wamid.2"),
	})
	if err == nil {
		t.Fatalf("expected no-op (ErrNoRows) on redelivered dispatch")
	}
}

// TestProcurementDuplicateResponseNoOp verifies the idempotent response insert
// on (recipient_id, raw_message_id).
func TestProcurementDuplicateResponseNoOp(t *testing.T) {
	ctx := context.Background()
	q := testStore
	org, _ := createOrgWithAccount(t, ctx, q)

	run, err := q.CreateInquiryRun(ctx, sqlc.CreateInquiryRunParams{
		OrganizationID:    org,
		Nota:              helpers.ToPgTextPtr(nil),
		CreatedByMemberID: helpers.ToPgText("member-1"),
	})
	if err != nil {
		t.Fatalf("create run: %v", err)
	}
	contact, err := q.CreateSupplierContact(ctx, sqlc.CreateSupplierContactParams{
		OrganizationID:  org,
		PhoneNumber:     helpers.ToPgText("+573002222030"),
		DisplayName:     helpers.ToPgText("Prov"),
		NumeroDocumento: helpers.ToPgText("900555666"),
	})
	if err != nil {
		t.Fatalf("seed contact: %v", err)
	}
	supplier, err := q.CreateSupplier(ctx, sqlc.CreateSupplierParams{
		OrganizationID: org,
		ContactID:      contact.ID,
		Nit:            "900555666",
		DeliveryDays:   pgtype.Int4{},
		MinOrderAmount: pgtype.Numeric{},
		Notes:          pgtype.Text{},
	})
	if err != nil {
		t.Fatalf("create supplier: %v", err)
	}
	recipient, err := q.CreateInquiryRecipient(ctx, sqlc.CreateInquiryRecipientParams{
		OrganizationID: org,
		RunID:          run.ID,
		SupplierID:     supplier.ID,
		ContactID:      contact.ID,
		DraftedMessage: helpers.ToPgText("Mensaje"),
	})
	if err != nil {
		t.Fatalf("create recipient: %v", err)
	}

	first, err := q.CreateInquiryResponse(ctx, sqlc.CreateInquiryResponseParams{
		OrganizationID: org,
		RecipientID:    recipient.ID,
		RawMessageID:   "wamid.inbound.1",
		Extracted:      []byte(`[{"product_name":"Papel","disponible":true}]`),
		Resumen:        helpers.ToPgText("Disponible"),
		Confidence:     pgtype.Float8{Float64: 0.9, Valid: true},
		RequiereHumano: false,
	})
	if err != nil {
		t.Fatalf("insert response: %v", err)
	}
	if first.ID == 0 {
		t.Fatalf("expected persisted response id")
	}

	// Redelivery of the same raw_message_id must be a no-op (no row returned).
	_, err = q.CreateInquiryResponse(ctx, sqlc.CreateInquiryResponseParams{
		OrganizationID: org,
		RecipientID:    recipient.ID,
		RawMessageID:   "wamid.inbound.1",
		Extracted:      []byte(`[{"product_name":"Papel","disponible":false}]`),
		Resumen:        helpers.ToPgText("Agotado"),
		Confidence:     pgtype.Float8{Float64: 0.9, Valid: true},
		RequiereHumano: true,
	})
	if err == nil {
		t.Fatalf("expected no-op (ErrNoRows) on duplicate response insert")
	}

	var count int32
	if err := testPool.QueryRow(ctx,
		"SELECT COUNT(*) FROM procurement.inquiry_responses WHERE recipient_id=$1",
		recipient.ID,
	).Scan(&count); err != nil {
		t.Fatalf("count responses: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 response row, got %d", count)
	}
}
