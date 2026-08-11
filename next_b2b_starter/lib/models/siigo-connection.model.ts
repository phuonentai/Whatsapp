export type SiigoConnectionStatus =
  | "none"
  | "awaiting_setup"
  | "connected"
  | "numeracion_ok"
  | "sandbox_ok"
  | "live"
  | "paused"
  | "invoicing_disabled";

export interface SiigoConnection {
  organizationId: number;
  provider: string;
  status: SiigoConnectionStatus;
  nit?: string;
  siigoCompanyName?: string;
  lastError?: string;
  pausedAt?: string | null;
}

export interface SiigoConnectInput {
  client_id: string;
  client_secret: string;
  nit: string;
}

export interface SiigoNumeration {
  mode: "auto" | "manual";
  resolution_id?: string;
  prefijo?: string;
  next_number?: string;
  confirmed_at?: string | null;
}

export interface ImportCounts {
  total: number;
  nuevos: number;
  existentes: number;
  duplicados: number;
  sin_nit: number;
  sin_nombre: number;
  contactos: number;
  sin_contacto: number;
}

export interface TestInvoiceResult {
  invoice_id?: string;
  status: string;
  cufe?: string;
}

export interface AdminConnectionRow {
  organization_id: number;
  provider: string;
  status: SiigoConnectionStatus;
  nit?: string;
  siigo_company_name?: string;
  last_error?: string;
  numeration?: {
    mode: string;
    prefijo?: string;
    next_number?: string;
    confirmed_at?: string | null;
  };
  last_import_run?: {
    kind: string;
    counts: Record<string, number>;
    pulled_at: string;
  };
}
