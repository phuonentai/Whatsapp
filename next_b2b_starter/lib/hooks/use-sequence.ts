"use client";

import { useCallback, useState } from "react";
import type { PlaybookGuionDto } from "@/lib/api/api/dto/playbook.dto";

interface SequenceState {
  conversationId: number;
  guion: PlaybookGuionDto;
  stepIndex: number;
}

/**
 * Drives a scripted sequence (guion with `pasos`) in the inbox composer.
 * start() fills the first step, advance() moves to the next after a send and
 * returns the next step's message (null when the sequence ends). The sequence
 * is keyed by conversation id: switching conversations makes it inactive
 * without resetting state eagerly.
 */
export function useSequence(conversationId: number) {
  const [state, setState] = useState<SequenceState | null>(null);

  const start = useCallback(
    (g: PlaybookGuionDto): string | null => {
      if (!g.pasos || g.pasos.length === 0) return null;
      setState({ conversationId, guion: g, stepIndex: 0 });
      return g.pasos[0].mensaje;
    },
    [conversationId]
  );

  const advance = useCallback((): string | null => {
    if (!state || state.conversationId !== conversationId) return null;
    const next = state.stepIndex + 1;
    if (next >= state.guion.pasos!.length) {
      setState(null);
      return null;
    }
    setState({ ...state, stepIndex: next });
    return state.guion.pasos![next].mensaje;
  }, [state, conversationId]);

  const reset = useCallback(() => {
    setState(null);
  }, []);

  const active = state !== null && state.conversationId === conversationId;

  return {
    active,
    stepIndex: active ? state!.stepIndex + 1 : null,
    totalSteps: active ? state!.guion.pasos!.length : 0,
    start,
    advance,
    reset,
  };
}
