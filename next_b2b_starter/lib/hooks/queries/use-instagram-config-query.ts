import { useQuery } from "@tanstack/react-query";
import { instagramConfigRepository } from "@/lib/api/api/repositories/instagram-config-repository";
import { queryKeys } from "./query-keys";

export function useInstagramConfigQuery(options?: { enabled?: boolean }) {
  return useQuery({
    queryKey: queryKeys.instagramConfig.detail(),
    queryFn: () => instagramConfigRepository.getConfig(),
    retry: false,
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
    ...options,
  });
}

export function useInstagramWebhookHealth(options?: { enabled?: boolean }) {
  return useQuery({
    queryKey: queryKeys.instagramConfig.health(),
    queryFn: () => instagramConfigRepository.getHealth(),
    enabled: options?.enabled ?? true,
    refetchInterval: 60 * 1000,
  });
}
