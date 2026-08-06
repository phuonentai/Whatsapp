import { apiClient } from "../client/api-client";
import {
  BootstrapOrganizationRequestDto,
  BootstrapOrganizationResponseDto,
  SignupMagicLinkRequestDto,
  SignupMagicLinkResponseDto,
} from "../dto/auth.dto";
import {
  SignupOrganization,
  SignupOwner,
  SignupResult,
} from "@/lib/models/signup.model";
import { generateSecurePassword } from "@/lib/utils/password-generator";

class SignupRepository {
  // Legacy method with password (kept for backwards compatibility)
  async bootstrapOrganization(
    owner: SignupOwner & { password?: string },
    organization: SignupOrganization
  ): Promise<SignupResult> {
    const payload: BootstrapOrganizationRequestDto = {
      org_display_name: organization.displayName,
      owner_email: owner.email,
      owner_password: owner.password || "", // Fallback for legacy
      owner_name: owner.fullName,
    };

    const response = await apiClient.post<BootstrapOrganizationResponseDto>(
      "/auth/signup",
      payload,
      { skipAuth: true }
    );

    return this.toSignupResult(response);
  }

  // New magic link signup method
  async createOrganizationWithMagicLink(
    owner: SignupOwner,
    organization: SignupOrganization
  ): Promise<SignupResult> {
    // Generate a secure random password for backend compatibility
    // Users won't see or use this - they authenticate via magic link
    const securePassword = generateSecurePassword(24);

    const payload: SignupMagicLinkRequestDto = {
      org_display_name: organization.displayName,
      owner_email: owner.email,
      owner_name: owner.fullName,
      owner_password: securePassword,
      industry: organization.industry || "Technology", // Fallback to Technology if not set
    };

    // Debug logging (remove in production)
    console.log("Signup payload being sent:", {
      ...payload,
      owner_password: "[REDACTED]", // Don't log actual password
    });

    const response = await apiClient.post<SignupMagicLinkResponseDto>(
      "/auth/signup",
      payload,
      { skipAuth: true }
    );

    return this.toMagicLinkResult(response);
  }

  private toSignupResult(dto: BootstrapOrganizationResponseDto): SignupResult {
    return {
      orgId: dto.org_id,
      orgName: dto.org_name ?? dto.display_name,
      displayName: dto.display_name,
      ownerUserId: dto.owner_user_id,
      ownerEmail: dto.owner_email,
      ownerName: dto.owner_name,
      loginUrl: dto.login_url,
    };
  }

  private toMagicLinkResult(dto: SignupMagicLinkResponseDto): SignupResult {
    return {
      orgId: dto.org_id,
      orgName: dto.org_name ?? dto.display_name,
      displayName: dto.display_name,
      ownerUserId: "", // Not returned in magic link flow
      ownerEmail: dto.owner_email,
      ownerName: dto.owner_name,
      loginUrl: "/auth", // Redirect to auth page after magic link
    };
  }
}

export const signupRepository = new SignupRepository();
