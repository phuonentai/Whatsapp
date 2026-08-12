/**
 * Profile DTOs - Mirror backend ProfileResponse structure
 */

export interface ProfileOrganizationDto {
  organization_id: string;
  slug: string;
  name: string;
  status: string;
  // Display-only mirror of the Stytch org MFA policy (never used for
  // authorization decisions — Stytch is the sole enforcement point).
  mfa_policy?: string;
  mfa_methods?: string;
  allowed_mfa_methods?: string[];
}

export interface ProfileResponseDto {
  // Stytch member details
  member_id: string;
  email: string;
  name: string;
  roles: string[];
  permissions: string[];
  email_verified: boolean;
  status: string;

  // Organization details
  organization: ProfileOrganizationDto;

  // Internal account details
  account_id: number;
  created_at: string;
  updated_at: string;
}
