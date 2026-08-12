"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import Link from "next/link";
import { StytchB2B, useStytchMember } from "@stytch/nextjs/b2b";
import { StytchEventType } from "@stytch/vanilla-js";
import { Home, ShieldCheck } from "lucide-react";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { LogoMark } from "@/components/marketing/site-header";
import { useStytchConfig } from "@/lib/contexts/stytch-config-context";
import { PRODUCT_NAME } from "@/lib/brand";
import { ui, tpl } from "@/lib/copy/ui";

const trustPoints = [
  {
    title: ui.auth.highlightData,
    body: ui.auth.highlightDataBody,
  },
  {
    title: ui.auth.highlightPasswordless,
    body: ui.auth.highlightPasswordlessBody,
  },
  {
    title: ui.auth.highlightOrg,
    body: ui.auth.highlightOrgBody,
  },
];

/**
 * Auth surface (stytch-enterprise-suite / stytch-login-surface): the pre-built
 * Stytch B2B component in Discovery mode renders the project-enabled primary
 * methods — email magic link, OAuth social (Google/Microsoft), email OTP, SSO —
 * plus passkeys per the project configuration. Org self-serve creation stays
 * hidden (`disableCreateOrganization` + dashboard toggle); the governed
 * `POST /auth/signup` remains the only org-creation path.
 *
 * The custom email form with membership pre-validation (`check-email` +
 * `sendMagicLink`) is retired from this page; Stytch's discovery flow surfaces
 * no joinable organization for unknown addresses (org-existence concealment).
 */
