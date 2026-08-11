// Command seed-e2e seeds the e2e test database with the orgs, accounts, and
// subscriptions that the Playwright suite (next_b2b_starter/e2e) depends on.
//
// The mock auth middleware resolves the X-Test-Org-ID header value
// ("<slug>:<email>") via stytch_org_id and email, so each seeded org's
// stytch_org_id SHALL equal its slug.
//
// Usage:
//
//	POSTGRES_HOST=localhost POSTGRES_PORT=5432 POSTGRES_USER=postgres \
//	POSTGRES_PASSWORD=postgres POSTGRES_DB=saas_db_test \
//	AUTH_MOCK_ENABLED=true go run ./cmd/seed-e2e
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const planFree = "free"
const planPro = "pro"
const planEnterprise = "enterprise"

type seedOrg struct {
	slug     string
	plan     string
	accounts []seedAccountRow
}

type seedAccountRow struct {
	email    string
	fullName string
	role     string
}

var seedOrgs = []seedOrg{
	{
		slug: "test-org-free",
		plan: planFree,
		accounts: []seedAccountRow{
			{email: "admin-free@test.com", fullName: "Free Admin", role: "admin"},
		},
	},
	{
		slug: "test-org-pro",
		plan: planPro,
		accounts: []seedAccountRow{
			{email: "admin-pro@test.com", fullName: "Pro Admin", role: "admin"},
			{email: "member-pro@test.com", fullName: "Pro Member", role: "member"},
		},
	},
	{
		slug: "test-org-enterprise",
		plan: planEnterprise,
		accounts: []seedAccountRow{
			{email: "admin-enterprise@test.com", fullName: "Enterprise Admin", role: "admin"},
		},
	},
	{
		slug: "test-org-rbac",
		plan: planPro,
		accounts: []seedAccountRow{
			{email: "admin-rbac@test.com", fullName: "RBAC Admin", role: "admin"},
			{email: "manager-rbac@test.com", fullName: "RBAC Manager", role: "member"},
			{email: "member-rbac@test.com", fullName: "RBAC Member", role: "member"},
		},
	},
	{
		// Dedicated org for the Siigo onboarding e2e suite: connection and
		// import state stays isolated from the general-purpose orgs.
		slug: "test-org-siigo",
		plan: planPro,
		accounts: []seedAccountRow{
			{email: "admin-siigo@test.com", fullName: "Siigo Admin", role: "admin"},
			{email: "member-siigo@test.com", fullName: "Siigo Member", role: "member"},
		},
	},
}

func planMetadata(plan string) []byte {
	features := map[string]any{
		"plan": plan,
	}
	switch plan {
	case planPro:
		features["crm_features"] = "crm_contacts_manage,crm_companies,crm_deals,crm_activities"
	case planEnterprise:
		features["crm_features"] = "crm_contacts_manage,crm_companies,crm_deals,crm_activities,crm_tags"
	case planFree:
		features["crm_features"] = "crm_contacts_manage"
	}
	raw, _ := json.Marshal(features)
	return raw
}

func main() {
	ctx := context.Background()
	host := envOr("POSTGRES_HOST", "localhost")
	port := envOr("POSTGRES_PORT", "5432")
	user := envOr("POSTGRES_USER", "postgres")
	password := envOr("POSTGRES_PASSWORD", "postgres")
	dbname := envOr("POSTGRES_DB", "saas_db_test")

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	if os.Getenv("SKIP_MIGRATIONS") != "true" {
		if err := runMigrations(ctx, pool); err != nil {
			log.Fatalf("migrate: %v", err)
		}
	}

	for _, org := range seedOrgs {
		if err := seedOrgRow(ctx, pool, org); err != nil {
			log.Fatalf("seed org %s: %v", org.slug, err)
		}
	}

	log.Printf("seed-e2e complete: %d orgs", len(seedOrgs))
}

// runMigrations executes every *.up.sql migration file in order. It mirrors
// postgres.PostgresManager.RunMigrations (which is never invoked at server
// startup), keeping the e2e test DB in sync with the migration tree.
func runMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	dir := envOr("MIGRATION_URL", "internal/db/postgres/sqlc/migrations")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read migration dir %s: %w", dir, err)
	}

	var files []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".up.sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	for _, name := range files {
		content, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}
		if _, err := pool.Exec(ctx, string(content)); err != nil {
			return fmt.Errorf("execute %s: %w", name, err)
		}
		log.Printf("applied migration: %s", name)
	}
	return nil
}

