export interface RevenuePointDto {
  periodo: string;
  monto_total: number;
}

export interface TopCustomerDto {
  nombre: string;
  monto_total: number;
}

export interface FunnelEntryDto {
  etapa: string;
  cantidad: number;
  monto_total: number;
}

export interface FunnelReportDto {
  etapas: FunnelEntryDto[];
  otras_pipelines?: FunnelEntryDto | null;
  ganado?: FunnelEntryDto | null;
  perdido?: FunnelEntryDto | null;
  abandonado?: FunnelEntryDto | null;
}

export interface InactiveContactDto {
  telefono: string;
  nombre: string;
  ultimo_mensaje_at?: string | null;
  clasificacion: "inactivo" | "sin_actividad";
}
