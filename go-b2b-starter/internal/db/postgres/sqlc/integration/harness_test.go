//go:build integration

package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/moasq/go-b2b-starter/internal/db/helpers"
	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	testStore sqlc.Store
	testPool  *pgxpool.Pool
)

// migrationsToApply lists every .up.sql file in the migrations directory in
// apply order. The migrate CLI cannot apply this migration set because older
// version prefixes were reused during renumbering; the harness applies the
// files directly instead.
var migrationsToApply = []string{
	"000001_create_file_manager_schema.up.sql",
	"000002_create_organizations_schema.up.sql",
	"000003_enforce_role_enum.up.sql",
	"000004_create_subscription_billing_schema.up.sql",
	"000005_update_quota_tracking_schema.up.sql",
	"000006_create_example_resources.up.sql",
	"000007_create_resource_embeddings.up.sql",
	"000008_create_documents_schema.up.sql",
	"000009_create_cognitive_schema.up.sql",
	"000010_create_whatsapp_crm_schema.up.sql",
	"000011_extend_crm_contacts.up.sql",
	"000012_create_crm_companies_pipelines_deals.up.sql",
	"000013_create_crm_activities_tags.up.sql",
	"000014_add_whatsapp_config_outbound_fields.up.sql",
	"000015_add_billing_provider_to_organizations.up.sql",
	"000016_create_crm_integrity_constraints.up.sql",
	"000017_create_modules_tickets.up.sql",
	"000018_create_ai_usage_ledger.up.sql",
	"000019_create_agent_schema.up.sql",
	"000020_create_playbooks.up.sql",
	"000021_create_invoices.up.sql",
	"000022_add_tenant_isolation.up.sql",
	"000023_create_whatsapp_signup_flows.up.sql",
	"000024_make_webhook_logs_org_nullable.up.sql",
	"000025_update_playbook_sequence_seeds.up.sql",
	"000026_create_outbox_events.up.sql",
	"000027_add_webhook_logs_delivery_key.up.sql",
	"000028_create_org_connections.up.sql",
	"000029_create_campaign_segments.up.sql",
	"000030_onboarding_data.up.sql",
	"000031_create_client_payments.up.sql",
	"000032_seed_analytics_module.up.sql",
	"000033_add_instagram_schema.up.sql",
	"000034_create_conversation_contexts.up.sql",
	"000035_add_campaign_message.up.sql",
	"000036_create_whatsapp_templates.up.sql",
	"000037_create_procurement_schema.up.sql",
}

func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "pgvector/pgvector:pg16",
			ExposedPorts: []string{"5432/tcp"},
			Env: map[string]string{
				"POSTGRES_USER":     "postgres",
				"POSTGRES_PASSWORD": "postgres",
				"POSTGRES_DB":       "postgres",
			},
			WaitingFor: wait.ForListeningPort("5432/tcp").WithStartupTimeout(120 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to start postgres container: %v\n", err)
		os.Exit(1)
	}

	mappedPort, err := container.MappedPort(ctx, "5432/tcp")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get mapped port: %v\n", err)
		os.Exit(1)
	}
	host, err := container.Host(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get host: %v\n", err)
		os.Exit(1)
	}

	dsn := fmt.Sprintf("postgres://postgres:postgres@%s:%s/postgres?sslmode=disable", host, mappedPort.Port())
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create pool: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Ensure the server is fully ready before applying migrations.
	for i := 0; i < 30; i++ {
		if err := pool.Ping(ctx); err == nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	if err := applyMigrations(ctx, pool); err != nil {
		fmt.Fprintf(os.Stderr, "failed to apply migrations: %v\n", err)
		os.Exit(1)
	}

	testPool = pool
	testStore = sqlc.NewStore(pool)

	code := m.Run()
	os.Exit(code)
}

// applyMigrations executes the up migrations in explicit order. It cannot use
// golang-migrate because the repo's migration files contain a duplicated
// version prefix (000002), which the library rejects.
func applyMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		return fmt.Errorf("cannot resolve migrations directory")
	}
	migrationsDir := filepath.Join(filepath.Dir(thisFile), "..", "migrations")

	files := make([]string, 0, len(migrationsToApply))
	for _, name := range migrationsToApply {
		full := filepath.Join(migrationsDir, name)
		if _, err := os.Stat(full); err != nil {
			return fmt.Errorf("migration %s missing: %w", name, err)
		}
		files = append(files, full)
	}
	sort.Strings(files)

	for _, f := range files {
		body, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", f, err)
		}
		if _, err := pool.Exec(ctx, string(body)); err != nil {
			return fmt.Errorf("apply migration %s: %w", filepath.Base(f), err)
		}
	}
	return nil
}

// orgID is the id of the first organization created by createOrgWithAccount.
var orgSeq int32 = 1

// createOrgWithAccount creates a fresh org + account for test isolation and
// returns their ids.
func createOrgWithAccount(t *testing.T, ctx context.Context, q sqlc.Querier) (int32, int32) {
	t.Helper()
	org, err := q.CreateOrganization(ctx, sqlc.CreateOrganizationParams{
		Slug:   fmt.Sprintf("org-it-%d-%d", orgSeq, time.Now().UnixNano()),
		Name:   fmt.Sprintf("Org IT %d", orgSeq),
		Status: "active",
	})
	if err != nil {
		t.Fatalf("create org: %v", err)
	}
	orgSeq++
	acc, err := q.CreateAccount(ctx, sqlc.CreateAccountParams{
		OrganizationID: org.ID,
		Email:          fmt.Sprintf("owner%d@example.com", org.ID),
		FullName:       "Owner",
		Role:           "owner",
		Status:         "active",
		StytchMemberID: helpers.ToPgText(""),
		StytchRoleID:   helpers.ToPgText(""),
		StytchRoleSlug: helpers.ToPgText(""),
	})
	if err != nil {
		t.Fatalf("create account: %v", err)
	}
	return org.ID, acc.ID
}

func isPgError(err error, code string) bool {
	if err == nil {
		return false
	}
	pgErr, ok := unwrapPgError(err)
	if !ok {
		return false
	}
	return pgErr.Code == code
}

func unwrapPgError(err error) (*pgconn.PgError, bool) {
	for err != nil {
		pgErr, ok := err.(*pgconn.PgError)
		if ok {
			return pgErr, true
		}
		type unwrapper interface{ Unwrap() error }
		u, ok := err.(unwrapper)
		if !ok {
			return nil, false
		}
		err = u.Unwrap()
	}
	return nil, false
}
