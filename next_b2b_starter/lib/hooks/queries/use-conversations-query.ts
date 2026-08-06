import { useQuery } from "@tanstack/react-query";
import { conversationRepository } from "@/lib/api/api/repositories/conversation-repository";
import { queryKeys } from "./query-keys";

export function useConversationsQuery(params?: { status?: string }) {
  return useQuery({
    queryKey: queryKeys.crm.conversations(params),
    queryFn: () => conversationRepository.listConversations(params),
    refetchInterval: 5000,
    staleTime: 5000,
  });
}
