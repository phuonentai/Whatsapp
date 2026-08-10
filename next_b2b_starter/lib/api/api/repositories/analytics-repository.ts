import { apiClient } from "../client/api-client";
import type {
  FunnelReportDto,
  InactiveContactDto,
  RevenuePointDto,
  TopCustomerDto,
} from "../dto/analytics.dto";

const BASE = "/analytics";

type Envelope<T> = { data?: T; success?: boolean };

async function unwrap<T>(request: Promise<Envelope<T> | T>): Promise<T> {
  const response = await request;
  if (response !== null && typeof response === "object" && "data" in response) {
    return (response as Envelope<T>).data as T;
  }
  return response as T;
}

export interface RevenueParams extends Record<string, string | number | undefined> {
  period?: "week" | "month";
  from?: string;
  to?: string;
}

export const analyticsRepository = {
  revenue: (params: RevenueParams = {}) =>
    unwrap(apiClient.get<Envelope<RevenuePointDto[]>>(`${BASE}/revenue`, { params })),
  topCustomers: (limit?: number) =>
    unwrap(apiClient.get<Envelope<TopCustomerDto[]>>(`${BASE}/top-customers`, {
      params: { limit },
    })),
  funnel: () => unwrap(apiClient.get<Envelope<FunnelReportDto>>(`${BASE}/funnel`)),
  inactiveContacts: (days?: number) =>
    unwrap(apiClient.get<Envelope<InactiveContactDto[]>>(`${BASE}/inactive-contacts`, {
      params: { days },
    })),
};
