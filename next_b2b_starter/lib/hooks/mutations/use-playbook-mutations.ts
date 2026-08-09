"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { playbookRepository } from "@/lib/api/api/repositories/playbook-repository";
import { queryKeys } from "@/lib/hooks/queries/query-keys";

export function useApplyPlaybook() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (key: string) => playbookRepository.apply(key),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.playbooks.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.crm.pipelines() });
      queryClient.invalidateQueries({ queryKey: queryKeys.crm.tags() });
      queryClient.invalidateQueries({ queryKey: queryKeys.modules.all });
    },
  });
}

export function useResetPlaybook() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (key: string) => playbookRepository.reset(key),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.playbooks.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.crm.pipelines() });
      queryClient.invalidateQueries({ queryKey: queryKeys.crm.tags() });
      queryClient.invalidateQueries({ queryKey: queryKeys.modules.all });
    },
  });
}
