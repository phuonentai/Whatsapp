export interface BootstrapOrganizationRequestDto {
  org_display_name: string;
  owner_email: string;
  owner_name: string;
}

// Magic Link Signup DTOs
export interface SignupMagicLinkRequestDto {
  org_display_name: string;
  owner_email: string;
  owner_name: string;
  industry?: string;
}

export interface SignupMagicLinkResponseDto {
  message: string;
  org_id: string;
  org_name?: string;
  display_name: string;
  owner_email: string;
  owner_name: string;
  magic_link_sent: boolean;
}

export interface BootstrapOrganizationResponseDto {
  org_id: string;
  org_name?: string;
  display_name: string;
  owner_user_id: string;
  owner_email: string;
  owner_name: string;
  login_url: string;
}

export interface LoginRequestDto {
  email: string;
  password: string;
}

export interface LoginResponseDto {
  access_token: string;
  token_type?: string;
  expires_in?: number;
  refresh_token?: string;
}
