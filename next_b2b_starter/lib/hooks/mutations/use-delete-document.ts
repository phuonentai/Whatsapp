// lib/hooks/mutations/use-delete-document.ts

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { queryKeys } from "../queries/query-keys";
import { documentRepository } from "@/lib/api/api/repositories/document-repository";

interface DeleteDocumentVariables {
  documentId: number;
}

export function useDeleteDocument() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ documentId }: DeleteDocumentVariables) =>
      documentRepository.deleteDocument(documentId),

    onSuccess: (_data, variables) => {
      // Invalidate documents list to refetch without deleted document
      queryClient.invalidateQueries({
        queryKey: queryKeys.documents.all,
      });
    },

    onError: (error, variables) => {
      console.error("[Mutation] Document deletion failed:", {
        error,
        documentId: variables.documentId,
      });
    },
  });
}
