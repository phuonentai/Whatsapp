import { apiClient } from "../client/api-client";
import type {
  AdminConnectionRow,
  ImportCounts,
  SiigoConnection,
  SiigoConnectInput,
  SiigoNumeration,
  TestInvoiceResult,
} from "@/lib/models/siigo-connection.model";

// The backend wraps all responses in a { data, success } envelope.
// Unwrap it so callers receive the payload directly (matches crm-repository).
type Envelope<T> = { data?: T; success?: boolean };

function unwrap<T>(response: unknown): T {
  return (response as Envelope<T>).data as T;
}

class SiigoRepository {
  async getStatus(): Promise<SiigoConnection> {
    const response = await apiClient.get<SiigoConnection>("/v1/org/siigo/status");
    return unwrap<SiigoConnection>(response);
  }

  async connect(input: SiigoConnectInput): Promise<SiigoConnection> {
    const response = await apiClient.post<SiigoConnection>("/v1/org/siigo/connect", input);
    return unwrap<SiigoConnection>(response);
  }

  async requestAssisted(): Promise<SiigoConnection> {
    const response = await apiClient.post<SiigoConnection>("/v1/org/siigo/request-assisted");
    return unwrap<SiigoConnection>(response);
  }

  async pause(): Promise<SiigoConnection> {
    const response = await apiClient.post<SiigoConnection>("/v1/org/siigo/pause");
    return unwrap<SiigoConnection>(response);
  }

  async resume(): Promise<SiigoConnection> {
    const response = await apiClient.post<SiigoConnection>("/v1/org/siigo/resume");
    return unwrap<SiigoConnection>(response);
  }

  async activate(): Promise<SiigoConnection> {
    const response = await apiClient.post<SiigoConnection>("/v1/org/siigo/activate");
    return unwrap<SiigoConnection>(response);
  }

  async getNumeration(): Promise<SiigoNumeration> {
    const response = await apiClient.get<SiigoNumeration>("/v1/org/siigo/numeration");
    return unwrap<SiigoNumeration>(response);
  }

  async confirmNumeration(): Promise<SiigoNumeration> {
    const response = await apiClient.post<SiigoNumeration>("/v1/org/siigo/confirm-numeration");
    return unwrap<SiigoNumeration>(response);
  }

  async importPreview(): Promise<ImportCounts> {
    const response = await apiClient.get<ImportCounts>("/v1/org/siigo/import/preview");
    return unwrap<ImportCounts>(response);
  }

  async importConfirm(): Promise<ImportCounts> {
    const response = await apiClient.post<ImportCounts>("/v1/org/siigo/import/confirm");
    return unwrap<ImportCounts>(response);
  }

  async sync(): Promise<ImportCounts> {
    const response = await apiClient.post<ImportCounts>("/v1/org/siigo/sync");
    return unwrap<ImportCounts>(response);
  }

  async testInvoice(): Promise<TestInvoiceResult> {
    const response = await apiClient.post<TestInvoiceResult>("/v1/org/siigo/test-invoice");
    return unwrap<TestInvoiceResult>(response);
  }

  async adminListConnections(): Promise<AdminConnectionRow[]> {
    const response = await apiClient.get<AdminConnectionRow[]>("/v1/admin/siigo/connections");
    return unwrap<AdminConnectionRow[]>(response);
  }

  async adminProvision(input: {
    organization_id: number;
    client_id: string;
    client_secret: string;
    nit: string;
  }): Promise<AdminConnectionRow> {
    const response = await apiClient.post<AdminConnectionRow>("/v1/admin/siigo/provision", input);
    return unwrap<AdminConnectionRow>(response);
  }
}

export const siigoRepository = new SiigoRepository();
