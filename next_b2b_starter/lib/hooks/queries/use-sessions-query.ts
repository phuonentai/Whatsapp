// lib/hooks/queries/use-sessions-query.ts

import { useQuery, type UseQueryOptions } from "@tanstack/react-query";
import { queryKeys } from "./query-keys";
import { cognitiveRepository } from "@/lib/api/api/repositories/cognitive-repository";
import type { ChatSession, ChatMessage } from "@/lib/models/cognitive.model";

interface UseSessionsQueryOptions {
  enabled?: boolean;
}

export function useSessionsQuery(
  options: UseSessionsQueryOptions = {},
  queryOptions?: Omit<
    UseQueryOptions<ChatSession[], Error>,
    "queryKey" | "queryFn" | "enabled"
  >
) {
  const { enabled = true } = options;

  return useQuery({
    queryKey: queryKeys.cognitive.sessions(),
    queryFn: () => cognitiveRepository.listSessions(),
    enabled,
    staleTime: 1 * 60 * 1000, // 1 minute
    gcTime: 5 * 60 * 1000, // 5 minutes cache
    retry: 1,
    refetchOnWindowFocus: false,
    ...queryOptions,
  });
}

/**
 * Convenience hook to get sessions array directly
 */
export function useSessions(options: UseSessionsQueryOptions = {}) {
  const { data } = useSessionsQuery(options);
  return data ?? [];
}

interface UseSessionMessagesQueryOptions {
  sessionId: number;
  enabled?: boolean;
}

export function useSessionMessagesQuery(
  options: UseSessionMessagesQueryOptions,
  queryOptions?: Omit<
    UseQueryOptions<ChatMessage[], Error>,
    "queryKey" | "queryFn" | "enabled"
  >
) {
  const { sessionId, enabled = true } = options;

  return useQuery({
    queryKey: queryKeys.cognitive.messages(sessionId),
    queryFn: () => cognitiveRepository.getSessionMessages(sessionId),
    enabled: enabled && sessionId > 0,
    staleTime: 30 * 1000, // 30 seconds
    gcTime: 5 * 60 * 1000, // 5 minutes cache
    retry: 1,
    refetchOnWindowFocus: false,
    ...queryOptions,
  });
}

/**
 * Convenience hook to get messages array directly
 */
export function useSessionMessages(sessionId: number, enabled: boolean = true) {
  const { data } = useSessionMessagesQuery({ sessionId, enabled });
  return data ?? [];
}
