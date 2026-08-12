import { useMutation, useQueryClient } from "@tanstack/react-query";
import { conversationRepository } from "@/lib/api/api/repositories/conversation-repository";
import { queryKeys } from "../queries/query-keys";

export function useUpdateConversationAssignee() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ conversationId, assignee }: { conversationId: number; assignee: string | null }) =>
      conversationRepository.updateAssignee(conversationId, assignee),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.crm.conversations() });
    },
  });
}
