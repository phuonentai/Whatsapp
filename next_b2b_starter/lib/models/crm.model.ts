export enum TipoDocumento {
  CC = "CC",
  NIT = "NIT",
  CE = "CE",
  TI = "TI",
  PP = "PP",
}

export enum TipoEmpresa {
  Microempresa = "microempresa",
  Pequena = "pequena",
  Mediana = "mediana",
  Grande = "grande",
}

export enum EstadoNegocio {
  Abierto = "abierto",
  Ganado = "ganado",
  Perdido = "perdido",
  Abandonado = "abandonado",
}

export enum ActividadTipo {
  Nota = "nota",
  Llamada = "llamada",
  Correo = "correo",
  Reunion = "reunion",
  Tarea = "tarea",
  WhatsApp = "whatsapp_message",
  Sistema = "sistema",
}

export type FeatureKey = "crm_contacts_manage" | "crm_companies" | "crm_deals" | "crm_activities" | "crm_tags";
