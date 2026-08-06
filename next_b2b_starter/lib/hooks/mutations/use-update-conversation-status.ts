import { useMutation, useQueryClient } from "@tanstack/react-query";
import { conversationRepository } from "@/lib/api/api/repositories/conversation-repository";
import { queryKeys } from "../queries/query-keys";
import type { ConversationStatus } from "@/lib/models/conversation.model";

export function useUpdateConversationStatus() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ conversationId, status }: { conversationId: number; status: ConversationStatus }) =>
      conversationRepository.updateStatus(conversationId, status),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.crm.conversations() });
    },
  });
}
