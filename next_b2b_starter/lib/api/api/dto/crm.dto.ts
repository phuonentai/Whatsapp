export interface ContactDto {
  id: number;
  organization_id: number;
  phone_number: string;
  display_name: string;
  email?: string;
  company_id?: number;
  source: string;
  lead_status: string;
  job_title?: string;
  assigned_to?: number;
  tipo_documento?: string;
  numero_documento?: string;
  avatar_url?: string;
  is_blocked: boolean;
  last_message_at?: string;
  tags?: TagDto[];
  created_at: string;
  updated_at: string;
}

export interface CompanyDto {
  id: number;
  organization_id: number;
  name: string;
  nit?: string;
  tipo_empresa?: string;
  sector?: string;
  ciudad?: string;
  departamento?: string;
  website?: string;
  phone?: string;
  address?: string;
  notes?: string;
  owner_account_id?: number;
  total_contactos?: number;
  total_negocios?: number;
  tags?: TagDto[];
  created_at: string;
  updated_at: string;
}

export interface DealDto {
  id: number;
  organization_id: number;
  nombre: string;
  contact_id?: number;
  company_id?: number;
  pipeline_id: number;
  stage_id?: number;
  monto?: number;
  moneda: string;
  fecha_cierre_esperada?: string;
  estado: string;
  probabilidad?: number;
  notas?: string;
  assigned_to?: number;
  contact_name?: string;
  company_name?: string;
  tags?: TagDto[];
  created_at: string;
  updated_at: string;
}

export interface PipelineDto {
  id: number;
  organization_id: number;
  nombre: string;
  es_predeterminado: boolean;
  orden: number;
  etapas: PipelineStageDto[];
  created_at: string;
  updated_at: string;
}

export interface PipelineStageDto {
  id: number;
  pipeline_id: number;
  nombre: string;
  orden: number;
  color?: string;
  probabilidad?: number;
  created_at: string;
  updated_at: string;
}

export interface ActivityDto {
  id: number;
  organization_id: number;
  contact_id?: number;
  company_id?: number;
  deal_id?: number;
  conversation_id?: number;
  tipo: string;
  asunto?: string;
  contenido?: string;
  estado?: string;
  fecha_vencimiento?: string;
  realizada_por?: number;
  realizada_por_nombre?: string;
  realizada_en: string;
  metadata?: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface TagDto {
  id: number;
  organization_id: number;
  nombre: string;
  color?: string;
  created_at: string;
  updated_at: string;
}

export interface ModuleStateDto {
  enabled: boolean;
  features?: string[];
  config?: Record<string, unknown>;
}

export interface EntitlementDto {
  funcionalidades: Record<string, boolean>;
  cuotas: Record<string, number>;
  uso: Record<string, number>;
  solo_lectura: boolean;
  periodo_gracia: boolean;
  plan: string;
  modulos: Record<string, ModuleStateDto>;
}
