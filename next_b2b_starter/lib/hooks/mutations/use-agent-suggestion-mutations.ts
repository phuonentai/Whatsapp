import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { agentRepository } from "@/lib/api/api/repositories/agent-repository";
import { queryKeys } from "@/lib/hooks/queries/query-keys";

export function useApproveSuggestion() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ suggestionId, editedBody }: { suggestionId: number; editedBody?: string }) =>
      agentRepository.approveSuggestion(suggestionId, editedBody),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.agent.suggestions.pending() });
      toast.success("Respuesta enviada");
    },
    onError: (error: Error) => {
      toast.error(error.message || "La respuesta fue denegada por las reglas de gobernanza");
    },
  });
}

export function useRejectSuggestion() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (suggestionId: number) => agentRepository.rejectSuggestion(suggestionId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.agent.suggestions.pending() });
      toast.success("Sugerencia descartada");
    },
    onError: (error: Error) => {
      toast.error(error.message || "Error al descartar la sugerencia");
    },
  });
}
