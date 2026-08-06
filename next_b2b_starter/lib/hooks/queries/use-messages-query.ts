import { useQuery } from "@tanstack/react-query";
import { conversationRepository } from "@/lib/api/api/repositories/conversation-repository";
import { queryKeys } from "./query-keys";

export function useMessagesQuery(conversationId?: number) {
  return useQuery({
    queryKey: queryKeys.crm.messages(conversationId ?? 0),
    queryFn: () => conversationRepository.listMessages(conversationId!),
    enabled: !!conversationId,
    refetchInterval: 5000,
    staleTime: 5000,
  });
}
