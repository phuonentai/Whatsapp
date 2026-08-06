import { useQuery } from "@tanstack/react-query";
import { crmRepository } from "@/lib/api/api/repositories/crm-repository";
import { queryKeys } from "./queries/query-keys";

export function useEntitlementQuery() {
  return useQuery({
    queryKey: queryKeys.crm.entitlement(),
    queryFn: () => crmRepository.getEntitlement(),
    staleTime: 5 * 60 * 1000,
  });
}

export function useFeature(key: string): boolean {
  const { data } = useEntitlementQuery();
  return data?.funcionalidades?.[key] ?? false;
}

export function useFeatures(): Record<string, boolean> {
  const { data } = useEntitlementQuery();
  return data?.funcionalidades ?? {};
}

export function useQuota(key: string): { used: number; max: number } {
  const { data } = useEntitlementQuery();
  return { used: data?.uso?.[key] ?? 0, max: data?.cuotas?.[key] ?? 0 };
}

export function useIsReadOnly(): boolean {
  const { data } = useEntitlementQuery();
  return data?.solo_lectura ?? false;
}

export function useIsGracePeriod(): boolean {
  const { data } = useEntitlementQuery();
  return data?.periodo_gracia ?? false;
}
