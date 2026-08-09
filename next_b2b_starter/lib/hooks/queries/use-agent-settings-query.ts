import { useQuery } from "@tanstack/react-query";
import { agentRepository } from "@/lib/api/api/repositories/agent-repository";
import { queryKeys } from "./query-keys";

export function useAgentSettingsQuery(options?: { enabled?: boolean }) {
  return useQuery({
    queryKey: queryKeys.agent.settings(),
    queryFn: () => agentRepository.getSettings(),
    retry: false,
    staleTime: 60 * 1000,
    ...options,
  });
}