func seedOrgRow(ctx context.Context, pool *pgxpool.Pool, org seedOrg) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var orgID int
	err = tx.QueryRow(ctx,
		`SELECT id FROM organizations.organizations WHERE slug = $1`,
		org.slug,
	).Scan(&orgID)
	if err == pgx.ErrNoRows {
		err = tx.QueryRow(ctx,
			`INSERT INTO organizations.organizations (slug, name, status, stytch_org_id)
			 VALUES ($1, $2, 'active', $3) RETURNING id`,
			org.slug, "E2E "+org.slug, org.slug,
		).Scan(&orgID)
	}
	if err != nil {
		return fmt.Errorf("upsert org: %w", err)
	}

	// Ensure stytch_org_id always matches the slug (mock auth resolution key)
	_, err = tx.Exec(ctx,
		`UPDATE organizations.organizations SET stytch_org_id = $2 WHERE id = $1`,
		orgID, org.slug,
	)
	if err != nil {
		return fmt.Errorf("set stytch_org_id: %w", err)
	}

	for _, acc := range org.accounts {
		if err := seedAccount(ctx, tx, orgID, acc); err != nil {
			return err
		}
	}

	// Subscription drives feature gating / entitlement
	now := time.Now()
	_, err = tx.Exec(ctx, `
		INSERT INTO subscription_billing.subscriptions (
			organization_id, external_customer_id, subscription_id,
			subscription_status, product_id, product_name, plan_name,
			current_period_start, current_period_end, cancel_at_period_end, metadata
		) VALUES ($1, $2, $3, 'active', $4, $5, $6, $7, $8, false, $9)
		ON CONFLICT (organization_id) DO UPDATE SET
			subscription_status = EXCLUDED.subscription_status,
			plan_name = EXCLUDED.plan_name,
			metadata = EXCLUDED.metadata,
			current_period_end = EXCLUDED.current_period_end`,
		orgID,
		"e2e-customer-"+org.slug,
		"e2e-sub-"+org.slug,
		"e2e-product-"+org.slug,
		org.slug+" plan",
		org.plan,
		now,
		now.AddDate(1, 0, 0),
		planMetadata(org.plan),
	)
	if err != nil {
		return fmt.Errorf("upsert subscription: %w", err)
	}

	// Quota row: GetQuotaStatus (used by the paywall middleware) INNER JOINs
	// quota_tracking and marks the subscription active only when
	// invoice_count > 0, so every seeded org gets invoice_count = 1.
	_, err = tx.Exec(ctx, `
		INSERT INTO subscription_billing.quota_tracking (
			organization_id, invoice_count, max_seats, period_start, period_end
		) VALUES ($1, 1, 10, $2, $3)
		ON CONFLICT (organization_id) DO UPDATE SET
			invoice_count = EXCLUDED.invoice_count,
			period_end = EXCLUDED.period_end`,
		orgID, now, now.AddDate(1, 0, 0),
	)
	if err != nil {
		return fmt.Errorf("upsert quota: %w", err)
	}

	// Agent: kill_switch=true so the LLM pipeline never runs during e2e.
	// Without it every inbound webhook triggers the metered OpenAI call
	// (placeholder key → 401 + retry), which stalls the synchronous webhook
	// handler and flaky-timeouts the whatsapp-inbox idempotency spec.
	_, err = tx.Exec(ctx, `
		INSERT INTO agent.agent_settings (
			organization_id, mode, tone, timezone, kill_switch,
			consent_required, guardrails
		) VALUES ($1, 'copilot', 'formal', 'America/Bogota', true, true, '{}'::jsonb)
		ON CONFLICT (organization_id) DO UPDATE SET
			kill_switch = EXCLUDED.kill_switch`,
		orgID,
	)
	if err != nil {
		return fmt.Errorf("upsert agent settings: %w", err)
	}

	return tx.Commit(ctx)
}

func seedAccount(ctx context.Context, tx pgx.Tx, orgID int, acc seedAccountRow) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO organizations.accounts (
			organization_id, email, full_name, stytch_member_id,
			stytch_role_slug, stytch_email_verified, role, status
		) VALUES ($1, $2, $3, $4, $5, true, $6, 'active')
		ON CONFLICT (organization_id, email) DO UPDATE SET
			full_name = EXCLUDED.full_name,
			role = EXCLUDED.role,
			status = 'active'`,
		orgID, acc.email, acc.fullName, "mock-"+acc.email, acc.role, acc.role,
	)
	if err != nil {
		return fmt.Errorf("upsert account %s: %w", acc.email, err)
	}
	return nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
