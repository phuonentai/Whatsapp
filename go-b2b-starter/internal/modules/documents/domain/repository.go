package domain

import "context"

// DocumentRepository defines the interface for document data operations
type DocumentRepository interface {
	// Create creates a new document
	Create(ctx context.Context, doc *Document) (*Document, error)

	// GetByID retrieves a document by ID.
	// includeAdminOnly=false treats admin_only documents as nonexistent
	// (member without org:manage — no title leak).
	GetByID(ctx context.Context, orgID, docID int32, includeAdminOnly bool) (*Document, error)

	// GetByFileAssetID retrieves a document by file asset ID
	GetByFileAssetID(ctx context.Context, orgID, fileAssetID int32) (*Document, error)

	// List retrieves documents with pagination, filtered by visibility.
	List(ctx context.Context, orgID int32, limit, offset int32, includeAdminOnly bool) ([]*Document, error)

	// ListByStatus retrieves documents by status with pagination, filtered by
	// visibility.
	ListByStatus(ctx context.Context, orgID int32, status DocumentStatus, limit, offset int32, includeAdminOnly bool) ([]*Document, error)

	// UpdateStatus updates the document status
	UpdateStatus(ctx context.Context, orgID, docID int32, status DocumentStatus) (*Document, error)

	// UpdateExtractedText updates the extracted text and sets status to processed
	UpdateExtractedText(ctx context.Context, orgID, docID int32, text string) (*Document, error)

	// Update updates document metadata
	Update(ctx context.Context, doc *Document) (*Document, error)

	// Delete removes a document
	Delete(ctx context.Context, orgID, docID int32) error

	// Count returns the total count of documents for an organization, filtered
	// by visibility.
	Count(ctx context.Context, orgID int32, includeAdminOnly bool) (int64, error)

	// CountByStatus returns the count of documents with a specific status,
	// filtered by visibility.
	CountByStatus(ctx context.Context, orgID int32, status DocumentStatus, includeAdminOnly bool) (int64, error)

	// ListIndexedForCompliance returns documents that contributed chunks to the
	// org's RAG index (have at least one embedding), for the Ley 1581 export.
	ListIndexedForCompliance(ctx context.Context, orgID int32) ([]ComplianceDocument, error)
}
