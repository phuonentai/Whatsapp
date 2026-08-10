import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { instagramConfigRepository } from "@/lib/api/api/repositories/instagram-config-repository";
import { queryKeys } from "@/lib/hooks/queries/query-keys";

export function useToggleInstagramConfig() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: () => instagramConfigRepository.toggleConfig(),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.instagramConfig.all });
      toast.success(
        data.isActive
          ? "Instagram messaging is now active"
          : "Instagram messaging is now paused"
      );
    },
    onError: (error: Error) => {
      toast.error(error.message || "Failed to toggle Instagram configuration");
    },
  });
}
