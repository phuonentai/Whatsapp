"use client";

import { useState } from "react";
import { Loader2, ShieldCheck } from "lucide-react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { organizationRepository } from "@/lib/api/api/repositories/organization-repository";
import type { UserProfile } from "@/lib/models/member.model";

type MfaPolicy = "OPTIONAL" | "REQUIRED_FOR_ALL";
type MfaMethods = "ALL_ALLOWED" | "RESTRICTED";

interface MfaPolicySectionProps {
  profile: UserProfile;
}

/**
 * Compliance → Security: organization MFA policy management.
 *
 * Reads the current policy from the profile — a display-only mirror that is
 * NEVER used for authorization decisions (design verdict #5 / spec: policy
 * mirror is display-only); Stytch enforces MFA at session mint and is the sole
 * enforcement point. Updates are persisted to Stytch via the Go backend
 * (PUT /api/organizations/mfa-policy, org:manage gated, circuit-breaker
 * protected → 503 structured error shown on backend failure).
 *
 * This section is rendered inside the compliance view, which is itself gated
 * by `org:manage` in settings-content; the write path is additionally guarded
 * here so the component can never be driven without the permission.
 */
export function MfaPolicySection({ profile }: MfaPolicySectionProps) {
  // Display-only mirror of the Stytch org policy.
  const mirrorPolicy: MfaPolicy =
    profile.organizationMfaPolicy === "REQUIRED_FOR_ALL"
      ? "REQUIRED_FOR_ALL"
      : "OPTIONAL";
  const mirrorMethods: MfaMethods =
    profile.organizationMfaMethods === "ALL_ALLOWED"
      ? "ALL_ALLOWED"
      : "RESTRICTED";

  const [policy, setPolicy] = useState<MfaPolicy>(mirrorPolicy);
  const [methods, setMethods] = useState<MfaMethods>(mirrorMethods);
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const allowedMethods =
    profile.organizationAllowedMfaMethods?.length
      ? profile.organizationAllowedMfaMethods
      : ["totp"];

  const handleSave = async () => {
    setError(null);
    setIsSaving(true);
    try {
      await organizationRepository.updateMfaPolicy({
        mfa_policy: policy,
        mfa_methods: methods,
        // This change only ships TOTP as a factor (non-goal: SMS OTP);
        // restricted policies always allow exactly totp.
        allowed_mfa_methods: ["totp"],
      });
      toast.success("Política MFA actualizada");
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "No se pudo actualizar la política MFA";
      // The Go backend answers 503 with a structured error when Stytch is
      // unreachable or the circuit breaker is open; surface it distinctly.
      if (err instanceof Error && message.includes("503")) {
        setError(
          "El servicio de MFA está temporalmente no disponible. Tu política no fue cambiada (503)."
        );
      } else {
        setError(message);
      }
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center gap-3">
          <ShieldCheck className="h-6 w-6 text-slate-600" />
          <div>
            <CardTitle>Política MFA de la organización</CardTitle>
            <CardDescription>
              Exige verificación en dos pasos (app de autenticación) al iniciar
              sesión. La política se aplica en Stytch al momento de crear la
              sesión; este panel solo la refleja y actualiza.
            </CardDescription>
          </div>
        </div>
      </CardHeader>
      <CardContent className="space-y-6">
        <div className="grid gap-6 sm:grid-cols-2">
          <div className="space-y-2">
            <Label className="text-sm font-medium">Requisito</Label>
            <div className="flex gap-2">
              <Button
                type="button"
                variant={policy === "OPTIONAL" ? "default" : "outline"}
                size="sm"
                onClick={() => setPolicy("OPTIONAL")}
                className={
                  policy === "OPTIONAL"
                    ? "bg-slate-900 text-white hover:bg-slate-800"
                    : ""
                }
              >
                Opcional
              </Button>
              <Button
                type="button"
                variant={policy === "REQUIRED_FOR_ALL" ? "default" : "outline"}
                size="sm"
                onClick={() => setPolicy("REQUIRED_FOR_ALL")}
                className={
                  policy === "REQUIRED_FOR_ALL"
                    ? "bg-slate-900 text-white hover:bg-slate-800"
                    : ""
                }
              >
                Obligatorio para todos
              </Button>
            </div>
            <p className="text-xs text-slate-500">
              Opcional: solo los miembros que se inscriban en MFA deberán pasar
              el segundo factor. Obligatorio: todos los miembros deben
              completar MFA en cada inicio de sesión.
            </p>
          </div>

          <div className="space-y-2">
            <Label className="text-sm font-medium">Métodos permitidos</Label>
            <div className="flex gap-2">
              <Button
                type="button"
                variant={methods === "RESTRICTED" ? "default" : "outline"}
                size="sm"
                onClick={() => setMethods("RESTRICTED")}
                className={
                  methods === "RESTRICTED"
                    ? "bg-slate-900 text-white hover:bg-slate-800"
                    : ""
                }
              >
                Solo TOTP (app de autenticación)
              </Button>
            </div>
            <p className="text-xs text-slate-500">
              Métodos activos:{" "}
              <span className="font-mono">{allowedMethods.join(", ")}</span>.
              SMS OTP no está habilitado en este cambio.
            </p>
          </div>
        </div>

        {error ? (
          <p
            className="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-xs font-medium text-red-700"
            role="alert"
          >
            {error}
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
      </CardContent>
    </Card>
  );
}
