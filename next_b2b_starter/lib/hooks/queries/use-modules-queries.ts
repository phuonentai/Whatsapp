import { useQuery } from "@tanstack/react-query";
import { moduleRepository } from "@/lib/api/api/repositories/module-repository";
import { ticketRepository } from "@/lib/api/api/repositories/ticket-repository";
import { queryKeys } from "./query-keys";

export function useModulesCatalogQuery() {
  return useQuery({
    queryKey: queryKeys.modules.catalog(),
    queryFn: () => moduleRepository.getCatalog(),
    staleTime: 10 * 60 * 1000,
  });
}

export function useOrgModulesQuery() {
  return useQuery({
    queryKey: queryKeys.modules.org(),
    queryFn: () => moduleRepository.getOrgModules(),
    staleTime: 5 * 60 * 1000,
  });
}

export function useTicketsQuery(params?: { status?: string; assignee?: string }) {
  return useQuery({
    queryKey: queryKeys.crm.tickets(params),
    queryFn: () => ticketRepository.list(params),
  });
}

export function useTicketQuery(id: number) {
  return useQuery({
    queryKey: queryKeys.crm.ticket(id),
    queryFn: () => ticketRepository.get(id),
    enabled: !!id,
  });
}
