import { apiClient } from "../client/api-client";
import type { PlaybookDto } from "../dto/playbook.dto";

export interface ApplyPlaybookResult {
  key: string;
  applied_at: string;
}

interface Envelope<T> {
  data: T;
}

async function unwrap<T>(request: Promise<Envelope<T> | T>): Promise<T> {
  const response = await request;
  if (response !== null && typeof response === "object" && "data" in response) {
    return (response as Envelope<T>).data as T;
  }
  return response as T;
}

const BASE = "/playbooks";

export const playbookRepository = {
  getCatalog: () => unwrap(apiClient.get<Envelope<PlaybookDto[]>>(`${BASE}`)),
  apply: (key: string) => apiClient.post<ApplyPlaybookResult>(`${BASE}/${key}/apply`),
  reset: (key: string) => apiClient.post<{ key: string; reset: boolean }>(`${BASE}/${key}/reset`),
};
