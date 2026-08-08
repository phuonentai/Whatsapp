import { z } from "zod";

export const LEAD_STATUSES = ["nuevo", "contactado", "calificado", "descalificado", "cliente"] as const;

export const contactSchema = z.object({
  phone: z
    .string()
    .min(1, "El teléfono es requerido")
    .regex(/^\+573\d{9}$/, "Teléfono inválido"),
  display_name: z.string().optional(),
  email: z.string().email("Correo inválido").optional().or(z.literal("")),
  lead_status: z.enum(LEAD_STATUSES),
});

export type ContactFormValues = z.infer<typeof contactSchema>;

export const companySchema = z.object({
  name: z.string().min(1, "El nombre es requerido"),
  nit: z.string().optional(),
  sector: z.string().optional(),
  ciudad: z.string().optional(),
});

export type CompanyFormValues = z.infer<typeof companySchema>;

export const dealSchema = z.object({
  nombre: z.string().min(1, "El nombre es requerido"),
  monto: z.string().optional(),
  moneda: z.string().optional(),
  company_id: z.string().optional(),
  contact_id: z.string().optional(),
  stage_id: z.string().optional(),
});

export type DealFormValues = z.infer<typeof dealSchema>;
