import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { instagramConfigRepository } from "@/lib/api/api/repositories/instagram-config-repository";
import { queryKeys } from "@/lib/hooks/queries/query-keys";
import type { InstagramConfigInput } from "@/lib/models/instagram-config.model";

export function useUpsertInstagramConfig() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: InstagramConfigInput) =>
      instagramConfigRepository.upsertConfig(input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.instagramConfig.all });
      toast.success("Instagram configuration saved successfully");
    },
    onError: (error: Error) => {
      toast.error(error.message || "Failed to save Instagram configuration");
    },
  });
}
