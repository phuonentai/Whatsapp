import { apiClient } from "../client/api-client";
import type {
  AudienceBuildResultDto,
  CampaignDto,
  CampaignRecipientDto,
  EvalResultDto,
  SegmentDto,
  SegmentFilter,
} from "../dto/campaign.dto";

const BASE = "/crm/campanas";

type Envelope<T> = { data?: T; success?: boolean };

async function unwrap<T>(request: Promise<Envelope<T> | T>): Promise<T> {
  const response = await request;
  if (response !== null && typeof response === "object" && "data" in response) {
    return (response as Envelope<T>).data as T;
  }
  return response as T;
}

export const campaignRepository = {
  // Segments
  listSegments: () => unwrap(apiClient.get<Envelope<SegmentDto[]>>(`${BASE}/segments`)),
  createSegment: (data: { nombre: string; filter_spec: SegmentFilter[] }) =>
    unwrap(apiClient.post<Envelope<SegmentDto>>(`${BASE}/segments`, data)),
  updateSegment: (id: number, data: { nombre: string; filter_spec: SegmentFilter[] }) =>
    unwrap(apiClient.put<Envelope<SegmentDto>>(`${BASE}/segments/${id}`, data)),
  deleteSegment: (id: number) => unwrap(apiClient.delete<Envelope<{ deleted: boolean }>>(`${BASE}/segments/${id}`)),
  previewSegment: (id: number) => unwrap(apiClient.get<Envelope<EvalResultDto>>(`${BASE}/segments/${id}/preview`)),
  previewSpec: (filter_spec: SegmentFilter[]) =>
    unwrap(apiClient.post<Envelope<EvalResultDto>>(`${BASE}/segments/preview`, { filter_spec })),

  // AI audience builder (nothing persisted until save)
  aiBuild: (descripcion: string) =>
    unwrap(apiClient.post<Envelope<AudienceBuildResultDto>>(`${BASE}/segments/ai-build`, { descripcion })),

  // Campaigns
  listCampaigns: () => unwrap(apiClient.get<Envelope<CampaignDto[]>>(`${BASE}`)),
  createCampaign: (data: { nombre: string; segment_id: number }) =>
    unwrap(apiClient.post<Envelope<CampaignDto>>(`${BASE}`, data)),
  launchCampaign: (id: number) => unwrap(apiClient.post<Envelope<CampaignDto>>(`${BASE}/${id}/launch`)),
  listRecipients: (id: number, params?: { limit?: number; offset?: number }) =>
    unwrap(apiClient.get<Envelope<CampaignRecipientDto[]>>(`${BASE}/${id}/recipients`, { params })),
};
