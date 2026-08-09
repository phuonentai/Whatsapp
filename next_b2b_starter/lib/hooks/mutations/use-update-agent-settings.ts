import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { agentRepository } from "@/lib/api/api/repositories/agent-repository";
import { queryKeys } from "@/lib/hooks/queries/query-keys";
import type { AgentSettings } from "@/lib/models/agent.model";

export function useUpdateAgentSettings() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: Partial<AgentSettings>) =>
      agentRepository.updateSettings(input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.agent.settings() });
      toast.success("Configuración del asistente IA guardada");
    },
    onError: (error: Error) => {
      toast.error(error.message || "Error al guardar la configuración del asistente IA");
    },
  });
}
