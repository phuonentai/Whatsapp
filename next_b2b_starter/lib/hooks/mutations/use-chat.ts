// lib/hooks/mutations/use-chat.ts

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { queryKeys } from "../queries/query-keys";
import { cognitiveRepository } from "@/lib/api/api/repositories/cognitive-repository";
import type { ChatRequest, ChatResponse } from "@/lib/models/cognitive.model";

export function useChat() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (request: ChatRequest) => cognitiveRepository.chat(request),

    onSuccess: (data: ChatResponse) => {
      // Invalidate sessions list to include new session if created
      queryClient.invalidateQueries({
        queryKey: queryKeys.cognitive.sessions(),
      });

      // Invalidate messages for this session
      if (data.sessionId) {
        queryClient.invalidateQueries({
          queryKey: queryKeys.cognitive.messages(data.sessionId),
        });
      }
    },

    onError: (error, variables) => {
      console.error("[Mutation] Chat message failed:", {
        error,
        sessionId: variables.sessionId,
      });
    },
  });
}
