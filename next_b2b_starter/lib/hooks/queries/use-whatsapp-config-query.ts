import { useQuery } from "@tanstack/react-query";
import { whatsappConfigRepository } from "@/lib/api/api/repositories/whatsapp-config-repository";
import { queryKeys } from "./query-keys";

export function useWhatsAppConfigQuery(options?: { enabled?: boolean }) {
  return useQuery({
    queryKey: queryKeys.whatsappConfig.detail(),
    queryFn: () => whatsappConfigRepository.getConfig(),
    retry: false,
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
    ...options,
  });
}
