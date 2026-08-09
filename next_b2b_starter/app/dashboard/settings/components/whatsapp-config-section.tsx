"use client";

import { useState, useEffect, useCallback } from "react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { MessageCircle, CheckCircle, XCircle, Loader2, Copy, ExternalLink, ChevronDown, ChevronUp, LifeBuoy } from "lucide-react";
import { useWhatsAppConfigQuery } from "@/lib/hooks/queries/use-whatsapp-config-query";
import { useUpsertWhatsAppConfig } from "@/lib/hooks/mutations/use-upsert-whatsapp-config";
import { useToggleWhatsAppConfig } from "@/lib/hooks/mutations/use-toggle-whatsapp-config";
import { useWhatsAppSignupMetaConfig, useWhatsAppSignupStatus } from "@/lib/hooks/queries/use-whatsapp-signup-query";
import { useWhatsAppSignupExchange } from "@/lib/hooks/mutations/use-whatsapp-signup-mutation";
import type { WhatsAppConfigInput } from "@/lib/models/whatsapp-config.model";
import { toast } from "sonner";

declare global {
  interface Window {
    FB?: {
      init: (opts: Record<string, unknown>) => void;
      login: (
        callback: (response: {
          authResponse?: { code?: string; accessToken?: string };
          error?: { message?: string };
          status?: string;
        }) => void,
        opts?: Record<string, unknown>
      ) => void;
    };
  }
}

const MICRO_STATUS_STEPS = [
  "Connecting your WhatsApp...",
  "Verifying Coexistence session...",
  "Establishing secure token & webhooks...",
  "All set! Your WhatsApp is live.",
];

const IN_PROGRESS_STATUSES = new Set(["exchanging", "registering", "verifying"]);

function isConfigNotFound(error: unknown): boolean {
  if (!error) return false;
  const message = error instanceof Error ? error.message : String(error);
  return message.includes("No WhatsApp configuration found") || message.includes("config_not_found");
}

function loadFBSDK(appId: string): Promise<void> {
  return new Promise((resolve, reject) => {
    if (window.FB) {
      window.FB.init({ appId, version: "v21.0", cookie: true, xfbml: false });
      resolve();
      return;
    }
    const existing = document.getElementById("fb-jssdk") as HTMLScriptElement | null;
    if (existing) {
      existing.addEventListener("load", () => resolve());
      existing.addEventListener("error", () => reject(new Error("Failed to load the Meta SDK")));
      return;
    }
    const script = document.createElement("script");
    script.id = "fb-jssdk";
    script.src = "https://connect.facebook.net/en_US/sdk.js";
    script.async = true;
    script.onload = () => {
      window.FB?.init({ appId, version: "v21.0", cookie: true, xfbml: false });
      resolve();
    };
    script.onerror = () => reject(new Error("Failed to load the Meta SDK"));
    document.body.appendChild(script);
  });
}

