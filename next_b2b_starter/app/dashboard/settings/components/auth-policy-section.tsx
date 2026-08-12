"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { Loader2, ShieldCheck } from "lucide-react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import { Switch } from "@/components/ui/switch";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import {
  organizationRepository,
  type AllowedAuthMethod,
  type AuthPolicyMirror,
} from "@/lib/api/api/repositories/organization-repository";
import { usePermissions } from "@/lib/hooks/use-permissions";
import { PERMISSIONS } from "@/lib/auth/permissions";
import { ui } from "@/lib/copy/ui";

const METHOD_LABELS: Record<AllowedAuthMethod, string> = {
  magic_link: "Magic link por correo",
  email_otp: "Código por correo (email OTP)",
  sso: "SSO (SAML / OIDC)",
  google_oauth: "Google",
  microsoft_oauth: "Microsoft",
};

// The backend always writes `auth_methods: RESTRICTED` + the complete
// allowed-method list; the first write preserves the org's current effective
// set (at minimum magic_link). `magic_link` is the non-removable floor.
const PRIMARY_METHODS: AllowedAuthMethod[] = [
  "magic_link",
  "email_otp",
  "google_oauth",
  "microsoft_oauth",
  "sso",
];

function is503(error: unknown): boolean {
  return error instanceof Error && error.message.includes("503");
}

/**
 * Settings `?view=access` → "Política de autenticación" card (JIT policy UI,
 * stytch-enterprise-suite / stytch-login-surface + enterprise-sso).
 *
 * Reads the org auth policy from GET /api/organizations/auth-policy — a
 * display-only mirror that is NEVER used for authorization (Stytch enforces
 * auth methods and JIT at authentication time). The card toggles:
 *
 * - Domain-restricted email JIT join (`email_jit_provisioning: RESTRICTED` +
 *   `email_allowed_domains`). Copy states that domain JIT also enables OAuth
 *   JIT for verified provider emails on the allowed domains.
 * - SSO-JIT (`sso_jit_provisioning: RESTRICTED` scoped to the org's active
 *   connection ids, least privilege) — rendered only when the org has ≥1
 *   active SSO connection; never org-wide ALL_ALLOWED.
 * - Allowed primary methods (enforced-list mode: the backend writes
 *   `auth_methods: RESTRICTED` + the complete list).
 *
 * Saves via PUT /api/organizations/auth-policy (org:manage gated,
 * circuit-breaker protected). Breaker-open/unreachable surfaces the structured
 * 503 codes `auth_policy_unavailable` (read) / `auth_policy_update_unavailable`
 * (write) with no optimistic state change.
 */
