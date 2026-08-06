import { apiClient } from "../client/api-client";
import type { ContactDto, CompanyDto, DealDto, PipelineDto, ActivityDto, TagDto, EntitlementDto } from "../dto/crm.dto";

const BASE = "/crm";

export const crmRepository = {
  getEntitlement: () => apiClient.get<EntitlementDto>(`${BASE}/entitlement`),

  listContacts: (params?: { source?: string; lead_status?: string; limit?: number; offset?: number }) =>
    apiClient.get<ContactDto[]>(`${BASE}/contactos`, { params }),
  searchContacts: (q: string, params?: { limit?: number; offset?: number }) =>
    apiClient.get<ContactDto[]>(`${BASE}/contactos/search`, { params: { q, ...params } }),
  getContact: (id: number) => apiClient.get<ContactDto>(`${BASE}/contactos/${id}`),
  createContact: (data: Partial<ContactDto>) => apiClient.post<ContactDto>(`${BASE}/contactos`, data),
  updateContact: (id: number, data: Partial<ContactDto>) => apiClient.put<ContactDto>(`${BASE}/contactos/${id}`, data),
  deleteContact: (id: number) => apiClient.delete(`${BASE}/contactos/${id}`),

  listCompanies: (params?: { limit?: number; offset?: number }) =>
    apiClient.get<CompanyDto[]>(`${BASE}/empresas`, { params }),
  searchCompanies: (q: string) => apiClient.get<CompanyDto[]>(`${BASE}/empresas/search`, { params: { q } }),
  getCompany: (id: number) => apiClient.get<CompanyDto>(`${BASE}/empresas/${id}`),
  createCompany: (data: Partial<CompanyDto>) => apiClient.post<CompanyDto>(`${BASE}/empresas`, data),
  updateCompany: (id: number, data: Partial<CompanyDto>) => apiClient.put<CompanyDto>(`${BASE}/empresas/${id}`, data),
  deleteCompany: (id: number) => apiClient.delete(`${BASE}/empresas/${id}`),

  listDeals: (params?: { pipeline_id?: number; stage_id?: number; estado?: string; limit?: number; offset?: number }) =>
    apiClient.get<DealDto[]>(`${BASE}/negocios`, { params }),
  getDeal: (id: number) => apiClient.get<DealDto>(`${BASE}/negocios/${id}`),
  createDeal: (data: Partial<DealDto>) => apiClient.post<DealDto>(`${BASE}/negocios`, data),
  updateDeal: (id: number, data: Partial<DealDto>) => apiClient.put<DealDto>(`${BASE}/negocios/${id}`, data),
  moveDealStage: (id: number, data: { stage_id: number; old_stage_name: string; new_stage_name: string }) =>
    apiClient.put<DealDto>(`${BASE}/negocios/${id}/etapa`, data),
  deleteDeal: (id: number) => apiClient.delete(`${BASE}/negocios/${id}`),

  listPipelines: () => apiClient.get<PipelineDto[]>(`${BASE}/pipelines`),
  createPipeline: (data: Partial<PipelineDto>) => apiClient.post<PipelineDto>(`${BASE}/pipelines`, data),
  updatePipeline: (id: number, data: Partial<PipelineDto>) => apiClient.put<PipelineDto>(`${BASE}/pipelines/${id}`, data),
  createStage: (pipelineId: number, data: { nombre: string; orden: number; color?: string; probabilidad?: number }) =>
    apiClient.post(`${BASE}/pipelines/${pipelineId}/etapas`, data),
  updateStage: (pipelineId: number, stageId: number, data: Partial<{ nombre: string; orden: number; color: string; probabilidad: number }>) =>
    apiClient.put(`${BASE}/pipelines/${pipelineId}/etapas/${stageId}`, data),

  listActivities: (params?: { tipo?: string; limit?: number; offset?: number }) =>
    apiClient.get<ActivityDto[]>(`${BASE}/actividades`, { params }),
  createActivity: (data: Partial<ActivityDto>) => apiClient.post<ActivityDto>(`${BASE}/actividades`, data),
  listActivitiesByContact: (contactId: number) => apiClient.get<ActivityDto[]>(`${BASE}/actividades/contacto/${contactId}`),
  listActivitiesByDeal: (dealId: number) => apiClient.get<ActivityDto[]>(`${BASE}/actividades/negocio/${dealId}`),
  listActivitiesByCompany: (companyId: number) => apiClient.get<ActivityDto[]>(`${BASE}/actividades/empresa/${companyId}`),

  listTags: () => apiClient.get<TagDto[]>(`${BASE}/etiquetas`),
  createTag: (data: { nombre: string; color?: string }) => apiClient.post<TagDto>(`${BASE}/etiquetas`, data),
  deleteTag: (id: number) => apiClient.delete(`${BASE}/etiquetas/${id}`),
  tagEntity: (entityType: string, entityId: number, tagId: number) =>
    apiClient.post(`${BASE}/etiquetas/entity/${entityType}/${entityId}`, { tag_id: tagId }),
  untagEntity: (entityType: string, entityId: number, tagId: number) =>
    apiClient.delete(`${BASE}/etiquetas/entity/${entityType}/${entityId}/${tagId}`),
};
