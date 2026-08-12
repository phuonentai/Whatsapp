// lib/api/api/repositories/organization-repository.ts

import { apiClient } from "../client/api-client";

export interface UpdateOrganizationPayload {
  name: string;
  status: string;
}

export interface UpdateMfaPolicyPayload {
  mfa_policy: "OPTIONAL" | "REQUIRED_FOR_ALL";
  mfa_methods: "ALL_ALLOWED" | "RESTRICTED";
  allowed_mfa_methods: string[];
}

export type JitPolicy = "DISABLED" | "DOMAIN_RESTRICTED";
export type SsoJitPolicy = "DISABLED" | "CONNECTION_RESTRICTED";
export type AllowedAuthMethod =
  | "magic_link"
  | "email_otp"
  | "sso"
  | "google_oauth"
  | "microsoft_oauth";

/** Display-only mirror of the org auth policy (GET /organizations/auth-policy). */
export interface AuthPolicyMirror {
  email_jit_provisioning: JitPolicy;
  email_allowed_domains: string[];
  auth_methods_restricted: boolean;
  allowed_auth_methods: AllowedAuthMethod[];
  sso_jit_provisioning: SsoJitPolicy;
  sso_jit_provisioning_allowed_connections: string[];
  sso_default_connection_id: string;
  sso_active_connection_ids: string[];
}

/** Full org auth policy write payload (PUT /organizations/auth-policy). */
export interface UpdateAuthPolicyPayload {
  email_jit_provisioning: JitPolicy;
  email_allowed_domains: string[];
  allowed_auth_methods: AllowedAuthMethod[];
  sso_jit_provisioning: SsoJitPolicy;
  sso_jit_provisioning_allowed_connections: string[];
  sso_default_connection_id: string;
}

class OrganizationRepository {
  /**
   * Update the current organization (workspace) metadata.
   * Requires org:manage permission. Mirrors PUT /api/organizations.
   */
  async updateOrganization(payload: UpdateOrganizationPayload): Promise<void> {
    await apiClient.put("/organizations", payload);
  }

  /**
   * Update the organization's MFA policy in Stytch via the Go backend
   * (PUT /api/organizations/mfa-policy, org:manage gated). Values mirror the
   * Stytch B2B org settings; outbound calls are circuit-breaker protected and
   * surface a 503 structured error when the auth provider is unavailable.
   */
  async updateMfaPolicy(payload: UpdateMfaPolicyPayload): Promise<void> {
    await apiClient.put("/organizations/mfa-policy", payload);
  }

  /**
   * Read the organization's auth policy mirror (GET /organizations/auth-policy,
   * org:manage gated). Display-only: the values are NEVER consulted for
   * authorization — Stytch enforces auth methods and JIT at authentication.
   * The backend answers 503 with a structured error when the auth provider is
   * unavailable (circuit breaker open / 5xx).
   */
  async getAuthPolicy(): Promise<AuthPolicyMirror> {
    return apiClient.get<AuthPolicyMirror>("/organizations/auth-policy");
  }

  /**
   * Persist the full org auth policy to Stytch via the Go backend
   * (PUT /organizations/auth-policy, org:manage gated). The backend always
   * writes `auth_methods: RESTRICTED` (enforced-list mode) together with the
   * org's preserved method set plus the requested additions; outbound calls
   * are circuit-breaker protected and surface a 503 structured error when the
   * auth provider is unavailable.
   */
  async updateAuthPolicy(payload: UpdateAuthPolicyPayload): Promise<void> {
    await apiClient.put("/organizations/auth-policy", payload);
  }
}

export const organizationRepository = new OrganizationRepository();
