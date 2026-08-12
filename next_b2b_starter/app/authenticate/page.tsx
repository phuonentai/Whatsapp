"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { AlertCircle, CheckCircle2, Loader2, ShieldCheck } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { consumeMagicLink } from "@/lib/actions/auth/consume-magic-link";
import { authenticateTotp } from "@/lib/actions/auth/mfa";
import { ui } from "@/lib/copy/ui";

const SESSION_DURATION_MINUTES = Number(
  process.env.NEXT_PUBLIC_STYTCH_SESSION_DURATION_MINUTES ?? "480"
) || 480;

const DEFAULT_DESTINATION = "/dashboard";

type Phase = "verifying" | "mfa" | "success" | "error";

type StatusState = {
  phase: Phase;
  headline: string;
  message: string;
};

const INITIAL_STATUS: StatusState = {
  phase: "verifying",
  headline: ui.auth.authenticateVerifying,
  message: ui.auth.authenticateVerifyingBody,
};

interface MfaContext {
  intermediateSessionToken: string;
  memberId?: string;
  organizationId?: string;
}

function extractErrorMessage(error: unknown): string {
  if (error && typeof error === "object") {
    const typed = error as any;

    if (typed.error_message) {
      return typed.error_message;
    }

    if (typed.message) {
      return typed.message;
    }
  }

  return ui.auth.authenticateVerifyErrorBody;
}