export default function AuthPage() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const config = useStytchConfig();
  const { member, isInitialized } = useStytchMember();
  const [uiError, setUiError] = useState<string | null>(null);
  const [isRedirecting, setIsRedirecting] = useState(false);
  const hasRedirectedRef = useRef(false);
  const redirectTimeoutRef = useRef<number | null>(null);

  const targetAfterLogin = useMemo(() => {
    const returnTo = searchParams.get("returnTo") || "/dashboard";
    // Validate returnTo is a safe relative path:
    // - Must start with single /
    // - Cannot be protocol-relative (//), contain backslash (\), or have : before first /
    const isSafeRelativePath =
      returnTo.startsWith("/") &&
      !returnTo.startsWith("//") &&
      !returnTo.includes("\\") &&
      !returnTo.slice(1).includes(":");
    return isSafeRelativePath ? returnTo : "/dashboard";
  }, [searchParams]);

  // Absolute redirect target consumed by Stytch after magic-link/OAuth/OTP
  // exchange: the existing /authenticate page completes the session.
  const authenticateUrl = useMemo(
    () => new URL("/authenticate", config.baseUrl).toString(),
    [config.baseUrl]
  );

  const handleAuthSuccess = useCallback(() => {
    if (hasRedirectedRef.current) return;
    hasRedirectedRef.current = true;
    setIsRedirecting(true);
    router.replace(targetAfterLogin);
    setTimeout(() => {
      router.refresh();
    }, 150);
    if (typeof window !== "undefined") {
      if (redirectTimeoutRef.current !== null) {
        window.clearTimeout(redirectTimeoutRef.current);
      }
      redirectTimeoutRef.current = window.setTimeout(() => {
        window.location.assign(targetAfterLogin);
      }, 1500);
    }
  }, [router, targetAfterLogin]);

  useEffect(() => {
    if (!isInitialized) return;
    if (member) {
      // Intentional: redirect once the pre-built flow establishes the member
      // session. Synchronous setState here is the sanctioned adjustment (the
      // page must not render the login form for an authenticated member).
      // eslint-disable-next-line react-hooks/set-state-in-effect
      handleAuthSuccess();
    }
  }, [isInitialized, member, handleAuthSuccess]);

  useEffect(() => {
    router.prefetch(targetAfterLogin);
  }, [router, targetAfterLogin]);

  useEffect(() => {
    return () => {
      if (typeof window !== "undefined" && redirectTimeoutRef.current !== null) {
        window.clearTimeout(redirectTimeoutRef.current);
        redirectTimeoutRef.current = null;
      }
    };
  }, []);

  if (!isInitialized) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-slate-900">
        <div className="flex flex-col items-center gap-3">
          <div className="h-10 w-10 animate-spin rounded-full border-4 border-slate-700 border-t-emerald-500" />
          <p className="text-sm text-slate-400">{ui.auth.checkingSession}</p>
        </div>
      </div>
    );
  }

  if (isRedirecting || member) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-slate-900">
        <div className="flex flex-col items-center gap-3 rounded-xl border border-slate-700 bg-slate-800 px-8 py-10 shadow-lg">
          <div className="h-10 w-10 animate-spin rounded-full border-4 border-slate-700 border-t-emerald-500" />
          <div className="text-center">
            <p className="text-sm font-medium text-white">{ui.auth.redirectingTitle}</p>
            <p className="text-sm text-slate-400">{ui.auth.redirectingBody}</p>
            <p className="mt-4 text-xs text-slate-400">
              {ui.auth.slowerHint}{" "}
              <a
                href={targetAfterLogin}
                className="font-medium text-emerald-400 hover:underline"
              >
                {ui.auth.openWorkspace}
              </a>
              .
            </p>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-slate-50">
      <div className="mx-auto grid min-h-screen w-full max-w-6xl lg:grid-cols-2">
        {/* Panel oscuro de marca (plantilla Verifika) */}
        <aside className="relative hidden overflow-hidden bg-slate-900 text-white lg:flex lg:flex-col lg:justify-between p-12">
          <div
            aria-hidden="true"
            className="pointer-events-none absolute inset-0 bg-[radial-gradient(ellipse_at_top_left,rgba(16,185,129,0.20),transparent_55%)]"
          />
          <div className="relative">
            <Link href="/" className="flex items-center gap-3" aria-label={`${PRODUCT_NAME} — Inicio`}>
              <LogoMark />
              <span className="font-heading font-bold text-xl tracking-tight">{PRODUCT_NAME}</span>
            </Link>
            <div className="mt-12">
              <Badge
                variant="outline"
                className="w-fit items-center gap-2 border-emerald-500/30 bg-emerald-500/10 text-xs font-medium text-emerald-400"
              >
                <ShieldCheck className="h-3.5 w-3.5" aria-hidden />
                {ui.auth.secureAccessBadge}
              </Badge>
              <h1 className="mt-6 font-heading text-4xl font-bold tracking-tight text-balance">
                {ui.auth.welcomeBack}{" "}
                <span className="text-emerald-400">{PRODUCT_NAME}</span>
              </h1>
              <p className="mt-4 max-w-md text-lg text-slate-400 leading-relaxed">
                {ui.auth.authLead}
              </p>
            </div>
          </div>
          <div className="relative space-y-4">
            {trustPoints.map((point) => (
              <div key={point.title} className="flex items-start gap-3">
                <ShieldCheck className="mt-0.5 h-5 w-5 flex-none text-emerald-400" aria-hidden />
                <div>
                  <p className="text-sm font-semibold text-white">{point.title}</p>
                  <p className="text-sm text-slate-400">{point.body}</p>
                </div>
              </div>
            ))}
          </div>
        </aside>

        {/* Stytch B2B component */}
        <main className="flex items-center justify-center px-6 py-16">
          <div className="w-full max-w-md">
            <div className="mb-8 flex items-center justify-between lg:hidden">
              <Link href="/" className="flex items-center gap-3" aria-label={`${PRODUCT_NAME} — Inicio`}>
                <LogoMark />
                <span className="font-heading font-bold text-xl tracking-tight text-slate-900">{PRODUCT_NAME}</span>
              </Link>
              <Link href="/" className="flex items-center gap-2 text-sm text-slate-500 hover:text-emerald-600 transition-colors">
                <Home className="h-4 w-4" />
                <span>{ui.auth.home}</span>
              </Link>
            </div>
            <div className="rounded-2xl border border-slate-200 bg-white p-8 shadow-lg shadow-slate-900/5">
              <div className="mb-6 space-y-2 text-center">
                <h2 className="font-heading text-2xl font-bold text-slate-900">
                  {ui.auth.signInCardTitle} {PRODUCT_NAME}
                </h2>
                <p className="text-sm text-slate-500">{ui.auth.signInCardLead}</p>
              </div>

              {uiError ? (
                <Alert variant="destructive" className="mb-6 text-left">
                  <AlertTitle>{ui.auth.errorSnagTitle}</AlertTitle>
                  <AlertDescription>{uiError}</AlertDescription>
                </Alert>
              ) : null}

              {/* data-testid anchors the login-surface component tests */}
              <div data-testid="stytch-b2b-login">
                <StytchB2B
                  config={{
                    authFlowType: "Discovery",
                    products: ["emailMagicLinks", "oauth", "emailOtp", "sso"],
                    // Org self-serve creation stays hidden: POST /auth/signup
                    // is the only org-creation path.
                    disableCreateOrganization: true,
                    emailMagicLinksOptions: {
                      discoveryRedirectURL: authenticateUrl,
                      locale: "es",
                    },
                    oauthOptions: {
                      discoveryRedirectURL: authenticateUrl,
                      locale: "es",
                      providers: [{ type: "google" }, { type: "microsoft" }],
                    },
                    emailOtpOptions: {
                      locale: "es",
                    },
                    ssoOptions: {
                      loginRedirectURL: authenticateUrl,
                      signupRedirectURL: authenticateUrl,
                    },
                    sessionOptions: {
                      sessionDurationMinutes: config.sessionDurationMinutes,
                    },
                  }}
                  styles={{
                    fontFamily: "Inter, ui-sans-serif, system-ui, sans-serif",
                    container: { width: "100%" },
                    colors: { primary: "#10b981" },
                  }}
                  callbacks={{
                    onEvent: (event) => {
                      if (event.type === StytchEventType.AuthenticateFlowComplete) {
                        handleAuthSuccess();
                      }
                    },
                    onError: (error) => {
                      setUiError(error.message);
                    },
                  }}
                />
              </div>

              <p className="mt-6 text-center text-xs text-slate-400">
                {ui.auth.termsNote}
              </p>
              <p className="mt-4 text-center text-sm text-slate-500">
                {ui.auth.noAccount}{" "}
                <Link href="/signup" className="text-emerald-600 hover:underline font-medium">
                  {ui.auth.signUp}
                </Link>
              </p>
            </div>
            <div className="mt-6 rounded-xl border border-slate-200 bg-white p-5">
              <h2 className="text-sm font-semibold text-slate-900">
                {ui.auth.needHelp}
              </h2>
              <p className="mt-2 text-sm text-slate-500">
                {tpl(ui.auth.supportNote, { email: ui.auth.supportEmail })}
              </p>
            </div>
          </div>
        </main>
      </div>
    </div>
  );
}
