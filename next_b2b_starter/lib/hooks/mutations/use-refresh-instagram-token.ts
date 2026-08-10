import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { instagramConfigRepository } from "@/lib/api/api/repositories/instagram-config-repository";
import { queryKeys } from "@/lib/hooks/queries/query-keys";

export function useRefreshInstagramToken() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: () => instagramConfigRepository.refreshToken(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.instagramConfig.all });
      toast.success("Instagram access token refreshed successfully");
    },
    onError: (error: Error) => {
      toast.error(error.message || "Failed to refresh Instagram access token");
    },
  });
}
