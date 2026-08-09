import { apiClient } from "../client/api-client";

export interface CatalogModuleDto {
  key: string;
  name: string;
  description?: string;
  features: string[];
  requires: string[];
}

export interface OrgModuleDto {
  key: string;
  name: string;
  description?: string;
  features: string[];
  config: Record<string, unknown>;
}

const BASE = "/modules";

export const moduleRepository = {
  getCatalog: () => apiClient.get<CatalogModuleDto[]>(`${BASE}`),
  getOrgModules: () => apiClient.get<OrgModuleDto[]>(`${BASE}/org`),
  saveConfig: (key: string, config: Record<string, unknown>) =>
    apiClient.put<{ module: string; config: Record<string, unknown> }>(`${BASE}/${key}/config`, config),
};
