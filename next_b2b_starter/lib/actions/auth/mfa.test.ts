import { afterEach, beforeEach, describe, expect, it, vi, type Mock } from "vitest";

const mocks = vi.hoisted(() => ({
  cookies: vi.fn(),
  headers: vi.fn(),
  getMemberSession: vi.fn(),
  getStytchB2BClient: vi.fn(),
  recordAuthAudit: vi.fn(),
  cookieStoreSet: vi.fn(),
  totpsCreate: vi.fn(),
  totpsAuthenticate: vi.fn(),
  recoveryCodesRecover: vi.fn(),
}));

vi.mock("next/headers", () => ({
  cookies: () => ({ set: mocks.cookieStoreSet, get: vi.fn() }),
  headers: () => mocks.headers(),
}));

vi.mock("@/lib/auth/stytch/server", () => ({
  getMemberSession: mocks.getMemberSession,
  getStytchB2BClient: mocks.getStytchB2BClient,
}));

vi.mock("@/lib/auth/audit", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/auth/audit")>();
  return {
    ...actual,
    recordAuthAudit: mocks.recordAuthAudit,
  };
});

import {
  authenticateTotp,
  createTotp,
  verifyTotpEnrollment,
} from "@/lib/actions/auth/mfa";

const MEMBER_LIMIT_ENV = "MFA_RATE_LIMIT_PER_MEMBER_PER_HOUR";
const IP_LIMIT_ENV = "MFA_RATE_LIMIT_PER_IP_PER_HOUR";

const SESSION = {
  request_id: "req-1",
  status_code: 200,
  member_session: {} as never,
  session_token: "member-session-token",
  session_jwt: "member-session-jwt",
  member: {
    member_id: "member-123",
    email_address: "member@example.com",
    name: "Member",
    totp_registration_id: "",
  },
  organization: {
    organization_id: "org-456",
    organization_name: "Acme",
  },
};

function withEnv(envName: string, value: string, fn: () => void) {
  const previous = process.env[envName];
  process.env[envName] = value;
  try {
    fn();
  } finally {
    if (previous === undefined) {
      delete process.env[envName];
    } else {
      process.env[envName] = previous;
    }
  }
}

function mockHeaderStore(ip = "203.0.113.9") {
  mocks.headers.mockReturnValue({
    get: (name: string) => (name === "x-real-ip" ? ip : null),
  });
}

function mockClient() {
  mocks.getStytchB2BClient.mockReturnValue({
    totps: {
      create: mocks.totpsCreate,
      authenticate: mocks.totpsAuthenticate,
    },
    recoveryCodes: {
      recover: mocks.recoveryCodesRecover,
    },
  });
}

function stytchError(message: string) {
  return Object.assign(new Error(message), { error_message: message });
}

beforeEach(() => {
  mocks.cookieStoreSet.mockReset();
  mocks.recordAuthAudit.mockReset();
  mocks.totpsCreate.mockReset();
  mocks.totpsAuthenticate.mockReset();
  mocks.recoveryCodesRecover.mockReset();
  mocks.getMemberSession.mockReset();
  mocks.getMemberSession.mockResolvedValue(SESSION);
  mockHeaderStore();
  mockClient();
});

afterEach(() => {
  vi.restoreAllMocks();
});

