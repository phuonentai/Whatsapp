//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/moasq/go-b2b-starter/internal/db/helpers"
	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
)

// seedCampaignContact inserts a contact with granted consent and a valid
// E.164 phone so it passes the hard gates.
func seedCampaignContact(t *testing.T, ctx context.Context, q sqlc.Querier, orgID int32, phone, name string) int32 {
	t.Helper()
	c, err := q.UpsertContact(ctx, sqlc.UpsertContactParams{
		OrganizationID: orgID,
		PhoneNumber:    pgtype.Text{String: phone, Valid: true},
		DisplayName:    helpers.ToPgText(name),
	})
	if err != nil {
		t.Fatalf("upsert contact %s: %v", phone, err)
	}
	if _, err := q.UpdateContact(ctx, sqlc.UpdateContactParams{
		ID:             c.ID,
		OrganizationID: orgID,
		Column3:        helpers.ToPgText(""),
		CompanyID:      pgtype.Int4{},
		Column5:        helpers.ToPgText(name),
		Column6:        helpers.ToPgText("whatsapp"),
		Column7:        helpers.ToPgText("cliente"),
		Column8:        helpers.ToPgText(""),
		AssignedTo:     pgtype.Int4{},
		Column10:       helpers.ToPgText(""),
		Column11:       helpers.ToPgText(""),
		Column12:       helpers.ToPgText(""),
		Column13:       nil,
	}); err != nil {
		t.Fatalf("update contact %s: %v", phone, err)
	}
	// Grant consent (hard gate requires consent_status = 'granted').
	if _, err := q.UpdateContactConsent(ctx, sqlc.UpdateContactConsentParams{
		ID:             c.ID,
		OrganizationID: orgID,
		ConsentStatus:  "granted",
		ConsentedAt:    pgtype.Timestamp{},
	}); err != nil {
		t.Fatalf("grant consent: %v", err)
	}
	return c.ID
}

// setLeadStatus marks a contact as cliente (so it matches the segment filter).
func setLeadStatus(t *testing.T, ctx context.Context, q sqlc.Querier, orgID, contactID int32) {
	t.Helper()
	if _, err := q.UpdateContact(ctx, sqlc.UpdateContactParams{
		ID:             contactID,
		OrganizationID: orgID,
		Column3:        helpers.ToPgText(""),
		CompanyID:      pgtype.Int4{},
		Column5:        helpers.ToPgText(""),
		Column6:        helpers.ToPgText("whatsapp"),
		Column7:        helpers.ToPgText("cliente"),
		Column8:        helpers.ToPgText(""),
		AssignedTo:     pgtype.Int4{},
		Column10:       helpers.ToPgText(""),
		Column11:       helpers.ToPgText(""),
		Column12:       helpers.ToPgText(""),
		Column13:       nil,
	}); err != nil {
		t.Fatalf("set lead status: %v", err)
	}
}

// evalParams builds ListSegmentContactsParams with the given filters.
func makeEvalParams(orgID int32, specJSON string) sqlc.CountSegmentContactsParams {
	p := sqlc.CountSegmentContactsParams{OrganizationID: orgID}
	switch specJSON {
	case "lead_status=cliente":
		p.Column3 = "cliente"
	case "tag":
		p.Column6 = []int32{}
	case "recency":
		p.Column7 = 365
	case "source":
		p.Column2 = "whatsapp"
	}
	return p
}

