/**
 * Auth Event Audit Helper (server-only)
 *
 * Records authentication events into the organization audit stream (CRM
 * activity timeline, `tipo=sistema`) so they appear in the existing
 * `?view=audit` settings view.
 *
 * The helper posts to the existing Go endpoint `POST /api/crm/actividades`
 * using the member session JWT (read from cookies), matching how the CRM
 * repositories post activities. The audit payload (row metadata) contains
 * ONLY `{type, member_id, organization_id, detail}` — never session tokens,
 * JWTs, magic-link tokens, passwords, MFA codes, or raw error bodies.
 * `detail` is drawn from a bounded enum.
 *
 * Recording is best-effort and non-blocking: any failure (missing session,
 * network error, non-2xx response) is logged with `console.warn` and
 * swallowed — an audit failure NEVER alters the outcome of the auth action.
 *
 * This module MUST only be imported from server code (it reads cookies via
 * `next/headers` and issues server-side fetches).
 */

import { cookies } from "next/headers";

import { SESSION_JWT_COOKIE_NAME } from "@/lib/auth/constants";

/** Auth event types recorded into the audit stream (design D3). */
export const AUTH_AUDIT_EVENT_TYPES = [
  "magic_link_requested",
  "login_succeeded",
  "login_failed",
  "logout",
  "mfa_challenge_passed",
  "mfa_challenge_failed",
] as const;

export type AuthAuditEventType = (typeof AUTH_AUDIT_EVENT_TYPES)[number];

/**
 * Bounded, non-sensitive detail values allowed in audit rows. Never pass raw
 * Stytch error bodies — only mapped error codes from this set.
 */
export const AUTH_AUDIT_DETAILS = [
  "invalid_token",
  "expired_token",
  "mfa_required",
  "invalid_organization",
  "wrong_origin",
  "internal_error",
] as const;

export type AuthAuditDetail = (typeof AUTH_AUDIT_DETAILS)[number];

const EVENT_LABELS: Record<AuthAuditEventType, string> = {
  magic_link_requested: "Enlace mágico solicitado",
  login_succeeded: "Inicio de sesión exitoso",
  login_failed: "Inicio de sesión fallido",
  logout: "Cierre de sesión",
  mfa_challenge_passed: "Desafío MFA aprobado",
  mfa_challenge_failed: "Desafío MFA fallido",
};

export interface RecordAuthAuditInput {
  type: AuthAuditEventType;
  /** Stytch member id (attribution); never an email or token. */
  memberId?: string;
  /** Organization the event belongs to; the Go API still derives org from the session. */
  organizationId?: string;
  /** Bounded failure reason (e.g. `invalid_token`); omitted on success events. */
  detail?: AuthAuditDetail;
}

/** The audit payload: exactly `{type, member_id, organization_id, detail}`. */
interface AuditPayload {
  type: AuthAuditEventType;
  member_id?: string;
  organization_id?: string;
  detail?: AuthAuditDetail;
}

/** Body accepted by the Go activity-create endpoint (tipo=sistema row). */
interface CreateActivityBody {
  tipo: "sistema";
  asunto: string;
  contenido: string;
  metadata: AuditPayload;
}

function getGoBackendUrl(): string {
  return (
    process.env.NEXT_PUBLIC_GO_BACKEND_URL ??
    process.env.GO_BACKEND_URL ??
    "http://localhost:8080"
  );
}

async function readSessionJwt(): Promise<string | null> {
  const cookieStore = await cookies();
  return cookieStore.get(SESSION_JWT_COOKIE_NAME)?.value ?? null;
}

function isAuthAuditDetail(value: unknown): value is AuthAuditDetail {
  return (
    typeof value === "string" &&
    (AUTH_AUDIT_DETAILS as readonly string[]).includes(value)
  );
}

/**
 * Map a Stytch API error to a bounded audit detail. Raw error bodies are
 * NEVER recorded — unknown/unmapped errors collapse to `internal_error`.
 */
export function mapAuthErrorToDetail(error: unknown): AuthAuditDetail {
  let errorType: unknown;
  if (error && typeof error === "object" && "error_type" in error) {
    errorType = error.error_type;
  }
  return isAuthAuditDetail(errorType) ? errorType : "internal_error";
}

function buildActivityBody(input: RecordAuthAuditInput): CreateActivityBody {
  const payload: AuditPayload = { type: input.type };
  if (input.memberId) {
    payload.member_id = input.memberId;
  }
  if (input.organizationId) {
    payload.organization_id = input.organizationId;
  }
  if (input.detail && isAuthAuditDetail(input.detail)) {
    payload.detail = input.detail;
  }

  // Subject = human label; content = compact JSON of {type, detail} (design D1).
  const contenido: Record<string, string> = { type: input.type };
  if (payload.detail) {
    contenido.detail = payload.detail;
  }

  return {
    tipo: "sistema",
    asunto: EVENT_LABELS[input.type],
    contenido: JSON.stringify(contenido),
    metadata: payload,
  };
}

/**
 * Record an auth event into the organization audit stream (best-effort).
 *
 * Never throws: missing session, network failures, and non-2xx responses are
 * logged via `console.warn` and swallowed so the caller's auth outcome is
 * never altered by audit recording.
 */
export async function recordAuthAudit(input: RecordAuthAuditInput): Promise<void> {
  try {
    const sessionJwt = await readSessionJwt();
    if (!sessionJwt) {
      console.warn(
        `[Audit] No member session JWT available; skipping "${input.type}" audit record.`
      );
      return;
    }

    const body = buildActivityBody(input);

    const response = await fetch(`${getGoBackendUrl()}/api/crm/actividades`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${sessionJwt}`,
      },
      body: JSON.stringify(body),
    });

    if (!response.ok) {
      console.warn(
        `[Audit] Failed to record "${input.type}" audit event (HTTP ${response.status}).`
      );
    }
  } catch (error) {
    console.warn(
      `[Audit] Failed to record "${input.type}" audit event:`,
      error
    );
  }
}