describe("authenticateTotp (TOTP path)", () => {
  it("sets session cookies on success and records mfa_challenge_passed + login_succeeded", async () => {
    mocks.totpsAuthenticate.mockResolvedValue({
      session_token: "totp-session-token",
      session_jwt: "totp-session-jwt",
      member: SESSION.member,
      organization: SESSION.organization,
    });

    const result = await authenticateTotp({
      intermediateSessionToken: "intermediate-1",
      code: "123456",
      memberId: "member-123",
      organizationId: "org-456",
    });

    expect(result.success).toBe(true);
    if (!result.success) return;

    expect(mocks.cookieStoreSet).toHaveBeenCalledWith(
      "stytch_session",
      "totp-session-token",
      expect.objectContaining({ httpOnly: true, maxAge: 480 * 60 })
    );
    expect(mocks.cookieStoreSet).toHaveBeenCalledWith(
      "stytch_session_jwt",
      "totp-session-jwt",
      expect.any(Object)
    );
    expect(mocks.totpsAuthenticate).toHaveBeenCalledWith(
      expect.objectContaining({
        intermediate_session_token: "intermediate-1",
        code: "123456",
        member_id: "member-123",
        organization_id: "org-456",
      })
    );
    expect(mocks.recoveryCodesRecover).not.toHaveBeenCalled();
    expect(mocks.recordAuthAudit).toHaveBeenCalledWith({
      type: "mfa_challenge_passed",
      memberId: "member-123",
      organizationId: "org-456",
    });
    expect(mocks.recordAuthAudit).toHaveBeenCalledWith({
      type: "login_succeeded",
      memberId: "member-123",
      organizationId: "org-456",
    });
  });

  it("sets NO cookies on a wrong code and records a bounded mfa_challenge_failed", async () => {
    mocks.totpsAuthenticate.mockRejectedValue(
      stytchError("Invalid TOTP code")
    );

    const result = await authenticateTotp({
      intermediateSessionToken: "intermediate-1",
      code: "000000",
      memberId: "member-123",
      organizationId: "org-456",
    });

    expect(result.success).toBe(false);
    expect(mocks.cookieStoreSet).not.toHaveBeenCalled();
    expect(mocks.recordAuthAudit).toHaveBeenCalledWith(
      expect.objectContaining({ type: "mfa_challenge_failed" })
    );
    // detail stays within the bounded enum (unmapped Stytch error -> internal_error)
    const call = mocks.recordAuthAudit.mock.calls[0]?.[0] as { detail?: string };
    expect(["invalid_token", "internal_error"]).toContain(call?.detail);
  });
});

describe("authenticateTotp (recovery-code path)", () => {
  it("sets cookies DIRECTLY from the recover response and issues NO chained TOTP authenticate call", async () => {
    mocks.recoveryCodesRecover.mockResolvedValue({
      session_token: "recover-session-token",
      session_jwt: "recover-session-jwt",
      recovery_codes_remaining: 9,
      member: SESSION.member,
      organization: SESSION.organization,
    });

    const result = await authenticateTotp({
      intermediateSessionToken: "intermediate-1",
      recoveryCode: "abcd-efgh",
      memberId: "member-123",
      organizationId: "org-456",
    });

    expect(result.success).toBe(true);
    expect(mocks.recoveryCodesRecover).toHaveBeenCalledWith(
      expect.objectContaining({
        intermediate_session_token: "intermediate-1",
        recovery_code: "abcd-efgh",
      })
    );
    // NO follow-up TOTP authenticate: the recover call consumed the single-use
    // intermediate token and returned the session directly.
    expect(mocks.totpsAuthenticate).not.toHaveBeenCalled();
    expect(mocks.cookieStoreSet).toHaveBeenCalledWith(
      "stytch_session",
      "recover-session-token",
      expect.any(Object)
    );
    expect(mocks.cookieStoreSet).toHaveBeenCalledWith(
      "stytch_session_jwt",
      "recover-session-jwt",
      expect.any(Object)
    );
    expect(mocks.recordAuthAudit).toHaveBeenCalledWith({
      type: "mfa_challenge_passed",
      memberId: "member-123",
      organizationId: "org-456",
    });
  });

  it("rejects with a generic error when rate-limited and never calls Stytch", async () => {
    const prevMemberLimit = process.env[MEMBER_LIMIT_ENV];
    const prevIpLimit = process.env[IP_LIMIT_ENV];
    process.env[MEMBER_LIMIT_ENV] = "1";
    process.env[IP_LIMIT_ENV] = "30";
    mocks.recoveryCodesRecover.mockResolvedValue({
      session_token: "recover-session-token",
      session_jwt: "recover-session-jwt",
      recovery_codes_remaining: 9,
      member: SESSION.member,
      organization: SESSION.organization,
    });
    try {
      // First attempt consumes the only per-member slot.
      const first = await authenticateTotp({
        intermediateSessionToken: "intermediate-1",
        recoveryCode: "abcd-efgh-1",
        memberId: "member-limited",
        organizationId: "org-456",
      });
      expect(first.success).toBe(true);

      // Second attempt is throttled.
      const second = await authenticateTotp({
        intermediateSessionToken: "intermediate-1",
        recoveryCode: "abcd-efgh-2",
        memberId: "member-limited",
        organizationId: "org-456",
      });
      expect(second.success).toBe(false);
    } finally {
      if (prevMemberLimit === undefined) {
        delete process.env[MEMBER_LIMIT_ENV];
      } else {
        process.env[MEMBER_LIMIT_ENV] = prevMemberLimit;
      }
      if (prevIpLimit === undefined) {
        delete process.env[IP_LIMIT_ENV];
      } else {
        process.env[IP_LIMIT_ENV] = prevIpLimit;
      }
    }

    // Only the first (allowed) attempt reached Stytch and set cookies (2
    // calls: session token + session JWT); the throttled attempt set none.
    expect(mocks.recoveryCodesRecover).toHaveBeenCalledTimes(1);
    expect(mocks.cookieStoreSet).toHaveBeenCalledTimes(2);
    expect(mocks.recordAuthAudit).toHaveBeenCalledWith({
      type: "mfa_challenge_failed",
      memberId: "member-limited",
      organizationId: "org-456",
      detail: "internal_error",
    });
  });

  it("rejects generic errors without revealing code validity", async () => {
    mocks.recoveryCodesRecover.mockRejectedValue(
      stytchError("recovery code invalid")
    );

    const result = await authenticateTotp({
      intermediateSessionToken: "intermediate-1",
      recoveryCode: "bad-code",
      memberId: "member-123",
      organizationId: "org-456",
    });

    expect(result.success).toBe(false);
    expect(mocks.cookieStoreSet).not.toHaveBeenCalled();
  });
});

