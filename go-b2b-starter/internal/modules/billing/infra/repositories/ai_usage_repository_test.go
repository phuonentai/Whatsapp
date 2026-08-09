package repositories

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
	"github.com/moasq/go-b2b-starter/internal/modules/billing/domain"
)

// fakeAiUsageStore embeds sqlc.Store and simulates the idempotent event insert
// semantics (ON CONFLICT DO NOTHING): a duplicate (org, request_id) returns
// 0 rows affected.
type fakeAiUsageStore struct {
	sqlc.Store
	events     map[string]bool // key: orgID|requestID
	upserts    int
	err        error
	failUpsert bool
}

func (f *fakeAiUsageStore) InsertAiUsageEvent(ctx context.Context, arg sqlc.InsertAiUsageEventParams) (int64, error) {
	if f.err != nil {
		return 0, f.err
	}
	key := strconv.Itoa(int(arg.OrganizationID)) + "|" + arg.RequestID
	if f.events[key] {
		return 0, nil
	}
	f.events[key] = true
	return 1, nil
}

func (f *fakeAiUsageStore) UpsertAiUsage(ctx context.Context, arg sqlc.UpsertAiUsageParams) (sqlc.SubscriptionBillingAiUsage, error) {
	f.upserts++
	if f.failUpsert {
		return sqlc.SubscriptionBillingAiUsage{}, assert.AnError
	}
	return sqlc.SubscriptionBillingAiUsage{
		OrganizationID:  arg.OrganizationID,
		PeriodStart:     arg.PeriodStart,
		PeriodEnd:       arg.PeriodEnd,
		TokensInput:     arg.TokensInput,
		TokensOutput:    arg.TokensOutput,
		TokensEmbedding: arg.TokensEmbedding,
		CreditsUsed:     arg.CreditsUsed,
	}, nil
}

func newFakeAiUsageStore() *fakeAiUsageStore {
	return &fakeAiUsageStore{events: make(map[string]bool)}
}

func TestAiUsageRepository_RecordUsage_Success(t *testing.T) {
	store := newFakeAiUsageStore()
	repo := NewAiUsageRepository(store)

	recorded, err := repo.RecordUsage(context.Background(), &domain.AiUsageEvent{
		OrganizationID:  1,
		Feature:         "rag_chat",
		Model:           "gpt-5-mini",
		TokensInput:     100,
		TokensOutput:    50,
		CreditsConsumed: 1,
		RequestID:       "req-1",
	}, time.Now(), time.Now().Add(time.Hour))

	require.NoError(t, err)
	assert.True(t, recorded)
	assert.Equal(t, 1, store.upserts)
}

func TestAiUsageRepository_RecordUsage_DuplicateRequestID(t *testing.T) {
	store := newFakeAiUsageStore()
	repo := NewAiUsageRepository(store)

	event := &domain.AiUsageEvent{
		OrganizationID:  1,
		Feature:         "rag_chat",
		Model:           "gpt-5-mini",
		TokensInput:     100,
		CreditsConsumed: 1,
		RequestID:       "req-dup",
	}
	now := time.Now()

	first, err := repo.RecordUsage(context.Background(), event, now, now.Add(time.Hour))
	require.NoError(t, err)
	require.True(t, first)

	second, err := repo.RecordUsage(context.Background(), event, now, now.Add(time.Hour))
	require.NoError(t, err)
	assert.False(t, second, "duplicate request_id must not record again")
	assert.Equal(t, 1, store.upserts, "totals must be incremented exactly once")
}

func TestAiUsageRepository_RecordUsage_PropagatesEventError(t *testing.T) {
	store := newFakeAiUsageStore()
	store.err = assert.AnError
	repo := NewAiUsageRepository(store)

	_, err := repo.RecordUsage(context.Background(), &domain.AiUsageEvent{
		OrganizationID: 1,
		RequestID:       "req-x",
	}, time.Now(), time.Now())

	require.Error(t, err)
}

func TestAiUsageRepository_RecordUsage_PropagatesUpsertError(t *testing.T) {
	store := newFakeAiUsageStore()
	store.failUpsert = true
	repo := NewAiUsageRepository(store)

	_, err := repo.RecordUsage(context.Background(), &domain.AiUsageEvent{
		OrganizationID: 1,
		RequestID:       "req-y",
	}, time.Now(), time.Now())

	require.Error(t, err)
}
