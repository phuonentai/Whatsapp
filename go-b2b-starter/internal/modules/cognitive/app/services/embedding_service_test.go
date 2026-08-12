package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/moasq/go-b2b-starter/internal/modules/cognitive/domain"
)

// mockEmbeddingRepository records SearchSimilar calls.
type mockEmbeddingRepository struct {
	domain.EmbeddingRepository
	searchCalls []struct {
		orgID            int32
		limit            int32
		includeAdminOnly bool
	}
}

func (m *mockEmbeddingRepository) SearchSimilar(ctx context.Context, orgID int32, embedding []float64, limit int32, includeAdminOnly bool) ([]*domain.SimilarDocument, error) {
	m.searchCalls = append(m.searchCalls, struct {
		orgID            int32
		limit            int32
		includeAdminOnly bool
	}{orgID, limit, includeAdminOnly})
	return nil, nil
}

type staticVectorizer struct{}

func (staticVectorizer) Vectorize(ctx context.Context, text string) ([]float64, error) {
	return []float64{0.1, 0.2}, nil
}

func TestSearchSimilarDocumentsForwardsACLFlag(t *testing.T) {
	repo := &mockEmbeddingRepository{}
	svc := NewEmbeddingService(repo, staticVectorizer{})

	_, err := svc.SearchSimilarDocuments(context.Background(), 7, "pregunta", 3, false)
	require.NoError(t, err)
	_, err = svc.SearchSimilarDocuments(context.Background(), 7, "pregunta", 3, true)
	require.NoError(t, err)

	require.Len(t, repo.searchCalls, 2)
	assert.Equal(t, int32(7), repo.searchCalls[0].orgID)
	assert.Equal(t, false, repo.searchCalls[0].includeAdminOnly)
	assert.Equal(t, true, repo.searchCalls[1].includeAdminOnly)
}
