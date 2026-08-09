import { useQuery } from "@tanstack/react-query";
import { playbookRepository } from "@/lib/api/api/repositories/playbook-repository";
import { queryKeys } from "./query-keys";

export function usePlaybooksQuery(enabled = true) {
  return useQuery({
    queryKey: queryKeys.playbooks.catalog(),
    queryFn: () => playbookRepository.getCatalog(),
    enabled,
    staleTime: 5 * 60 * 1000,
  });
}
