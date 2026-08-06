import { useMutation, useQueryClient } from "@tanstack/react-query";
import { conversationRepository } from "@/lib/api/api/repositories/conversation-repository";
import { queryKeys } from "../queries/query-keys";

export function useSendMessage(conversationId: number) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (content: string) =>
      conversationRepository.sendMessage(conversationId, content),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.crm.messages(conversationId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.crm.conversations() });
    },
  });
}