export function WhatsAppConfigSection() {
  const { data: config, isLoading, error, refetch } = useWhatsAppConfigQuery();
  const upsertMutation = useUpsertWhatsAppConfig();
  const toggleMutation = useToggleWhatsAppConfig();
  const metaConfigQuery = useWhatsAppSignupMetaConfig();
  const exchangeMutation = useWhatsAppSignupExchange();

  const [phoneNumberId, setPhoneNumberId] = useState("");
  const [businessPhone, setBusinessPhone] = useState("");
  const [webhookSecret, setWebhookSecret] = useState("");
  const [verifyToken, setVerifyToken] = useState("");
  const [appId, setAppId] = useState("");
  const [wabaId, setWabaId] = useState("");
  const [accessToken, setAccessToken] = useState("");
  const [apiVersion, setApiVersion] = useState("");
  const [graphApiUrl, setGraphApiUrl] = useState("");
  const [isActive, setIsActive] = useState(false);
  const [validationError, setValidationError] = useState<string | null>(null);
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [connectError, setConnectError] = useState<string | null>(null);
  const [signupErrorCode, setSignupErrorCode] = useState<string | null>(null);
  const [microStepIndex, setMicroStepIndex] = useState(0);

  const hasExistingConfig = config !== undefined;
  const notConnected = !hasExistingConfig && isConfigNotFound(error);

  const signupStatusQuery = useWhatsAppSignupStatus({
    enabled: notConnected,
    refetchInterval: (query) =>
      query.state.data && IN_PROGRESS_STATUSES.has(query.state.data.status) ? 3000 : false,
  });

  const signupInProgress =
    exchangeMutation.isPending ||
    (signupStatusQuery.data !== undefined &&
      IN_PROGRESS_STATUSES.has(signupStatusQuery.data.status));

  const signupFailed =
    signupStatusQuery.data?.status === "failed" ||
    signupErrorCode !== null ||
    connectError !== null;

  // Map a persisted flow status to a micro-status step (data-driven, no state).
  const statusStepIndex =
    signupStatusQuery.data?.status === "exchanging"
      ? 0
      : signupStatusQuery.data?.status === "registering"
        ? 1
        : signupStatusQuery.data?.status === "verifying"
          ? 2
          : 0;

  const displayedStepIndex = exchangeMutation.isPending
    ? 0
    : signupInProgress
      ? statusStepIndex
      : microStepIndex;

  const callbackUrl = typeof window !== "undefined"
    ? `${window.location.origin}/api/v1/webhooks/whatsapp`
    : "";

  const copyCallbackUrl = useCallback(() => {
    navigator.clipboard.writeText(callbackUrl);
    toast.success("Callback URL copied to clipboard");
  }, [callbackUrl]);

  // Sanctioned render-phase state adjustment (React "adjusting state during
  // render" pattern): re-seed the form when the fetched config changes.
  const [prevConfig, setPrevConfig] = useState(config);
  if (config !== prevConfig) {
    setPrevConfig(config);
    if (config) {
      setPhoneNumberId(config.phoneNumberId);
      setBusinessPhone(config.businessPhone);
      setWebhookSecret("");
      setVerifyToken("");
      setAppId(config.appId ?? "");
      setWabaId(config.wabaId ?? "");
      setAccessToken("");
      setApiVersion(config.apiVersion ?? "v21.0");
      setGraphApiUrl(config.graphApiUrl ?? "https://graph.facebook.com");
      setIsActive(config.isActive);
    }
  }

  // When a polled flow reaches connected, surface the config.
  useEffect(() => {
    if (signupStatusQuery.data?.status === "connected") {
      refetch();
    }
  }, [signupStatusQuery.data?.status, refetch]);

  const handleSubmit = async () => {
    setValidationError(null);

    if (!phoneNumberId.trim() || !businessPhone.trim()) {
      setValidationError("Phone Number ID and Business Phone are required");
      return;
    }

    if (!hasExistingConfig && (!webhookSecret.trim() || !verifyToken.trim())) {
      setValidationError("Webhook Secret and Verify Token are required for new connections");
      return;
    }

    const input: WhatsAppConfigInput = {
      phone_number_id: phoneNumberId.trim(),
      business_phone: businessPhone.trim(),
      webhook_secret: webhookSecret.trim() || undefined,
      verify_token: verifyToken.trim() || undefined,
      app_id: appId.trim() || undefined,
      waba_id: wabaId.trim() || undefined,
      access_token: accessToken.trim() || undefined,
      api_version: apiVersion.trim() || undefined,
      graph_api_url: graphApiUrl.trim() || undefined,
    };

    try {
      await upsertMutation.mutateAsync(input);
    } catch {
    }
  };

  const handleToggle = async () => {
    try {
      await toggleMutation.mutateAsync();
    } catch {
    }
  };

  const handleConnect = async () => {
    setConnectError(null);
    setSignupErrorCode(null);
    setMicroStepIndex(0);

    try {
      const meta = await metaConfigQuery.refetch();
      const metaData = meta.data;
      if (!metaData) {
        setConnectError("Could not load the WhatsApp signup configuration.");
        return;
      }
      await loadFBSDK(metaData.app_id);

      const code = await new Promise<string | null>((resolve) => {
        if (!window.FB) {
          setConnectError("The Meta SDK is not available.");
          resolve(null);
          return;
        }
        window.FB.login(
          (response) => {
            if (response.error?.message) {
              setConnectError(response.error.message);
              resolve(null);
              return;
            }
            const authCode = response.authResponse?.code;
            if (!authCode) {
              setConnectError("Meta did not return an authorization code. Please try again.");
              resolve(null);
              return;
            }
            resolve(authCode);
          },
          {
            config_id: metaData.config_id,
            response_type: "code",
            override_default_response_type: true,
            ...(metaData.redirect_uri ? { extras: { redirect_uri: metaData.redirect_uri } } : {}),
          }
        );
      });

      if (!code) return;

      const result = await exchangeMutation.mutateAsync(code);
      if (result.status === "connected") {
        setMicroStepIndex(MICRO_STATUS_STEPS.length - 1);
        toast.success("WhatsApp connected successfully!");
        refetch();
      }
    } catch (err) {
      const message = err instanceof Error ? err.message : "Connection failed";
      if (message.includes("signup_failed")) {
        setSignupErrorCode(message);
        setConnectError("WhatsApp connection failed during provisioning.");
      } else if (message.includes("signup_already_connected")) {
        setConnectError("This organization is already connected.");
        refetch();
      } else if (message.includes("signup_in_progress")) {
        setConnectError("A connection is already in progress.");
      } else {
        setConnectError(message);
      }
    }
  };

  if (isLoading) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-64 rounded-xl" />
      </div>
    );
  }

  if (error && !notConnected) {
    return (
      <Alert variant="destructive" className="border border-red-200 bg-red-50">
        <AlertTitle>Failed to load configuration</AlertTitle>
        <AlertDescription>
          {error.message || "Could not fetch WhatsApp configuration. Please try again."}
        </AlertDescription>
        <Button variant="outline" size="sm" className="mt-3" onClick={() => refetch()}>
          Retry
        </Button>
      </Alert>
    );
  }

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <MessageCircle className="h-6 w-6 text-gray-600" />
              <div>
                <CardTitle>WhatsApp Business Integration</CardTitle>
                <CardDescription>
                  Connect your WhatsApp Business Account to receive and manage messages.
                </CardDescription>
              </div>
            </div>
            {hasExistingConfig && (
              <div className="flex items-center gap-2">
                <Label htmlFor="active-toggle" className="text-sm font-medium">
                  {isActive ? "Active" : "Inactive"}
                </Label>
                <Switch
                  id="active-toggle"
                  checked={isActive}
                  onCheckedChange={handleToggle}
                  disabled={toggleMutation.isPending}
                />
              </div>
            )}
          </div>
        </CardHeader>

        <CardContent className="space-y-5">
          {!hasExistingConfig && !signupInProgress && (
            <div className="flex flex-col items-center gap-4 rounded-xl border border-dashed border-gray-300 bg-gray-50 px-6 py-10 text-center">
              <MessageCircle className="h-10 w-10 text-gray-400" />
              <div>
                <h3 className="text-lg font-semibold text-gray-900">Connect WhatsApp</h3>
                <p className="mt-1 max-w-md text-sm text-gray-500">
                  Keep your mobile WhatsApp Business app and chat history while enabling
                  automated messaging through the Cloud API. Meta will guide you through
                  picking your business, number, and verification.
                </p>
              </div>
              <Button onClick={handleConnect} className="bg-emerald-600 text-white hover:bg-emerald-700">
                Connect WhatsApp
              </Button>
            </div>
          )}

          {signupInProgress && (
            <div className="space-y-3 rounded-xl border border-blue-200 bg-blue-50 px-6 py-6">
              <div className="flex items-center gap-2 text-sm font-medium text-blue-800">
                <Loader2 className="h-4 w-4 animate-spin" />
                {MICRO_STATUS_STEPS[Math.min(displayedStepIndex, MICRO_STATUS_STEPS.length - 1)]}
              </div>
              <div className="flex gap-1.5">
                {MICRO_STATUS_STEPS.slice(0, 3).map((step, idx) => (
                  <div
                    key={step}
                    className={`h-1.5 flex-1 rounded-full animate-pulse ${
                      idx <= displayedStepIndex ? "bg-emerald-500" : "bg-blue-200"
                    }`}
                  />
                ))}
              </div>
            </div>
          )}

          {signupFailed && (
            <Alert variant="destructive" className="border border-red-200 bg-red-50">
              <AlertTitle className="flex items-center gap-2">
                <LifeBuoy className="h-4 w-4" />
                Connection failed
              </AlertTitle>
              <AlertDescription>
                {connectError ?? "WhatsApp connection failed during provisioning."}
                {(signupStatusQuery.data?.error_code || signupErrorCode) && (
                  <span className="mt-1 block text-xs text-gray-500">
                    Error code: {signupStatusQuery.data?.error_code ?? signupErrorCode} — contact
                    support if this persists.
                  </span>
                )}
              </AlertDescription>
              <Button variant="outline" size="sm" className="mt-3" onClick={handleConnect}>
                Try again
              </Button>
            </Alert>
          )}

          {hasExistingConfig && config && (
            <div className="flex flex-col gap-3 rounded-xl border border-emerald-200 bg-emerald-50 px-6 py-4">
              <div className="flex items-center gap-2 text-sm font-medium text-emerald-800">
                <CheckCircle className="h-4 w-4" />
                WhatsApp connected
              </div>
              <div className="grid gap-2 text-sm text-gray-600 sm:grid-cols-3">
                <div>
                  <p className="text-xs text-gray-500">Business phone</p>
                  <p className="font-medium text-gray-800">{config.businessPhone}</p>
                </div>
                <div>
                  <p className="text-xs text-gray-500">Phone number ID</p>
                  <p className="font-medium text-gray-800">{config.phoneNumberId}</p>
                </div>
                <div>
                  <p className="text-xs text-gray-500">WABA ID</p>
                  <p className="font-medium text-gray-800">{config.wabaId ?? "—"}</p>
                </div>
              </div>
            </div>
          )}

          <div className="border-t border-gray-200 pt-4">
            <Button
              variant="ghost"
              size="sm"
              className="text-gray-500"
              onClick={() => setShowAdvanced((v) => !v)}
            >
              {showAdvanced ? <ChevronUp className="mr-1 h-4 w-4" /> : <ChevronDown className="mr-1 h-4 w-4" />}
              Advanced settings
            </Button>
          </div>

          {showAdvanced && (
            <div className="space-y-5 rounded-lg border border-gray-200 bg-gray-50 p-4">
              <div>
                <Label className="text-sm font-medium">Webhook Callback URL</Label>
                <p className="text-xs text-gray-500 mb-2">
                  Configure this URL in your Meta WhatsApp Business Dashboard under Webhook settings.
                </p>
                <div className="flex items-center gap-2">
                  <code className="flex-1 rounded-md border border-gray-300 bg-white px-3 py-2 text-sm text-gray-700 truncate">
                    {callbackUrl}
                  </code>
                  <Button variant="outline" size="sm" onClick={copyCallbackUrl} className="shrink-0">
                    <Copy className="h-4 w-4 mr-1" />
                    Copy
                  </Button>
                </div>
              </div>

              {validationError && (
                <Alert variant="destructive" className="border border-red-200 bg-red-50">
                  <AlertDescription>{validationError}</AlertDescription>
                </Alert>
              )}

              <div className="grid gap-5 sm:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="phone-number-id" className="text-sm font-medium">
                    Phone Number ID <span className="text-red-500">*</span>
                  </Label>
                  <Input
                    id="phone-number-id"
                    type="text"
                    placeholder="e.g. 123456789012345"
                    value={phoneNumberId}
                    onChange={(e) => setPhoneNumberId(e.target.value)}
                    disabled={upsertMutation.isPending}
                  />
                  <p className="text-xs text-gray-500">Found in Meta Business Dashboard under WhatsApp &gt; API Setup</p>
                </div>

                <div className="space-y-2">
                  <Label htmlFor="business-phone" className="text-sm font-medium">
                    Business Phone <span className="text-red-500">*</span>
                  </Label>
                  <Input
                    id="business-phone"
                    type="text"
                    placeholder="e.g. +573001234567"
                    value={businessPhone}
                    onChange={(e) => setBusinessPhone(e.target.value)}
                    disabled={upsertMutation.isPending}
                  />
                  <p className="text-xs text-gray-500">E.164 format with country code (e.g. +57 for Colombia)</p>
                </div>

                <div className="space-y-2">
                  <Label htmlFor="webhook-secret" className="text-sm font-medium">
                    Webhook Secret
                  </Label>
                  <Input
                    id="webhook-secret"
                    type="password"
                    placeholder={hasExistingConfig ? "Leave blank to keep current" : "Enter webhook secret"}
                    value={webhookSecret}
                    onChange={(e) => setWebhookSecret(e.target.value)}
                    disabled={upsertMutation.isPending}
                  />
                  <p className="text-xs text-gray-500">
                    {hasExistingConfig
                      ? "Leave blank to keep the existing value unchanged"
                      : "Required for new connections. Found in Meta Business Dashboard."}
                  </p>
                </div>

                <div className="space-y-2">
                  <Label htmlFor="verify-token" className="text-sm font-medium">
                    Verify Token
                  </Label>
                  <Input
                    id="verify-token"
                    type="password"
                    placeholder={hasExistingConfig ? "Leave blank to keep current" : "Enter verify token"}
                    value={verifyToken}
                    onChange={(e) => setVerifyToken(e.target.value)}
                    disabled={upsertMutation.isPending}
                  />
                  <p className="text-xs text-gray-500">
                    {hasExistingConfig
                      ? "Leave blank to keep the existing value unchanged"
                      : "Required for new connections. You define this token."}
                  </p>
                </div>

                <div className="space-y-2">
                  <Label htmlFor="waba-id" className="text-sm font-medium">
                    WABA ID
                  </Label>
                  <Input
                    id="waba-id"
                    type="text"
                    placeholder="e.g. 123456789012345"
                    value={wabaId}
                    onChange={(e) => setWabaId(e.target.value)}
                    disabled={upsertMutation.isPending}
                  />
                  <p className="text-xs text-gray-500">WhatsApp Business Account ID</p>
                </div>

                <div className="space-y-2">
                  <Label htmlFor="access-token" className="text-sm font-medium">
                    Permanent Access Token
                  </Label>
                  <Input
                    id="access-token"
                    type="password"
                    placeholder={hasExistingConfig ? "Leave blank to keep current" : "Enter access token"}
                    value={accessToken}
                    onChange={(e) => setAccessToken(e.target.value)}
                    disabled={upsertMutation.isPending}
                  />
                  <p className="text-xs text-gray-500">
                    {hasExistingConfig
                      ? "Leave blank to keep the existing value unchanged"
                      : "Required for sending messages. Generate in Meta Business Dashboard."}
                  </p>
                </div>

                <div className="space-y-2">
                  <Label htmlFor="api-version" className="text-sm font-medium">
                    API Version
                  </Label>
                  <Input
                    id="api-version"
                    type="text"
                    placeholder="v21.0"
                    value={apiVersion}
                    onChange={(e) => setApiVersion(e.target.value)}
                    disabled={upsertMutation.isPending}
                  />
                  <p className="text-xs text-gray-500">WhatsApp Cloud API version (default: v21.0)</p>
                </div>

                <div className="space-y-2">
                  <Label htmlFor="graph-api-url" className="text-sm font-medium">
                    Graph API URL
                  </Label>
                  <Input
                    id="graph-api-url"
                    type="text"
                    placeholder="https://graph.facebook.com"
                    value={graphApiUrl}
                    onChange={(e) => setGraphApiUrl(e.target.value)}
                    disabled={upsertMutation.isPending}
                  />
                  <p className="text-xs text-gray-500">Base URL for Graph API (default: https://graph.facebook.com)</p>
                </div>

                <div className="space-y-2">
                  <Label htmlFor="app-id" className="text-sm font-medium">
                    App ID
                  </Label>
                  <Input
                    id="app-id"
                    type="text"
                    placeholder="Optional"
                    value={appId}
                    onChange={(e) => setAppId(e.target.value)}
                    disabled={upsertMutation.isPending}
                  />
                  <p className="text-xs text-gray-500">WhatsApp Business App ID (optional)</p>
                </div>
              </div>

              <div className="flex items-center gap-4 pt-2">
                <Button
                  onClick={handleSubmit}
                  disabled={upsertMutation.isPending}
                  className="bg-gray-900 text-white hover:bg-gray-800"
                >
                  {upsertMutation.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                  {hasExistingConfig ? "Save changes" : "Connect WhatsApp"}
                </Button>

                {hasExistingConfig && config && (
                  <div className="flex items-center gap-2 text-sm">
                    {config.isActive ? (
                      <>
                        <CheckCircle className="h-4 w-4 text-emerald-500" />
                        <span className="text-emerald-700">Messages are being received</span>
                      </>
                    ) : (
                      <>
                        <XCircle className="h-4 w-4 text-gray-400" />
                        <span className="text-gray-500">Message receiving is paused</span>
                      </>
                    )}
                  </div>
                )}
              </div>
            </div>
          )}

          <p className="flex items-center gap-1 text-xs text-gray-400">
            <ExternalLink className="h-3 w-3" />
            Requiere configuracion previa de la app de Meta (Embedded Signup).
          </p>
        </CardContent>
      </Card>
    </div>
  );
}