export default function AuthenticateRedirectPage() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [status, setStatus] = useState<StatusState>(INITIAL_STATUS);
  const [mfaContext, setMfaContext] = useState<MfaContext | null>(null);

  // MFA challenge form state.
  const [mfaCode, setMfaCode] = useState("");
  const [useRecovery, setUseRecovery] = useState(false);
  const [recoveryCode, setRecoveryCode] = useState("");
  const [isSubmittingMfa, setIsSubmittingMfa] = useState(false);
  const [mfaError, setMfaError] = useState<string | null>(null);

  const hasAttemptedAuthRef = useRef(false);

  const magicLinkToken = searchParams.get("stytch_token") || searchParams.get("token");
  const returnTo = searchParams.get("returnTo")?.trim() || DEFAULT_DESTINATION;

  const redirectToDestination = useCallback(() => {
    router.push(returnTo);
    router.refresh();
  }, [returnTo, router]);

  const exchangeMagicLink = useCallback(async () => {
    if (!magicLinkToken) {
      setStatus({
        phase: "error",
        headline: ui.auth.authenticateMissing,
        message: ui.auth.authenticateMissingBody,
      });
      return;
    }

    hasAttemptedAuthRef.current = true;
    setStatus(INITIAL_STATUS);

    try {
      const result = await consumeMagicLink(
        magicLinkToken,
        SESSION_DURATION_MINUTES
      );

      if (!result.success) {
        throw new Error(result.error || ui.auth.authenticateFailed);
      }

      if (!result.data.memberAuthenticated) {
        // MFA challenge required: primary auth passed, the org/member
        // enrollment requires a second factor. Render the TOTP (or recovery
        // code) step instead of dead-ending the login (design D1).
        if (result.data.mfaRequired) {
          setMfaContext({
            intermediateSessionToken: result.data.intermediateSessionToken ?? "",
            memberId: result.data.member?.member_id,
            organizationId: result.data.organization?.organization_id,
          });
          setStatus({
            phase: "mfa",
            headline: ui.auth.mfaChallengeTitle,
            message: ui.auth.mfaChallengeBody,
          });
          return;
        }

        setStatus({
          phase: "error",
          headline: ui.auth.authenticateAdditional,
          message: ui.auth.authenticateAdditionalBody,
        });
        return;
      }

      setStatus({
        phase: "success",
        headline: ui.auth.authenticateSuccess,
        message: ui.auth.authenticateSuccessBody,
      });

      redirectToDestination();
    } catch (error) {
      setStatus({
        phase: "error",
        headline: ui.auth.authenticateVerifyError,
        message: extractErrorMessage(error),
      });
    }
  }, [magicLinkToken, redirectToDestination]);

  useEffect(() => {
    if (hasAttemptedAuthRef.current) return;
    void exchangeMagicLink();
  }, [exchangeMagicLink]);

  /** Submit the MFA challenge (TOTP code or recovery code). */
  const submitMfaChallenge = useCallback(async () => {
    if (!mfaContext) return;
    setIsSubmittingMfa(true);
    setMfaError(null);

    try {
      const result = await authenticateTotp({
        intermediateSessionToken: mfaContext.intermediateSessionToken,
        code: useRecovery ? undefined : mfaCode.trim(),
        recoveryCode: useRecovery ? recoveryCode.trim() : undefined,
        memberId: mfaContext.memberId,
        organizationId: mfaContext.organizationId,
      });

      if (!result.success) {
        // Generic error message: never reveals whether the code was close or
        // valid (rate-limit rejection and invalid codes look identical).
        setMfaError(ui.auth.mfaErrorGeneric);
        return;
      }

      setStatus({
        phase: "success",
        headline: ui.auth.authenticateSuccess,
        message: ui.auth.authenticateSuccessBody,
      });
      redirectToDestination();
    } catch (error) {
      setMfaError(extractErrorMessage(error));
    } finally {
      setIsSubmittingMfa(false);
    }
  }, [mfaContext, mfaCode, useRecovery, recoveryCode, redirectToDestination]);

  const isMfaStep = status.phase === "mfa";
  const codeValid = useRecovery
    ? recoveryCode.trim().length > 0
    : mfaCode.trim().length >= 6;

  const icon =
    status.phase === "success" ? (
      <CheckCircle2 className="h-10 w-10 text-emerald-500" aria-hidden="true" />
    ) : status.phase === "error" ? (
      <AlertCircle className="h-10 w-10 text-red-500" aria-hidden="true" />
    ) : status.phase === "mfa" ? (
      <ShieldCheck className="h-10 w-10 text-emerald-500" aria-hidden="true" />
    ) : (
      <Loader2
        className="h-10 w-10 animate-spin text-emerald-500"
        aria-hidden="true"
      />
    );

  return (
    <div className="flex min-h-screen items-center justify-center bg-slate-900 px-4">
      <div className="w-full max-w-md rounded-xl border border-slate-200 bg-white px-8 py-10 text-center shadow-lg">
        <div className="flex flex-col items-center gap-4">
          {icon}
          <h1 className="text-lg font-semibold text-slate-900" role="status">
            {status.headline}
          </h1>
          <p className="text-sm text-slate-500">{status.message}</p>

          {isMfaStep ? (
            <div className="mt-4 w-full space-y-4 text-left">
              <div className="space-y-2">
                <Label
                  htmlFor="mfa-code"
                  className="text-sm font-medium text-slate-700"
                >
                  {useRecovery
                    ? ui.auth.mfaRecoveryPlaceholder
                    : ui.auth.mfaCodePlaceholder}
                </Label>
                <Input
                  id="mfa-code"
                  type="text"
                  inputMode={useRecovery ? "text" : "numeric"}
                  autoComplete={useRecovery ? "one-time-code" : "one-time-code"}
                  maxLength={useRecovery ? 64 : 6}
                  placeholder={
                    useRecovery
                      ? ui.auth.mfaRecoveryPlaceholder
                      : ui.auth.mfaCodePlaceholder
                  }
                  value={useRecovery ? recoveryCode : mfaCode}
                  onChange={(e) =>
                    useRecovery
                      ? setRecoveryCode(e.target.value)
                      : setMfaCode(e.target.value)
                  }
                  onKeyDown={(e) => {
                    if (e.key === "Enter" && codeValid && !isSubmittingMfa) {
                      void submitMfaChallenge();
                    }
                  }}
                  disabled={isSubmittingMfa}
                  autoFocus
                />
              </div>

              {mfaError ? (
                <p
                  className="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-xs font-medium text-red-700"
                  role="alert"
                >
                  {mfaError}
                </p>
              ) : null}

              <Button
                onClick={() => void submitMfaChallenge()}
                disabled={!codeValid || isSubmittingMfa}
                className="w-full justify-center bg-emerald-500 text-white hover:bg-emerald-600"
              >
                {isSubmittingMfa ? (
                  <>
                    <Loader2
                      className="mr-2 h-4 w-4 animate-spin"
                      aria-hidden="true"
                    />
                    {ui.auth.mfaVerifying}
                  </>
                ) : useRecovery ? (
                  ui.auth.mfaRecoveryVerify
                ) : (
                  ui.auth.mfaVerifyCode
                )}
              </Button>

              <button
                type="button"
                onClick={() => {
                  setUseRecovery((v) => !v);
                  setMfaError(null);
                }}
                className="block w-full text-center text-xs font-medium text-emerald-600 hover:text-emerald-700"
              >
                {useRecovery
                  ? ui.auth.mfaUseAppCode
                  : ui.auth.mfaUseRecoveryCode}
              </button>

              <p className="text-xs text-slate-400">{ui.auth.authenticateHelp}</p>
            </div>
          ) : status.phase === "error" ? (
            <div className="mt-6 flex flex-col items-center gap-2">
              <Button asChild className="w-full justify-center bg-emerald-500 hover:bg-emerald-600 text-white">
                <Link href="/auth">{ui.auth.authenticateBackToLogin}</Link>
              </Button>
              <p className="text-xs text-slate-400">
                {ui.auth.authenticateHelp}
              </p>
            </div>
          ) : null}
        </div>
      </div>
    </div>
  );
}
