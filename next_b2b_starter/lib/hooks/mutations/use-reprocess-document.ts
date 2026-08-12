// lib/hooks/mutations/use-reprocess-document.ts

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { queryKeys } from "../queries/query-keys";
import { documentRepository } from "@/lib/api/api/repositories/document-repository";

interface ReprocessDocumentVariables {
  documentId: number;
}

export function useReprocessDocument() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ documentId }: ReprocessDocumentVariables) =>
      documentRepository.reprocessDocument(documentId),

    onSuccess: (_data, variables) => {
      // Invalidate documents list to reflect the new processing status.
      queryClient.invalidateQueries({
        queryKey: queryKeys.documents.all,
      });
      void variables;
    },

    onError: (error, variables) => {
      console.error("[Mutation] Document reprocess failed:", {
        error,
        documentId: variables.documentId,
      });
    },
  });
}
