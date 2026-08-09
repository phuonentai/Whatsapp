import { apiClient } from "../client/api-client";
import type { PlaybookDto } from "../dto/playbook.dto";

export interface ApplyPlaybookResult {
  key: string;
  applied_at: string;
}

const BASE = "/playbooks";

export const playbookRepository = {
  getCatalog: () => apiClient.get<PlaybookDto[]>(`${BASE}`),
  apply: (key: string) => apiClient.post<ApplyPlaybookResult>(`${BASE}/${key}/apply`),
  reset: (key: string) => apiClient.post<{ key: string; reset: boolean }>(`${BASE}/${key}/reset`),
};