export function AuthPolicySection() {
  const { hasPermission, isInitialized: permissionsReady } = usePermissions();
  const canManage = hasPermission(PERMISSIONS.ORG_MANAGE);

  const [mirror, setMirror] = useState<AuthPolicyMirror | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [readError, setReadError] = useState<string | null>(null);

  const [domainJitEnabled, setDomainJitEnabled] = useState(false);
  const [domainsInput, setDomainsInput] = useState("");
  const [ssoJitEnabled, setSsoJitEnabled] = useState(false);
  const [allowedMethods, setAllowedMethods] = useState<AllowedAuthMethod[]>([
    "magic_link",
  ]);
  const [isSaving, setIsSaving] = useState(false);
  const [writeError, setWriteError] = useState<string | null>(null);

  const loadPolicy = useCallback(async () => {
    setReadError(null);
    setIsLoading(true);
    try {
      const policy = await organizationRepository.getAuthPolicy();
      setMirror(policy);
      setDomainJitEnabled(policy.email_jit_provisioning === "DOMAIN_RESTRICTED");
      setDomainsInput((policy.email_allowed_domains ?? []).join(", "));
      setSsoJitEnabled(policy.sso_jit_provisioning === "CONNECTION_RESTRICTED");
      const effective =
        policy.auth_methods_restricted && policy.allowed_auth_methods?.length
          ? policy.allowed_auth_methods
          : PRIMARY_METHODS.filter(
              (m) => m !== "sso" || (policy.sso_active_connection_ids?.length ?? 0) > 0
            );
      setAllowedMethods(effective);
    } catch (err) {
      if (is503(err)) {
        setReadError(
          "El servicio de autenticación está temporalmente no disponible (503 — auth_policy_unavailable). No se pudo leer la política."
        );
      } else {
        setReadError(
          err instanceof Error
            ? err.message
            : "No se pudo leer la política de autenticación."
        );
      }
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    if (permissionsReady && canManage) {
      void loadPolicy();
    }
  }, [permissionsReady, canManage, loadPolicy]);

  const activeConnectionIDs = useMemo(
    () => mirror?.sso_active_connection_ids ?? [],
    [mirror]
  );
  const hasActiveSSO = activeConnectionIDs.length > 0;

  // `sso` is only toggleable when the org has active connections; otherwise it
  // stays off the list (the backend drops it for connection-less orgs).
  const toggleableMethods = useMemo(
    () =>
      PRIMARY_METHODS.filter(
        (m) => m !== "magic_link" && (m !== "sso" || hasActiveSSO)
      ),
    [hasActiveSSO]
  );

  const toggleMethod = (method: AllowedAuthMethod, checked: boolean) => {
    setAllowedMethods((prev) => {
      if (checked) {
        return prev.includes(method) ? prev : [...prev, method];
      }
      return prev.filter((m) => m !== method);
    });
  };

  const handleSave = async () => {
    setWriteError(null);
    setIsSaving(true);
    try {
      const parsedDomains = domainsInput
        .split(",")
        .map((d) => d.trim().toLowerCase())
        .filter(Boolean);

      const methods: AllowedAuthMethod[] = allowedMethods.includes("magic_link")
        ? allowedMethods
        : ["magic_link" as AllowedAuthMethod, ...allowedMethods];

      await organizationRepository.updateAuthPolicy({
        email_jit_provisioning: domainJitEnabled ? "DOMAIN_RESTRICTED" : "DISABLED",
        email_allowed_domains: parsedDomains,
        allowed_auth_methods: methods,
        sso_jit_provisioning:
          ssoJitEnabled && hasActiveSSO ? "CONNECTION_RESTRICTED" : "DISABLED",
        sso_jit_provisioning_allowed_connections:
          ssoJitEnabled && hasActiveSSO ? activeConnectionIDs : [],
        sso_default_connection_id: "",
      });
      toast.success("Política de autenticación actualizada");
      await loadPolicy();
    } catch (err) {
      if (is503(err)) {
        setWriteError(
          "El servicio de autenticación está temporalmente no disponible (503 — auth_policy_update_unavailable). Tu política no fue cambiada."
        );
      } else {
        setWriteError(
          err instanceof Error ? err.message : "No se pudo guardar la política."
        );
      }
    } finally {
      setIsSaving(false);
    }
  };

  if (permissionsReady && !canManage) {
    return (
      <Alert variant="destructive" className="border border-red-200 bg-red-50">
        <AlertTitle>Acceso restringido</AlertTitle>
        <AlertDescription>
          No tienes permisos para gestionar la política de autenticación.
        </AlertDescription>
      </Alert>
    );
  }

  return (
    <Card data-testid="auth-policy-section">
      <CardHeader>
        <div className="flex items-center gap-3">
          <ShieldCheck className="h-6 w-6 text-slate-600" />
          <div>
            <CardTitle>Acceso e identidad</CardTitle>
            <CardDescription>
              {ui.authPolicy.description}
            </CardDescription>
          </div>
        </div>
      </CardHeader>
      <CardContent className="space-y-6">
        {isLoading ? (
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
            Cargando política…
          </div>
        ) : readError ? (
          <Alert variant="destructive" className="border border-red-200 bg-red-50">
            <AlertTitle>No se pudo leer la política</AlertTitle>
            <AlertDescription className="space-y-3">
              {readError}
              <div>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => void loadPolicy()}
                >
                  Reintentar
                </Button>
              </div>
            </AlertDescription>
          </Alert>
        ) : (
          <>
            {/* Domain-restricted email JIT */}
            <div className="flex flex-col gap-3 rounded-lg border border-slate-200 p-4">
              <div className="flex items-start justify-between gap-4">
                <div className="space-y-1">
                  <Label htmlFor="domain-jit" className="text-sm font-medium">
                    Unión automática por dominio
                  </Label>
                  <p className="text-xs text-slate-500">
                    {ui.authPolicy.domainJitBody}
                  </p>
                </div>
                <Switch
                  id="domain-jit"
                  checked={domainJitEnabled}
                  onCheckedChange={setDomainJitEnabled}
                  aria-label="Permitir que compañeros con el dominio se unan automáticamente"
                />
              </div>
              {domainJitEnabled ? (
                <div className="space-y-2">
                  <Label htmlFor="allowed-domains" className="text-xs font-medium">
                    Dominios permitidos (separados por coma)
                  </Label>
                  <Input
                    id="allowed-domains"
                    value={domainsInput}
                    onChange={(e) => setDomainsInput(e.target.value)}
                    placeholder="ej. tuempresa.com"
                    className="w-full border-slate-200"
                  />
                  <p className="text-xs text-slate-500">
                    {ui.authPolicy.domainJitOauthImplication}
                  </p>
                </div>
              ) : null}
            </div>

            {/* SSO-JIT (only when the org has active SSO connections) */}
            {hasActiveSSO ? (
              <div className="flex items-start justify-between gap-4 rounded-lg border border-slate-200 p-4">
                <div className="space-y-1">
                  <Label htmlFor="sso-jit" className="text-sm font-medium">
                    SSO JIT (aprovisionamiento automático vía IdP)
                  </Label>
                  <p className="text-xs text-slate-500">
                    {ui.authPolicy.ssoJitBody}
                  </p>
                </div>
                <Switch
                  id="sso-jit"
                  checked={ssoJitEnabled}
                  onCheckedChange={setSsoJitEnabled}
                  aria-label="Permitir aprovisionamiento automático por SSO"
                />
              </div>
            ) : null}

            {/* Allowed primary methods (enforced-list mode) */}
            <div className="space-y-3 rounded-lg border border-slate-200 p-4">
              <div className="space-y-1">
                <Label className="text-sm font-medium">Métodos de acceso permitidos</Label>
                <p className="text-xs text-slate-500">
                  {ui.authPolicy.methodsBody}
                </p>
              </div>
              <div className="grid gap-2 sm:grid-cols-2">
                <label
                  key="magic_link"
                  className="flex items-center gap-2 text-sm text-slate-700"
                >
                  <Checkbox checked disabled aria-label="magic_link (siempre permitido)" />
                  {METHOD_LABELS.magic_link}
                  <span className="text-xs text-slate-400">(siempre)</span>
                </label>
                {toggleableMethods.map((method) => (
                  <label
                    key={method}
                    className="flex items-center gap-2 text-sm text-slate-700"
                  >
                    <Checkbox
                      checked={allowedMethods.includes(method)}
                      onCheckedChange={(checked) =>
                        toggleMethod(method, checked === true)
                      }
                      aria-label={method}
                    />
                    {METHOD_LABELS[method]}
                  </label>
                ))}
              </div>
            </div>

            {writeError ? (
              <p
                className="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-xs font-medium text-red-700"
                role="alert"
              >
                {writeError}
              </p>
            ) : null}

            <div className="flex items-center gap-3">
              <Button
                onClick={() => void handleSave()}
                disabled={isSaving}
                className="bg-emerald-500 text-white hover:bg-emerald-600"
              >
                {isSaving ? (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" aria-hidden="true" />
                ) : null}
                Guardar política
              </Button>
              <p className="text-xs text-slate-500">
                Requiere permiso org:manage. El cambio aplica desde el próximo
                inicio de sesión.
              </p>
            </div>
          </>
        )}
      </CardContent>
    </Card>
  );
}
