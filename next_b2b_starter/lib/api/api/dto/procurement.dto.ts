// Procurement module DTOs (add-supplier-inquiry-agent).
// Mirrors the Go domain JSON emitted by the /api/procurement handlers.

export interface SupplierDto {
  id: number;
  organization_id: number;
  contact_id: number;
  nit: string;
  display_name?: string;
  phone_number?: string;
  delivery_days?: number | null;
  min_order_amount?: number | null;
  notes?: string | null;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface ProductDto {
  id: number;
  organization_id: number;
  name: string;
  sku: string;
  unit: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export type RunStatus =
  | "draft"
  | "sending"
  | "awaiting_responses"
  | "completed"
  | "partially_answered"
  | "failed"
  | "escalated"
  | "cancelled";

export type RecipientStatus =
  | "pending"
  | "sent"
  | "answered"
  | "timed_out"
  | "failed";

export interface InquiryRunDto {
  id: number;
  organization_id: number;
  status: RunStatus;
  source: string;
  schedule_ref?: number | null;
  nota?: string | null;
  created_by_member_id?: string | null;
  sent_at?: string | null;
  completed_at?: string | null;
  created_at: string;
  updated_at: string;
}

export interface ResponseItemDto {
  product_name: string;
  sku?: string | null;
  disponible: boolean;
  precio_unitario?: number | null;
  moneda?: string;
  cantidad_disponible?: number | null;
  tiempo_entrega?: string | null;
  requiere_seguimiento: boolean;
}

export interface InquiryResponseDto {
  id: number;
  organization_id: number;
  recipient_id: number;
  raw_message_id: string;
  items: ResponseItemDto[];
  resumen: string;
  confidence?: number | null;
  requiere_humano: boolean;
  created_at: string;
}

export interface BoardRowDto {
  recipient_id: number;
  recipient_status: RecipientStatus;
  sent_at?: string | null;
  answered_at?: string | null;
  provider_message_id?: string | null;
  supplier_id: number;
  nit: string;
  delivery_days?: number | null;
  min_order_amount?: number | null;
  contact_id: number;
  display_name: string;
  phone_number: string;
  response?: InquiryResponseDto | null;
}

export interface BoardDto {
  run: InquiryRunDto;
  rows: BoardRowDto[];
  summary?: string | null;
}

export type OrderStatus = "placed" | "confirm_sent" | "send_blocked" | "confirm_failed";

export interface OrderDto {
  id: number;
  organization_id: number;
  run_id: number;
  supplier_id: number;
  contact_id: number;
  negocio_id?: number | null;
  status: OrderStatus;
  items: { product_id: number; quantity: number }[];
  notes?: string | null;
  confirm_message?: string | null;
  blocked_reason?: string | null;
  created_by_member_id?: string | null;
  created_at: string;
  updated_at: string;
}

export interface CreateSupplierInput {
  nit: string;
  phone: string;
  display_name: string;
  delivery_days?: number | null;
  min_order_amount?: number | null;
  notes?: string | null;
}

export interface UpdateSupplierInput {
  delivery_days?: number | null;
  min_order_amount?: number | null;
  notes?: string | null;
  is_active?: boolean;
}

export interface CreateProductInput {
  name: string;
  sku: string;
  unit: string;
}

export interface UpdateProductInput {
  name: string;
  sku: string;
  unit: string;
  is_active?: boolean;
}

export interface RunProductInput {
  product_id: number;
  quantity: number;
}

export interface CreateRunInput {
  supplier_ids: number[];
  products: RunProductInput[];
  nota?: string | null;
}

export interface PlaceOrderInput {
  supplier_id: number;
  items: { product_id: number; quantity: number }[];
  notes?: string | null;
  override?: boolean;
}

// ---- Inquiry scheduling (add-scheduled-inquiry-runs) ----
// NOTE: the Go domain structs serialize with Go field names (PascalCase) via
// gin c.JSON — the DTOs below mirror the verified wire shape.

export interface ScheduleDto {
  ID: number;
  OrganizationID: number;
  Name: string;
  RunTime: string;
  DaysOfWeek: number[];
  ProductIDs: number[];
  SupplierIDs: number[];
  Note?: string | null;
  IsActive: boolean;
  NextRunAt: string;
  LastRunAt?: string | null;
  LastRunOccurrenceAt?: string | null;
  CreatedAt: string;
  UpdatedAt: string;
}

export interface ScheduleStatusDto {
  Schedule: ScheduleDto;
  LastRunID?: number;
  LastRunStatus?: string;
  LastRunAt?: string | null;
  HasLastRun: boolean;
}

export interface ScheduledRunDto {
  ID: number;
  OrganizationID: number;
  Status: string;
  Source: string;
  ScheduleRef?: number | null;
  Nota?: string | null;
  CreatedAt: string;
}

export interface FollowUpSettingsDto {
  OrganizationID: number;
  Enabled: boolean;
  DeadlineHours: number;
  MaxNudges: number;
  MessageTemplate: string;
}

export interface ScheduleDetailDto {
  Schedule: ScheduleDto;
  FollowUp: FollowUpSettingsDto;
  RecentRuns: ScheduledRunDto[];
  OverdueRecipients: number;
}

export interface CreateScheduleInput {
  name: string;
  run_time: string;
  days_of_week: number[];
  product_ids: number[];
  supplier_ids: number[];
  note?: string | null;
}

export interface UpdateFollowUpSettingsInput {
  enabled: boolean;
  deadline_hours: number;
  max_nudges: number;
  message_template?: string | null;
}
