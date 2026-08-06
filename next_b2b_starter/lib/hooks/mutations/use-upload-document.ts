// lib/hooks/mutations/use-upload-document.ts

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { queryKeys } from "../queries/query-keys";
import { documentRepository } from "@/lib/api/api/repositories/document-repository";
import type { Document } from "@/lib/models/document.model";

interface UploadDocumentVariables {
  file: File;
  title: string;
}

export function useUploadDocument() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ file, title }: UploadDocumentVariables) =>
      documentRepository.uploadDocument(file, title),

    onSuccess: (data: Document) => {
      // Invalidate documents list to refetch with new document
      queryClient.invalidateQueries({
        queryKey: queryKeys.documents.all,
      });
    },

    onError: (error, variables) => {
      console.error("[Mutation] Document upload failed:", {
        error,
        fileName: variables.file.name,
      });
    },
  });
}
