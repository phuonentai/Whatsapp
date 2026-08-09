import { useQuery } from "@tanstack/react-query";
import type { UseQueryOptions } from "@tanstack/react-query";
import { whatsappSignupRepository } from "@/lib/api/api/repositories/whatsapp-signup-repository";
import type { WhatsAppSignupStatus } from "@/lib/models/whatsapp-signup.model";
import { queryKeys } from "./query-keys";

export function useWhatsAppSignupMetaConfig(options?: { enabled?: boolean }) {
  return useQuery({
    queryKey: queryKeys.whatsappSignup.metaConfig(),
    queryFn: () => whatsappSignupRepository.getMetaConfig(),
    retry: false,
    staleTime: 10 * 60 * 1000,
    ...options,
  });
}

export function useWhatsAppSignupStatus(
  options?: Pick<
    UseQueryOptions<WhatsAppSignupStatus, Error>,
    "enabled" | "refetchInterval"
  >
) {
  return useQuery({
    queryKey: queryKeys.whatsappSignup.status(),
    queryFn: () => whatsappSignupRepository.getStatus(),
    retry: false,
    ...options,
  });
}
