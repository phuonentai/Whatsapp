// lib/api/api/repositories/document-repository.ts

import { apiClient } from "../client/api-client";
import {
  DocumentDto,
  ListDocumentsResponseDto,
  UploadDocumentResponseDto,
} from "../dto/document.dto";
import {
  Document,
  DocumentListResponse,
  DocumentListFilter,
  DocumentStatus,
} from "@/lib/models/document.model";

class DocumentRepository {
  /**
   * Upload a document (PDF)
   */
  async uploadDocument(file: File, title: string): Promise<Document> {
    const formData = new FormData();
    formData.append("file", file);
    formData.append("title", title);

    const response = await apiClient.post<UploadDocumentResponseDto>(
      "/example_documents/upload",
      formData
    );

    return this.toDocument(response);
  }

  /**
   * List documents with optional filters
   */
  async listDocuments(options?: DocumentListFilter): Promise<DocumentListResponse> {
    const params = new URLSearchParams();

    if (options?.status) {
      params.append("status", options.status);
    }
    if (options?.limit) {
      params.append("limit", String(options.limit));
    }
    if (options?.offset) {
      params.append("offset", String(options.offset));
    }

    const queryString = params.toString();
    const endpoint = queryString
      ? `/example_documents?${queryString}`
      : "/example_documents";

    type ListDocumentsApiResponse =
      | ListDocumentsResponseDto
      | {
          data?: ListDocumentsResponseDto;
          documents?: DocumentDto[];
          total?: number;
        };

    const response = await apiClient.get<ListDocumentsApiResponse>(endpoint);

    // Handle flexible response format
    const dto: ListDocumentsResponseDto =
      "data" in response && response.data
        ? response.data
        : {
            documents:
              (response as { documents?: DocumentDto[] }).documents ??
              (response as ListDocumentsResponseDto).documents ??
              [],
            total:
              (response as { total?: number }).total ??
              (response as ListDocumentsResponseDto).total ??
              0,
            limit: options?.limit ?? 50,
            offset: options?.offset ?? 0,
          };

    return {
      documents: (dto.documents ?? []).map((item) => this.toDocument(item)),
      total: dto.total ?? dto.documents?.length ?? 0,
      limit: dto.limit,
      offset: dto.offset,
    };
  }

  /**
   * Delete a document
   */
  async deleteDocument(id: number): Promise<boolean> {
    await apiClient.delete(`/example_documents/${id}`);
    return true;
  }

  /**
   * Transform DTO to Document model
   */
  private toDocument(dto: DocumentDto | UploadDocumentResponseDto): Document {
    return {
      id: dto.id,
      title: dto.title,
      fileName: dto.file_name,
      contentType: dto.content_type,
      fileSize: dto.file_size,
      status: dto.status as DocumentStatus,
      extractedText: "extracted_text" in dto ? dto.extracted_text : undefined,
      metadata: "metadata" in dto ? dto.metadata : undefined,
      createdAt: new Date(dto.created_at),
      updatedAt: new Date(dto.updated_at),
    };
  }
}

export const documentRepository = new DocumentRepository();
