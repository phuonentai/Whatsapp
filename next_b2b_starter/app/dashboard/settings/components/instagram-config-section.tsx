"use client";

import { useState, useEffect } from "react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Instagram, CheckCircle, XCircle, Loader2, Copy, RefreshCw } from "lucide-react";
import { useInstagramConfigQuery, useInstagramWebhookHealth } from "@/lib/hooks/queries/use-instagram-config-query";
import { useUpsertInstagramConfig } from "@/lib/hooks/mutations/use-upsert-instagram-config";
import { useToggleInstagramConfig } from "@/lib/hooks/mutations/use-toggle-instagram-config";
import { useRefreshInstagramToken } from "@/lib/hooks/mutations/use-refresh-instagram-token";
import { toast } from "sonner";

function isConfigNotFound(error: unknown): boolean {
  if (!error) return false;
  const message = error instanceof Error ? error.message : String(error);
  return message.includes("No Instagram configuration found") || message.includes("config_not_found");
}

function formatExpiry(iso?: string): string {
  if (!iso) return "Unknown";
  return new Date(iso).toLocaleDateString();
}

export function InstagramConfigSection() {
  const { data: config, isLoading, error, refetch } = useInstagramConfigQuery();
  const healthQuery = useInstagramWebhookHealth({ enabled: config !== undefined });
  const upsertMutation = useUpsertInstagramConfig();
  const toggleMutation = useToggleInstagramConfig();
  const refreshMutation = useRefreshInstagramToken();

  const [igUserId, setIgUserId] = useState("");
  const [igUsername, setIgUsername] = useState("");
  const [fbPageId, setFbPageId] = useState("");
  const [accessToken, setAccessToken] = useState("");
  const [webhookSecret, setWebhookSecret] = useState("");
  const [verifyToken, setVerifyToken] = useState("");
  const [apiVersion, setApiVersion] = useState("");
  const [graphApiUrl, setGraphApiUrl] = useState("");
  const [isActive, setIsActive] = useState(false);
  const [validationError, setValidationError] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  const hasExistingConfig = config !== undefined;
  const notConnected = !hasExistingConfig && isConfigNotFound(error);

  useEffect(() => {
    if (config) {
      // Sync the form from the fetched config (secrets stay blank = keep current).
      // eslint-disable-next-line react-hooks/set-state-in-effect
      setIgUserId(config.igUserId);
      setIgUsername(config.igUsername ?? "");
      setFbPageId(config.fbPageId ?? "");
      setAccessToken("");
      setWebhookSecret("");
      setVerifyToken("");
      setApiVersion(config.apiVersion ?? "");
      setGraphApiUrl(config.graphApiUrl ?? "");
      setIsActive(config.isActive);
    }
  }, [config]);

  const webhookUrl = `${window.location.origin}/api/v1/webhooks/instagram`;

  const handleCopyUrl = async () => {
    try {
      await navigator.clipboard.writeText(webhookUrl);
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {
      toast.error("Failed to copy webhook URL");
    }
  };

  const handleSave = async () => {
    setValidationError(null);
    if (!igUserId.trim()) {
      setValidationError("IG user ID is required");
      return;
    }
    if (!hasExistingConfig && (!accessToken.trim() || !webhookSecret.trim() || !verifyToken.trim())) {
      setValidationError("Access token, webhook secret, and verify token are required when connecting");
      return;
    }

    await upsertMutation.mutateAsync({
      ig_user_id: igUserId.trim(),
      ig_username: igUsername.trim() || undefined,
      fb_page_id: fbPageId.trim() || undefined,
      access_token: accessToken.trim() || undefined,
      webhook_secret: webhookSecret.trim() || undefined,
      verify_token: verifyToken.trim() || undefined,
      api_version: apiVersion.trim() || undefined,
      graph_api_url: graphApiUrl.trim() || undefined,
    });
    setAccessToken("");
    setWebhookSecret("");
    setVerifyToken("");
    refetch();
  };

  const webhookActive =
    healthQuery.data && healthQuery.data.last_24h > 0 && config?.isActive;

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Instagram className="h-5 w-5 text-pink-600" />
          Instagram Business Integration
        </CardTitle>
        <CardDescription>
          Connect your Instagram business account to receive and reply to DMs in the inbox.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {isLoading ? (
          <div className="space-y-3">
            <Skeleton className="h-8 w-full" />
            <Skeleton className="h-8 w-full" />
            <Skeleton className="h-8 w-3/4" />
          </div>
        ) : notConnected ? (
          <Alert>
            <AlertTitle>Connect Instagram</AlertTitle>
            <AlertDescription className="space-y-2">
              <p>
                Obtain these values from the Meta developer dashboard or Graph API Explorer:
              </p>
              <ul className="list-inside list-disc text-sm">
                <li>
                  <strong>IG user ID</strong> — the business Instagram account ID
                  (visible in webhook payloads as <code>recipient.id</code>)
                </li>
                <li>
                  <strong>Access token</strong> — long-lived token with{" "}
                  <code>instagram_manage_messages</code> and{" "}
                  <code>instagram_basic</code> permissions
                </li>
                <li>
                  <strong>Webhook secret / verify token</strong> — used to sign and verify
                  webhook deliveries
                </li>
              </ul>
              <p>
                Configure the webhook callback URL below in Meta&apos;s app webhook settings
                (field <code>messages</code>) with the verify token above.
              </p>
            </AlertDescription>
          </Alert>
        ) : error ? (
          <Alert variant="destructive">
            <AlertTitle>Failed to load configuration</AlertTitle>
            <AlertDescription className="flex items-center gap-2">
              {error.message}
              <Button variant="outline" size="sm" onClick={() => refetch()}>
                Retry
              </Button>
            </AlertDescription>
          </Alert>
        ) : (
          <div className="space-y-4">
            {config?.tokenExpiryWarning && (
              <Alert variant="destructive">
                <AlertTitle className="flex items-center gap-2">
                  <XCircle className="h-4 w-4" />
                  Token expiring soon
                </AlertTitle>
                <AlertDescription className="space-y-2">
                  <p>
                    Your Instagram access token expires{" "}
                    {config.tokenExpiresAt ? `on ${formatExpiry(config.tokenExpiresAt)}` : "soon"}.
                    Refresh it to keep receiving and sending DMs.
                  </p>
                  <Button
                    size="sm"
                    onClick={() => refreshMutation.mutate()}
                    disabled={refreshMutation.isPending}
                  >
                    {refreshMutation.isPending ? (
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    ) : (
                      <RefreshCw className="mr-2 h-4 w-4" />
                    )}
                    Refresh token
                  </Button>
                </AlertDescription>
              </Alert>
            )}

            {validationError && (
              <Alert variant="destructive">
                <AlertTitle>Validation error</AlertTitle>
                <AlertDescription>{validationError}</AlertDescription>
              </Alert>
            )}

            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="ig-user-id">IG user ID</Label>
                <Input
                  id="ig-user-id"
                  value={igUserId}
                  onChange={(e) => setIgUserId(e.target.value)}
                  placeholder="e.g. 17841400000000000"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="ig-username">IG username</Label>
                <Input
                  id="ig-username"
                  value={igUsername}
                  onChange={(e) => setIgUsername(e.target.value)}
                  placeholder="e.g. mi.tienda"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="fb-page-id">Facebook page ID</Label>
                <Input
                  id="fb-page-id"
                  value={fbPageId}
                  onChange={(e) => setFbPageId(e.target.value)}
                  placeholder="Optional"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="access-token">Access token</Label>
                <Input
                  id="access-token"
                  type="password"
                  value={accessToken}
                  onChange={(e) => setAccessToken(e.target.value)}
                  placeholder={hasExistingConfig ? "Leave blank to keep current" : "Enter access token"}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="webhook-secret">Webhook secret</Label>
                <Input
                  id="webhook-secret"
                  type="password"
                  value={webhookSecret}
                  onChange={(e) => setWebhookSecret(e.target.value)}
                  placeholder={hasExistingConfig ? "Leave blank to keep current" : "Enter webhook secret"}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="verify-token">Verify token</Label>
                <Input
                  id="verify-token"
                  type="password"
                  value={verifyToken}
                  onChange={(e) => setVerifyToken(e.target.value)}
                  placeholder={hasExistingConfig ? "Leave blank to keep current" : "Enter verify token"}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="api-version">API version</Label>
                <Input
                  id="api-version"
                  value={apiVersion}
                  onChange={(e) => setApiVersion(e.target.value)}
                  placeholder="v21.0"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="graph-api-url">Graph API URL</Label>
                <Input
                  id="graph-api-url"
                  value={graphApiUrl}
                  onChange={(e) => setGraphApiUrl(e.target.value)}
                  placeholder="https://graph.facebook.com"
                />
              </div>
            </div>

            <div className="flex items-center justify-between rounded-lg border p-3">
              <div>
                <p className="text-sm font-medium">Active</p>
                <p className="text-xs text-gray-500">
                  {isActive ? "Receiving and sending DMs" : "Instagram messaging paused"}
                </p>
              </div>
              <Switch
                checked={isActive}
                onCheckedChange={(checked) => {
                  setIsActive(checked);
                  toggleMutation.mutate();
                }}
              />
            </div>

            <div className="rounded-lg border p-3">
              <p className="text-sm font-medium">Webhook callback URL</p>
              <div className="mt-2 flex items-center gap-2">
                <code className="flex-1 truncate rounded bg-gray-100 px-2 py-1 text-xs">
                  {webhookUrl}
                </code>
                <Button variant="outline" size="sm" onClick={handleCopyUrl}>
                  {copied ? <CheckCircle className="h-4 w-4 text-green-600" /> : <Copy className="h-4 w-4" />}
                  {copied ? "Copied" : "Copy"}
                </Button>
              </div>
              <p className="mt-2 text-xs text-gray-500">
                Register this URL in your Meta app&apos;s webhook settings with the{" "}
                <code>messages</code> field subscribed.
              </p>
            </div>

            <div className="rounded-lg border p-3">
              <p className="text-sm font-medium">Webhook health</p>
              <div className="mt-2 flex items-center gap-2 text-xs">
                {healthQuery.isLoading ? (
                  <Skeleton className="h-4 w-40" />
                ) : webhookActive ? (
                  <span className="flex items-center gap-1.5 text-green-600">
                    <CheckCircle className="h-4 w-4" />
                    Webhooks active — last 24h: {healthQuery.data?.last_24h ?? 0} deliveries
                  </span>
                ) : (
                  <span className="flex items-center gap-1.5 text-yellow-600">
                    <XCircle className="h-4 w-4" />
                    No webhooks received in the last 24 hours
                  </span>
                )}
              </div>
            </div>

            <Button onClick={handleSave} disabled={upsertMutation.isPending}>
              {upsertMutation.isPending ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : null}
              Save
            </Button>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
