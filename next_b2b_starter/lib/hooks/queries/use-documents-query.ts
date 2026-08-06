// lib/hooks/queries/use-documents-query.ts

import { useQuery, type UseQueryOptions } from "@tanstack/react-query";
import { queryKeys } from "./query-keys";
import { documentRepository } from "@/lib/api/api/repositories/document-repository";
import type { DocumentListResponse, DocumentListFilter } from "@/lib/models/document.model";

interface UseDocumentsQueryOptions {
  status?: DocumentListFilter["status"];
  limit?: number;
  offset?: number;
  enabled?: boolean;
}

export function useDocumentsQuery(
  options: UseDocumentsQueryOptions = {},
  queryOptions?: Omit<
    UseQueryOptions<DocumentListResponse, Error>,
    "queryKey" | "queryFn" | "enabled"
  >
) {
  const { status, limit = 50, offset = 0, enabled = true } = options;

  const filters: DocumentListFilter = { status, limit, offset };

  return useQuery({
    queryKey: queryKeys.documents.list(filters),
    queryFn: () => documentRepository.listDocuments(filters),
    enabled,
    staleTime: 2 * 60 * 1000, // 2 minutes
    gcTime: 5 * 60 * 1000, // 5 minutes cache
    retry: 1,
    refetchOnWindowFocus: false,
    ...queryOptions,
  });
}

/**
 * Convenience hook to get documents array directly
 */
export function useDocuments(options: UseDocumentsQueryOptions = {}) {
  const { data } = useDocumentsQuery(options);
  return data?.documents ?? [];
}

/**
 * Hook to get total document count
 */
export function useDocumentCount(options: UseDocumentsQueryOptions = {}) {
  const { data } = useDocumentsQuery(options);
  return data?.total ?? 0;
}
