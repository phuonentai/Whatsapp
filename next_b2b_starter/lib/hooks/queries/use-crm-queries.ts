import { useQuery } from "@tanstack/react-query";
import { crmRepository } from "@/lib/api/api/repositories/crm-repository";
import { queryKeys } from "./query-keys";

export function useContactsQuery(params?: { source?: string; lead_status?: string; limit?: number; offset?: number }) {
  return useQuery({
    queryKey: queryKeys.crm.contacts(params),
    queryFn: () => crmRepository.listContacts(params),
  });
}

export function useContactQuery(id: number) {
  return useQuery({
    queryKey: queryKeys.crm.contact(id),
    queryFn: () => crmRepository.getContact(id),
    enabled: !!id,
  });
}

export function useCompaniesQuery(params?: { limit?: number; offset?: number }) {
  return useQuery({
    queryKey: queryKeys.crm.companies(params),
    queryFn: () => crmRepository.listCompanies(params),
  });
}

export function useCompanyQuery(id: number) {
  return useQuery({
    queryKey: queryKeys.crm.company(id),
    queryFn: () => crmRepository.getCompany(id),
    enabled: !!id,
  });
}

export function useDealsQuery(params?: { pipeline_id?: number; stage_id?: number; estado?: string }) {
  return useQuery({
    queryKey: queryKeys.crm.deals(params),
    queryFn: () => crmRepository.listDeals(params),
  });
}

export function useDealQuery(id: number) {
  return useQuery({
    queryKey: queryKeys.crm.deal(id),
    queryFn: () => crmRepository.getDeal(id),
    enabled: !!id,
  });
}

export function usePipelinesQuery() {
  return useQuery({
    queryKey: queryKeys.crm.pipelines(),
    queryFn: () => crmRepository.listPipelines(),
    staleTime: 10 * 60 * 1000,
  });
}

export function useActivitiesQuery(params?: { tipo?: string }) {
  return useQuery({
    queryKey: queryKeys.crm.activities(params),
    queryFn: () => crmRepository.listActivities(params),
  });
}

export function useContactActivitiesQuery(contactId: number) {
  return useQuery({
    queryKey: queryKeys.crm.contactActivities(contactId),
    queryFn: () => crmRepository.listActivitiesByContact(contactId),
    enabled: !!contactId,
  });
}

export function useDealActivitiesQuery(dealId: number) {
  return useQuery({
    queryKey: queryKeys.crm.dealActivities(dealId),
    queryFn: () => crmRepository.listActivitiesByDeal(dealId),
    enabled: !!dealId,
  });
}

export function useTagsQuery() {
  return useQuery({
    queryKey: queryKeys.crm.tags(),
    queryFn: () => crmRepository.listTags(),
    staleTime: 10 * 60 * 1000,
  });
}
