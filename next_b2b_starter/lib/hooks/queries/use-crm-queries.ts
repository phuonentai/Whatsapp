import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { crmRepository } from "@/lib/api/api/repositories/crm-repository";
import { queryKeys } from "./query-keys";

export interface ContactsQueryParams {
  source?: string;
  lead_status?: string;
  page?: number;
  pageSize?: number;
}

export interface CompaniesQueryParams {
  page?: number;
  pageSize?: number;
}

/**
 * Derive the API limit/offset pair from a page/pageSize request. Callers that
 * omit pagination keep fetching the full list (dashboard counts, dialog
 * pickers) — only paged views pass page/pageSize.
 */
function paginationParams(params?: { page?: number; pageSize?: number }) {
  const { page, pageSize } = params ?? {};
  if (page && pageSize) {
    return { limit: pageSize, offset: (page - 1) * pageSize };
  }
  return {};
}

export function useContactsQuery(params?: ContactsQueryParams) {
  const { page, pageSize, ...filters } = params ?? {};
  const requestParams = page && pageSize
    ? { ...filters, ...paginationParams(params) }
    : filters;
  const query = useQuery({
    // Page-scoped key: TanStack caches each offset window separately, so the
    // previous page's data survives while the next one loads.
    queryKey: queryKeys.crm.contacts(requestParams),
    queryFn: () => crmRepository.listContacts(requestParams),
    placeholderData: keepPreviousData,
  });
  return { ...query, data: query.data?.items, total: query.data?.total ?? 0 };
}

export function useContactQuery(id: number) {
  return useQuery({
    queryKey: queryKeys.crm.contact(id),
    queryFn: () => crmRepository.getContact(id),
    enabled: !!id,
  });
}

export function useCompaniesQuery(params?: CompaniesQueryParams) {
  const { page, pageSize, ...filters } = params ?? {};
  const requestParams = page && pageSize
    ? { ...filters, ...paginationParams(params) }
    : filters;
  const query = useQuery({
    queryKey: queryKeys.crm.companies(requestParams),
    queryFn: () => crmRepository.listCompanies(requestParams),
    placeholderData: keepPreviousData,
  });
  return { ...query, data: query.data?.items, total: query.data?.total ?? 0 };
}

export function useCompanyQuery(id: number) {
  return useQuery({
    queryKey: queryKeys.crm.company(id),
    queryFn: () => crmRepository.getCompany(id),
    enabled: !!id,
  });
}

export function useDealsQuery(params?: { pipeline_id?: number; stage_id?: number; estado?: string; contact_id?: number }) {
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

export function useActivitiesQuery(params?: { tipo?: string; limit?: number; offset?: number }) {
  const query = useQuery({
    queryKey: queryKeys.crm.activities(params),
    queryFn: () => crmRepository.listActivities(params),
  });
  return { ...query, data: query.data?.items, total: query.data?.total ?? 0 };
}

export function useContactActivitiesQuery(contactId: number) {
  const query = useQuery({
    queryKey: queryKeys.crm.contactActivities(contactId),
    queryFn: () => crmRepository.listActivitiesByContact(contactId),
    enabled: !!contactId,
  });
  return { ...query, data: query.data?.items, total: query.data?.total ?? 0 };
}

export function useDealActivitiesQuery(dealId: number) {
  const query = useQuery({
    queryKey: queryKeys.crm.dealActivities(dealId),
    queryFn: () => crmRepository.listActivitiesByDeal(dealId),
    enabled: !!dealId,
    // Deal activities are time-sensitive (stage changes create activities); the
    // global default disables refetch-on-mount, which would keep a stale empty
    // timeline when a deal was moved while viewing another route.
    refetchOnMount: true,
  });
  return { ...query, data: query.data?.items, total: query.data?.total ?? 0 };
}

export function useTagsQuery() {
  return useQuery({
    queryKey: queryKeys.crm.tags(),
    queryFn: () => crmRepository.listTags(),
    staleTime: 10 * 60 * 1000,
  });
}

export function useEntityTagsQuery(entityType: string, entityId: number) {
  return useQuery({
    queryKey: queryKeys.crm.entityTags(entityType, entityId),
    queryFn: () => crmRepository.listEntityTags(entityType, entityId),
    enabled: !!entityId,
  });
}
