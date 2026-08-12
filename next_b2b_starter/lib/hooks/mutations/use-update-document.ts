// lib/hooks/mutations/use-update-document.ts

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { queryKeys } from "../queries/query-keys";
import { documentRepository } from "@/lib/api/api/repositories/document-repository";
import type { UpdateDocumentRequestDto } from "@/lib/api/api/dto/document.dto";
import type { Document } from "@/lib/models/document.model";

interface UpdateDocumentVariables {
  documentId: number;
  fields: UpdateDocumentRequestDto;
}

export function useUpdateDocument() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ documentId, fields }: UpdateDocumentVariables) =>
      documentRepository.updateDocument(documentId, fields),

    onSuccess: (_data: Document, variables) => {
      // Invalidate documents list so the new title/visibility is reflected.
      queryClient.invalidateQueries({
        queryKey: queryKeys.documents.all,
      });
      void variables;
    },

    onError: (error, variables) => {
      console.error("[Mutation] Document update failed:", {
        error,
        documentId: variables.documentId,
      });
    },
  });
}
