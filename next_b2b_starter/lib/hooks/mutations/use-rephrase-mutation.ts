// lib/hooks/mutations/use-rephrase-mutation.ts

import { useMutation } from "@tanstack/react-query";
import { agentRepository } from "@/lib/api/api/repositories/agent-repository";
import type { RephraseMode } from "@/lib/models/agent.model";

/**
 * Transforms the composer draft through the writing-assist endpoint.
 * The result replaces the draft for review — nothing is ever sent here.
 * Toast/draft handling lives in the composer; this hook stays thin.
 */
export function useRephraseMutation() {
  return useMutation({
    mutationFn: ({ text, mode }: { text: string; mode: RephraseMode }) =>
      agentRepository.rephrase(text, mode),
  });
}

/**
 * Detects the 402 credits-exhausted response surfaced by the apiClient
 * ("API Error 402: <message>").
 */
export function isAiCreditsExhausted(error: unknown): boolean {
  return error instanceof Error && /API Error 402/.test(error.message);
}
