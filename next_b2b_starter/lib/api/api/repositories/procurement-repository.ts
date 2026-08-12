// Procurement API repository (/api/procurement/...). Unwraps the
// { data, success } envelope like the CRM repository.

import { apiClient } from "../client/api-client";
import type {
  BoardDto,
  CreateProductInput,
  CreateRunInput,
  CreateScheduleInput,
  CreateSupplierInput,
  FollowUpSettingsDto,
  InquiryRunDto,
  OrderDto,
  PlaceOrderInput,
  ProductDto,
  ScheduleDetailDto,
  ScheduleDto,
  ScheduleStatusDto,
  SupplierDto,
  UpdateFollowUpSettingsInput,
  UpdateProductInput,
  UpdateSupplierInput,
} from "../dto/procurement.dto";

const BASE = "/procurement";

type Envelope<T> = { data?: T; success?: boolean };

async function unwrap<T>(request: Promise<Envelope<T> | T>): Promise<T> {
  const response = await request;
  if (response !== null && typeof response === "object" && "data" in response) {
    return (response as Envelope<T>).data as T;
  }
  return response as T;
}

export const procurementRepository = {
  // ---- suppliers ----
  listSuppliers: () => unwrap(apiClient.get<Envelope<SupplierDto[]>>(`${BASE}/suppliers`)),
  createSupplier: (data: CreateSupplierInput) =>
    unwrap(apiClient.post<Envelope<SupplierDto>>(`${BASE}/suppliers`, data)),
  updateSupplier: (id: number, data: UpdateSupplierInput) =>
    unwrap(apiClient.put<Envelope<SupplierDto>>(`${BASE}/suppliers/${id}`, data)),

  // ---- products ----
  listProducts: () => unwrap(apiClient.get<Envelope<ProductDto[]>>(`${BASE}/products`)),
  createProduct: (data: CreateProductInput) =>
    unwrap(apiClient.post<Envelope<ProductDto>>(`${BASE}/products`, data)),
  updateProduct: (id: number, data: UpdateProductInput) =>
    unwrap(apiClient.put<Envelope<ProductDto>>(`${BASE}/products/${id}`, data)),

  // ---- runs ----
  listRuns: () => unwrap(apiClient.get<Envelope<InquiryRunDto[]>>(`${BASE}/runs`)),
  createRun: (data: CreateRunInput) =>
    unwrap(apiClient.post<Envelope<InquiryRunDto>>(`${BASE}/runs`, data)),
  sendRun: (id: number) => unwrap(apiClient.post<Envelope<InquiryRunDto>>(`${BASE}/runs/${id}/send`)),
  getRunBoard: (id: number) => unwrap(apiClient.get<Envelope<BoardDto>>(`${BASE}/runs/${id}`)),

  // ---- orders ----
  placeOrder: (runId: number, data: PlaceOrderInput) =>
    unwrap(apiClient.post<Envelope<OrderDto>>(`${BASE}/runs/${runId}/orders`, data)),
  listRunOrders: (runId: number) =>
    unwrap(apiClient.get<Envelope<OrderDto[]>>(`${BASE}/runs/${runId}/orders`)),

  // ---- schedules (add-scheduled-inquiry-runs) ----
  listSchedules: () => unwrap(apiClient.get<Envelope<ScheduleStatusDto[]>>(`${BASE}/schedules`)),
  getSchedule: (id: number) => unwrap(apiClient.get<Envelope<ScheduleDetailDto>>(`${BASE}/schedules/${id}`)),
  createSchedule: (data: CreateScheduleInput) =>
    unwrap(apiClient.post<Envelope<ScheduleDto>>(`${BASE}/schedules`, data)),
  updateSchedule: (id: number, data: CreateScheduleInput) =>
    unwrap(apiClient.put<Envelope<ScheduleDto>>(`${BASE}/schedules/${id}`, data)),
  pauseSchedule: (id: number) =>
    unwrap(apiClient.post<Envelope<ScheduleDto>>(`${BASE}/schedules/${id}/pause`)),
  resumeSchedule: (id: number) =>
    unwrap(apiClient.post<Envelope<ScheduleDto>>(`${BASE}/schedules/${id}/resume`)),
  deleteSchedule: (id: number) =>
    unwrap(apiClient.delete<Envelope<{ deleted: boolean }>>(`${BASE}/schedules/${id}`)),

  // ---- follow-up settings (add-scheduled-inquiry-runs) ----
  getFollowUpSettings: () =>
    unwrap(apiClient.get<Envelope<FollowUpSettingsDto>>(`${BASE}/followup-settings`)),
  updateFollowUpSettings: (data: UpdateFollowUpSettingsInput) =>
    unwrap(apiClient.put<Envelope<FollowUpSettingsDto>>(`${BASE}/followup-settings`, data)),
};