describe("createTotp (duplicate guard)", () => {
  it("performs NO create call when the member already has a totp_registration_id", async () => {
    mocks.getMemberSession.mockResolvedValue({
      ...SESSION,
      member: {
        ...SESSION.member,
        totp_registration_id: "totp-existing-1",
      },
    });

    const result = await createTotp();

    expect(result.success).toBe(true);
    if (!result.success) return;
    expect(result.data).toEqual({
      status: "existing",
      totpRegistrationId: "totp-existing-1",
    });
    expect(mocks.totpsCreate).not.toHaveBeenCalled();
  });

  it("creates a TOTP instance with the member session JWT when no registration exists", async () => {
    mocks.totpsCreate.mockResolvedValue({
      totp_registration_id: "totp-new-1",
      qr_code: "data:image/png;base64,QR",
      secret: "MANUAL-SECRET",
      recovery_codes: ["rc-1", "rc-2"],
    });

    const result = await createTotp();

    expect(result.success).toBe(true);
    if (!result.success) return;
    expect(result.data.status).toBe("created");
    expect(result.data.qrCode).toBe("data:image/png;base64,QR");
    expect(result.data.secret).toBe("MANUAL-SECRET");
    expect(result.data.recoveryCodes).toEqual(["rc-1", "rc-2"]);
    expect(mocks.totpsCreate).toHaveBeenCalledWith({
      organization_id: "org-456",
      member_id: "member-123",
      session_jwt: "member-session-jwt",
    });
  });

  it("rejects when the member is not signed in", async () => {
    mocks.getMemberSession.mockResolvedValue(null);

    const result = await createTotp();

    expect(result.success).toBe(false);
    expect(mocks.totpsCreate).not.toHaveBeenCalled();
  });
});

describe("verifyTotpEnrollment", () => {
  it("authenticates the code with set_mfa_enrollment=enroll on the member session", async () => {
    mocks.totpsAuthenticate.mockResolvedValue({
      member: { ...SESSION.member, totp_registration_id: "totp-new-1" },
    });

    const result = await verifyTotpEnrollment({ code: "654321" });

    expect(result.success).toBe(true);
    expect(mocks.totpsAuthenticate).toHaveBeenCalledWith(
      expect.objectContaining({
        code: "654321",
        session_jwt: "member-session-jwt",
        set_mfa_enrollment: "enroll",
      })
    );
    expect(mocks.cookieStoreSet).not.toHaveBeenCalled();
  });

  it("rejects an empty code", async () => {
    const result = await verifyTotpEnrollment({ code: "   " });

    expect(result.success).toBe(false);
    expect(mocks.totpsAuthenticate).not.toHaveBeenCalled();
  });
});