func TestSegmentEvalAppliesHardGates(t *testing.T) {
	ctx := context.Background()
	q := testStore
	orgID, _ := createOrgWithAccount(t, ctx, q)

	grantedID := seedCampaignContact(t, ctx, q, orgID, "+573001111111", "Ana Cliente")

	// Contact without consent: default 'none' after upsert, matches the
	// filter but fails the consent gate.
	noConsent, err := q.UpsertContact(ctx, sqlc.UpsertContactParams{
		OrganizationID: orgID,
		PhoneNumber:    pgtype.Text{String: "+573002222222", Valid: true},
		DisplayName:    helpers.ToPgText("Sin Consentimiento"),
	})
	if err != nil {
		t.Fatalf("upsert no-consent contact: %v", err)
	}
	setLeadStatus(t, ctx, q, orgID, noConsent.ID)

	// Contact with invalid phone (no + prefix): matches the filter but fails
	// the phone gate.
	invalidPhone, err := q.UpsertContact(ctx, sqlc.UpsertContactParams{
		OrganizationID: orgID,
		PhoneNumber:    pgtype.Text{String: "573003333333", Valid: true},
		DisplayName:    helpers.ToPgText("Telefono Invalido"),
	})
	if err != nil {
		t.Fatalf("upsert invalid-phone contact: %v", err)
	}
	setLeadStatus(t, ctx, q, orgID, invalidPhone.ID)
	if _, err := q.UpdateContactConsent(ctx, sqlc.UpdateContactConsentParams{
		ID:             invalidPhone.ID,
		OrganizationID: orgID,
		ConsentStatus:  "granted",
		ConsentedAt:    pgtype.Timestamp{},
	}); err != nil {
		t.Fatalf("grant consent to invalid-phone contact: %v", err)
	}

	res, err := q.CountSegmentContacts(ctx, makeEvalParams(orgID, "lead_status=cliente"))
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if res.Total != 1 {
		t.Fatalf("expected 1 granted+valid-phone match, got %d (excluded %d)", res.Total, res.ExcludedByGates)
	}
	// Total matched by filter (3) minus granted+valid (1) = 2 gate exclusions.
	if res.ExcludedByGates != 2 {
		t.Fatalf("expected 2 gate exclusions, got %d", res.ExcludedByGates)
	}

	ids, err := q.ListSegmentContacts(ctx, sqlc.ListSegmentContactsParams{
		OrganizationID: orgID,
		Column3:        "cliente",
		Limit:          50,
		Offset:         0,
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(ids) != 1 || ids[0].ID != grantedID {
		t.Fatalf("expected only the granted+valid contact, got %d rows", len(ids))
	}
}

func TestSegmentEvalTagsJoinAndOrgIsolation(t *testing.T) {
	ctx := context.Background()
	q := testStore
	orgID, _ := createOrgWithAccount(t, ctx, q)
	otherOrg, _ := createOrgWithAccount(t, ctx, q)

	contactID := seedCampaignContact(t, ctx, q, orgID, "+573004444444", "Etiquetado")

	tag, err := q.CreateTag(ctx, sqlc.CreateTagParams{
		OrganizationID: orgID,
		Nombre:         "mayorista",
		Color:          helpers.ToPgText("#F59E0B"),
	})
	if err != nil {
		t.Fatalf("create tag: %v", err)
	}
	if _, err := q.AttachTag(ctx, sqlc.AttachTagParams{
		TagID:      tag.ID,
		EntityType: "contact",
		EntityID:   contactID,
	}); err != nil {
		t.Fatalf("attach tag: %v", err)
	}

	params := sqlc.CountSegmentContactsParams{OrganizationID: orgID, Column6: []int32{tag.ID}}
	res, err := q.CountSegmentContacts(ctx, params)
	if err != nil {
		t.Fatalf("count by tag: %v", err)
	}
	if res.Total != 1 {
		t.Fatalf("expected 1 tagged contact, got %d", res.Total)
	}

	// Org isolation: same tag id must match nothing in the other org.
	otherRes, err := q.CountSegmentContacts(ctx, sqlc.CountSegmentContactsParams{OrganizationID: otherOrg, Column6: []int32{tag.ID}})
	if err != nil {
		t.Fatalf("count by tag in other org: %v", err)
	}
	if otherRes.Total != 0 {
		t.Fatalf("expected 0 matches in other org, got %d", otherRes.Total)
	}
}

func TestCampaignSnapshotDedupAndGuardedRelaunch(t *testing.T) {
	ctx := context.Background()
	q := testStore
	orgID, _ := createOrgWithAccount(t, ctx, q)

	contactA := seedCampaignContact(t, ctx, q, orgID, "+573005555555", "A")
	contactB := seedCampaignContact(t, ctx, q, orgID, "+573006666666", "B")

	segment, err := q.CreateSegment(ctx, sqlc.CreateSegmentParams{
		OrganizationID: orgID,
		Nombre:         "Clientes",
		FilterSpec:     []byte(`[{"field":"lead_status","op":"eq","value":"cliente"}]`),
		CreatedBy:      helpers.ToPgText("m1"),
	})
	if err != nil {
		t.Fatalf("create segment: %v", err)
	}
	campaign, err := q.CreateCampaign(ctx, sqlc.CreateCampaignParams{
		OrganizationID: orgID,
		Nombre:         "Promo",
		SegmentID:      segment.ID,
		CreatedBy:      helpers.ToPgText("m1"),
	})
	if err != nil {
		t.Fatalf("create campaign: %v", err)
	}

	inserted, err := q.SnapshotCampaignRecipients(ctx, sqlc.SnapshotCampaignRecipientsParams{
		CampaignID: campaign.ID,
		Column2:    []int32{contactA, contactB},
	})
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if inserted != 2 {
		t.Fatalf("expected 2 inserted, got %d", inserted)
	}

	// Idempotent re-snapshot with overlap: no new rows.
	inserted, err = q.SnapshotCampaignRecipients(ctx, sqlc.SnapshotCampaignRecipientsParams{
		CampaignID: campaign.ID,
		Column2:    []int32{contactA, contactB},
	})
	if err != nil {
		t.Fatalf("re-snapshot: %v", err)
	}
	if inserted != 0 {
		t.Fatalf("expected 0 duplicate inserts, got %d", inserted)
	}

	// Guarded launch works once...
	launched, err := q.LaunchCampaign(ctx, sqlc.LaunchCampaignParams{
		ID:             campaign.ID,
		OrganizationID: orgID,
		RecipientCount: 2,
	})
	if err != nil {
		t.Fatalf("launch: %v", err)
	}
	if launched.Status != "ready" {
		t.Fatalf("expected ready, got %s", launched.Status)
	}

	// ...and the second launch hits the draft guard (no row returned).
	_, err = q.LaunchCampaign(ctx, sqlc.LaunchCampaignParams{
		ID:             campaign.ID,
		OrganizationID: orgID,
		RecipientCount: 2,
	})
	if err != pgx.ErrNoRows {
		t.Fatalf("expected ErrNoRows on relaunch, got %v", err)
	}

	recipients, err := q.ListCampaignRecipients(ctx, sqlc.ListCampaignRecipientsParams{
		CampaignID: campaign.ID,
		Limit:      50,
		Offset:     0,
	})
	if err != nil {
		t.Fatalf("list recipients: %v", err)
	}
	if len(recipients) != 2 {
		t.Fatalf("expected 2 recipients, got %d", len(recipients))
	}
}

func TestCampaignOrgScopedFKRejectsForeignSegment(t *testing.T) {
	ctx := context.Background()
	q := testStore
	orgID, _ := createOrgWithAccount(t, ctx, q)
	otherOrg, _ := createOrgWithAccount(t, ctx, q)

	segment, err := q.CreateSegment(ctx, sqlc.CreateSegmentParams{
		OrganizationID: otherOrg,
		Nombre:         "De otro org",
		FilterSpec:     []byte(`[]`),
	})
	if err != nil {
		t.Fatalf("create segment: %v", err)
	}

	_, err = q.CreateCampaign(ctx, sqlc.CreateCampaignParams{
		OrganizationID: orgID,
		Nombre:         "Cross org",
		SegmentID:      segment.ID,
	})
	if err == nil {
		t.Fatal("expected org-scoped FK violation for foreign segment")
	}
	if !isPgError(err, "23503") {
		t.Fatalf("expected FK violation (23503), got %v", err)
	}
}

func TestCampaignDownMigrationRollsBackCleanly(t *testing.T) {
	ctx := context.Background()

	// Down migration must drop the three tables and the module seed row.
	body, err := readMigration("000029_create_campaign_segments.down.sql")
	if err != nil {
		t.Fatalf("read down migration: %v", err)
	}
	if _, err := testPool.Exec(ctx, string(body)); err != nil {
		t.Fatalf("apply down migration: %v", err)
	}

	for _, table := range []string{"crm.campaign_recipients", "crm.campaigns", "crm.segments"} {
		var exists bool
		if err := testPool.QueryRow(ctx,
			"SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema='crm' AND table_name=$1)",
			table[len("crm."):],
		).Scan(&exists); err != nil {
			t.Fatalf("check table %s: %v", table, err)
		}
		if exists {
			t.Fatalf("expected table %s to be dropped", table)
		}
	}

	var moduleCount int
	if err := testPool.QueryRow(ctx, "SELECT COUNT(*) FROM modules.modules WHERE key='campaigns'").Scan(&moduleCount); err != nil {
		t.Fatalf("check module seed: %v", err)
	}
	if moduleCount != 0 {
		t.Fatalf("expected campaigns module seed removed, found %d", moduleCount)
	}

	// Re-apply the up migration so the rest of the suite stays intact.
	up, err := readMigration("000029_create_campaign_segments.up.sql")
	if err != nil {
		t.Fatalf("read up migration: %v", err)
	}
	if _, err := testPool.Exec(ctx, string(up)); err != nil {
		t.Fatalf("re-apply up migration: %v", err)
	}
}
