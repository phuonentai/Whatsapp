import { useQuery } from "@tanstack/react-query";
import { agentRepository } from "@/lib/api/api/repositories/agent-repository";
import { queryKeys } from "./query-keys";

/**
 * Fetches AI-derived context for a conversation (summary, intent, key facts).
 * Disabled until a conversation is selected; the panel renders its "learning"
 * state while this is loading or after an unavailable/structural response.
 */
export function useConversationContextQuery(conversationId?: number) {
  return useQuery({
    queryKey: queryKeys.agent.context(conversationId ?? -1),
    queryFn: () => agentRepository.getConversationContext(conversationId!),
    enabled: conversationId != null,
    staleTime: 60_000,
  });
}
