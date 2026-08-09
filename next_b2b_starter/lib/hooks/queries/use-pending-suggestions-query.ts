import { useQuery } from "@tanstack/react-query";
import { agentRepository } from "@/lib/api/api/repositories/agent-repository";
import { queryKeys } from "./query-keys";

export function usePendingSuggestionsQuery(options?: { enabled?: boolean; refetchInterval?: number }) {
  return useQuery({
    queryKey: queryKeys.agent.suggestions.pending(),
    queryFn: () => agentRepository.listSuggestions(),
    retry: false,
    staleTime: 15 * 1000,
    refetchInterval: options?.refetchInterval ?? 30_000,
    ...options,
  });
}
