"use client";

import { useQuery, type UseQueryOptions } from "@tanstack/react-query";

import { usageRepository } from "@/lib/api/api/repositories/usage-repository";
import type { AiUsageDto } from "@/lib/api/api/dto/ai-usage.dto";

const aiUsageQueryKey = ["ai-usage"] as const;

/**
 * Fetch the current-period AI usage (credits and tokens) for the org.
 */
export function useAiUsageQuery(
  enabled = true,
  options?: Omit<UseQueryOptions<AiUsageDto, Error>, "queryKey" | "queryFn">
) {
  return useQuery({
    queryKey: aiUsageQueryKey,
    queryFn: async (): Promise<AiUsageDto> => usageRepository.getAiUsage(),
    enabled,
    staleTime: 60 * 1000, // 1 minute
    gcTime: 5 * 60 * 1000, // 5 minutes
    refetchOnWindowFocus: false,
    retry: 1,
    ...options,
  });
}
