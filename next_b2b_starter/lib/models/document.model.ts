// lib/models/document.model.ts

export type DocumentStatus = "pending" | "processing" | "processed" | "failed";

export type DocumentVisibility = "workspace" | "admin_only";

export interface Document {
  id: number;
  title: string;
  fileName: string;
  contentType: string;
  fileSize: number;
  status: DocumentStatus;
  visibility: DocumentVisibility;
  extractedText?: string;
  metadata?: Record<string, unknown>;
  createdAt: Date;
  updatedAt: Date;
}

export interface DocumentListResponse {
  documents: Document[];
  total: number;
  limit: number;
  offset: number;
}

export interface DocumentListFilter {
  status?: DocumentStatus;
  limit?: number;
  offset?: number;
}

export const DocumentHelpers = {
  getStatusConfig: (status: DocumentStatus) => {
    const configs: Record<DocumentStatus, { label: string; color: string; bgColor: string }> = {
      pending: {
        label: "Pendiente",
        color: "text-amber-700",
        bgColor: "bg-amber-100 border-amber-200",
      },
      processing: {
        label: "Procesando",
        color: "text-amber-700",
        bgColor: "bg-amber-100 border-amber-200",
      },
      processed: {
        label: "Listo",
        color: "text-emerald-700",
        bgColor: "bg-emerald-100 border-emerald-200",
      },
      failed: {
        label: "Error",
        color: "text-red-700",
        bgColor: "bg-red-100 border-red-200",
      },
    };
    return configs[status] || configs.pending;
  },

  formatVisibility: (visibility: DocumentVisibility): string => {
    return visibility === "admin_only"
      ? "Solo administradores"
      : "Visible para todos";
  },

  /**
   * Legible similarity label — never raw JSON (app rule).
   */
  getSimilarityLabel: (score: number): string => {
    return `${Math.round(score * 100)}% coincidencia`;
  },

  formatFileSize: (bytes: number): string => {
    if (bytes === 0) return "0 B";
    const k = 1024;
    const sizes = ["B", "KB", "MB", "GB"];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`;
  },

  formatDate: (date: Date): string => {
    return new Intl.DateTimeFormat("es-CO", {
      month: "short",
      day: "numeric",
      year: "numeric",
    }).format(date);
  },
};
