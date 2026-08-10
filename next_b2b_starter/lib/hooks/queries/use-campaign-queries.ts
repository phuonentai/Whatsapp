import { useQuery } from "@tanstack/react-query";
import { campaignRepository } from "@/lib/api/api/repositories/campaign-repository";
import { queryKeys } from "./query-keys";

export function useSegmentsQuery() {
  return useQuery({
    queryKey: queryKeys.crm.segments(),
    queryFn: () => campaignRepository.listSegments(),
  });
}

export function useSegmentPreviewQuery(id: number) {
  return useQuery({
    queryKey: queryKeys.crm.segmentPreview(id),
    queryFn: () => campaignRepository.previewSegment(id),
    enabled: !!id,
  });
}

export function useCampaignsQuery() {
  return useQuery({
    queryKey: queryKeys.crm.campaigns(),
    queryFn: () => campaignRepository.listCampaigns(),
  });
}

export function useCampaignRecipientsQuery(campaignId: number, params?: { limit?: number; offset?: number }) {
  return useQuery({
    queryKey: queryKeys.crm.campaignRecipients(campaignId),
    queryFn: () => campaignRepository.listRecipients(campaignId, params),
    enabled: !!campaignId,
  });
}
