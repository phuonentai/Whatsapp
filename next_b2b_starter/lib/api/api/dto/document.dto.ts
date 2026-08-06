// lib/api/api/dto/document.dto.ts

export type DocumentStatus = "pending" | "processing" | "processed" | "failed";

export interface DocumentDto {
  id: number;
  title: string;
  file_name: string;
  content_type: string;
  file_size: number;
  status: DocumentStatus;
  extracted_text?: string;
  metadata?: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface ListDocumentsRequestDto {
  status?: DocumentStatus;
  limit?: number;
  offset?: number;
}

export interface ListDocumentsResponseDto {
  documents: DocumentDto[];
  total: number;
  limit: number;
  offset: number;
}

export interface UploadDocumentResponseDto {
  id: number;
  title: string;
  file_name: string;
  content_type: string;
  file_size: number;
  status: DocumentStatus;
  created_at: string;
  updated_at: string;
}
