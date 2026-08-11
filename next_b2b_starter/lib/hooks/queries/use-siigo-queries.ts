import { useQuery } from "@tanstack/react-query";
import { siigoRepository } from "@/lib/api/api/repositories/siigo-repository";
import { queryKeys } from "./query-keys";

export function useSiigoStatusQuery(options?: { enabled?: boolean; refetchInterval?: number | false }) {
  return useQuery({
    queryKey: queryKeys.siigo.status(),
    queryFn: () => siigoRepository.getStatus(),
    retry: false,
    staleTime: 30 * 1000,
    gcTime: 5 * 60 * 1000,
    ...options,
  });
}

export function useSiigoNumerationQuery(options?: { enabled?: boolean }) {
  return useQuery({
    queryKey: queryKeys.siigo.numeration(),
    queryFn: () => siigoRepository.getNumeration(),
    retry: false,
    staleTime: 30 * 1000,
    enabled: false,
    ...options,
  });
}

export function useImportPreviewQuery(options?: { enabled?: boolean }) {
  return useQuery({
    queryKey: queryKeys.siigo.importPreview(),
    queryFn: () => siigoRepository.importPreview(),
    retry: false,
    staleTime: 0,
    enabled: false,
    ...options,
  });
}

export function useAdminConnectionsQuery(options?: { enabled?: boolean; refetchInterval?: number | false }) {
  return useQuery({
    queryKey: queryKeys.siigo.adminConnections(),
    queryFn: () => siigoRepository.adminListConnections(),
    retry: false,
    staleTime: 30 * 1000,
    ...options,
  });
}
