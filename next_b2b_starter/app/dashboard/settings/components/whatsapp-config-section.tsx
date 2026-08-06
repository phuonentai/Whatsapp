"use client";

import { useState, useEffect, useCallback } from "react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { MessageCircle, CheckCircle, XCircle, Loader2, Copy, ExternalLink } from "lucide-react";
import { useWhatsAppConfigQuery } from "@/lib/hooks/queries/use-whatsapp-config-query";
import { useUpsertWhatsAppConfig } from "@/lib/hooks/mutations/use-upsert-whatsapp-config";
import { useToggleWhatsAppConfig } from "@/lib/hooks/mutations/use-toggle-whatsapp-config";
import type { WhatsAppConfigInput } from "@/lib/models/whatsapp-config.model";
import { toast } from "sonner";

export function WhatsAppConfigSection() {
  const { data: config, isLoading, error, refetch } = useWhatsAppConfigQuery();
  const upsertMutation = useUpsertWhatsAppConfig();
  const toggleMutation = useToggleWhatsAppConfig();

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

  const hasExistingConfig = config !== undefined;

  const callbackUrl = typeof window !== "undefined"
    ? `${window.location.origin}/api/v1/webhooks/whatsapp`
    : "";

  const copyCallbackUrl = useCallback(() => {
    navigator.clipboard.writeText(callbackUrl);
    toast.success("Callback URL copied to clipboard");
  }, [callbackUrl]);

  useEffect(() => {
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
  }, [config]);

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

  if (isLoading) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-64 rounded-xl" />
      </div>
    );
  }

  if (error && !config) {
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
          {!hasExistingConfig && (
            <Alert className="border border-blue-200 bg-blue-50">
              <AlertTitle>No connection yet</AlertTitle>
              <AlertDescription>
                To connect WhatsApp, you need a WhatsApp Business Account from Meta. Fill in the
                credentials from your Meta Business Dashboard below.
              </AlertDescription>
            </Alert>
          )}

          {validationError && (
            <Alert variant="destructive" className="border border-red-200 bg-red-50">
              <AlertDescription>{validationError}</AlertDescription>
            </Alert>
          )}

          {/* Webhook Callback URL */}
          <div className="rounded-lg border border-gray-200 bg-gray-50 p-4">
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
        </CardContent>
      </Card>
    </div>
  );
}
