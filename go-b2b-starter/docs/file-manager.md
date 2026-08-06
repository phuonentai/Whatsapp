# File Manager Guide

The file manager provides file storage using Cloudflare R2 (object storage) with PostgreSQL for searchable metadata.

## Architecture

**Dual-layer design:**

**R2 Storage** - Stores actual file content
**PostgreSQL** - Stores searchable metadata

This separation enables fast querying while leveraging object storage scalability.

## Components

**FileRepository**: Combined operations (upload, download, delete, search)
**R2Repository**: R2 object storage operations
**FileMetadataRepository**: Database metadata operations
**FileService**: Business logic with validation

## File Upload

### Basic Upload

```go
req := &domain.FileUploadRequest{
    Filename:    "document.pdf",
    ContentType: "application/pdf",
    Context:     file_manager.ContextDocument,
}

file, err := fileService.UploadFile(ctx, req, fileReader)
```

### Upload with Entity Linking

Link files to domain entities (like resources, users, etc.):

```go
req := &domain.FileUploadRequest{
    Filename:    "profile.jpg",
    ContentType: "image/jpeg",
    Context:     file_manager.ContextProfile,
}

file := &domain.FileAsset{
    EntityType: "user",
    EntityID:   userID,
}

uploadedFile, err := fileService.UploadFile(ctx, req, fileReader)
```

### Upload Flow

1. Validate file (size, type, magic bytes)
2. Save metadata to database (get ID)
3. Upload content to R2 (using database ID in key)
4. Update metadata with storage path
5. Rollback on failure (atomic operation)

## File Download

### Get Presigned URL

Generate temporary download link:

```go
url, err := fileService.GetPresignedURL(ctx, fileID, 15*time.Minute)
```

Returns a time-limited URL for direct download from R2.

### Download File Content

```go
content, err := fileService.DownloadFile(ctx, fileID)
```

Returns `io.ReadCloser` with file content.

## File Search

### By Entity

Get all files for a specific entity:

```go
files, err := fileRepo.GetByEntity(ctx, "resource", resourceID)
```

### By Category

Find files by category:

```go
documents, err := fileRepo.GetByCategory(ctx, file_manager.CategoryDocument, 10, 0)
```

### By Context

Search by context type:

```go
profiles, err := fileRepo.GetByContext(ctx, file_manager.ContextProfile, 20, 0)
```

## File Validation

Automatic validation on upload:

**Magic byte verification** - Validates file type matches content
**Size limits** - Configurable max file size
**Content type** - Ensures valid MIME type

Configure in `FileService` initialization.

## Contexts and Categories

### Predefined Contexts

- `ContextDocument` - General documents
- `ContextProfile` - Profile images
- `ContextAttachment` - Email/message attachments
- `ContextThumbnail` - Image thumbnails

### Categories

- `CategoryDocument` - PDFs, docs
- `CategoryImage` - Images
- `CategoryVideo` - Videos
- `CategoryArchive` - ZIP, TAR files

Defined in `internal/files/domain/constants.go`.

## Configuration

```env
# Cloudflare R2
R2_ACCOUNT_ID=your-account-id
R2_ACCESS_KEY_ID=your-access-key
R2_SECRET_ACCESS_KEY=your-secret-key
R2_BUCKET_NAME=files
R2_REGION=auto  # Usually "auto" for R2
```

## Common Patterns

### Upload User Avatar

```go
func (s *service) UpdateAvatar(ctx context.Context, userID int32, avatar io.Reader) error {
    req := &domain.FileUploadRequest{
        Filename:    fmt.Sprintf("avatar_%d.jpg", userID),
        ContentType: "image/jpeg",
        Context:     file_manager.ContextProfile,
    }

    file, err := s.fileService.UploadFile(ctx, req, avatar)
    if err != nil {
        return err
    }

    // Link to user
    return s.userRepo.UpdateAvatar(ctx, userID, file.ID)
}
```

### Get Entity Files

```go
func (h *Handler) GetResourceFiles(c *gin.Context) {
    resourceID := parseID(c.Param("id"))

    files, err := h.fileRepo.GetByEntity(c.Request.Context(), "resource", resourceID)
    if err != nil {
        c.JSON(500, gin.H{"error": "failed to get files"})
        return
    }

    c.JSON(200, files)
}
```

### Delete File

```go
func (s *service) DeleteResource(ctx context.Context, resourceID int32) error {
    // Get associated files
    files, err := s.fileRepo.GetByEntity(ctx, "resource", resourceID)
    if err != nil {
        return err
    }

    // Delete files
    for _, file := range files {
        err = s.fileService.DeleteFile(ctx, file.ID)
        if err != nil {
            return err
        }
    }

    // Delete resource
    return s.resourceRepo.Delete(ctx, resourceID)
}
```

## File Locations

| Component | Path |
|-----------|------|
| Domain entities | `internal/files/domain/` |
| File service | `internal/files/internal/app/` |
| R2 repository | `internal/files/internal/infra/r2/` |
| Metadata repository | `internal/files/internal/infra/metadata/` |
| Constants | `internal/files/domain/constants.go` |

## Next Steps

- **Upload files**: Integrate file upload in your features
- **Link entities**: Associate files with domain objects
- **R2 documentation**: https://developers.cloudflare.com/r2/
