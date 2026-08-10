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

// Backend wraps list responses in `{ data: ..., success: true }`; unwrap the
// envelope so the typed arrays below match the wire format.
type Envelope<T> = { data?: T; success?: boolean };

async function unwrap<T>(request: Promise<Envelope<T> | T>): Promise<T> {
  const response = await request;
  if (response !== null && typeof response === "object" && "data" in response) {
    return (response as Envelope<T>).data as T;
  }
  return response as T;
}

export const moduleRepository = {
  getCatalog: () => unwrap(apiClient.get<Envelope<CatalogModuleDto[]>>(`${BASE}`)),
  getOrgModules: () => unwrap(apiClient.get<Envelope<OrgModuleDto[]>>(`${BASE}/org`)),
  saveConfig: (key: string, config: Record<string, unknown>) =>
    apiClient.put<{ module: string; config: Record<string, unknown> }>(`${BASE}/${key}/config`, config),
};
