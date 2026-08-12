"use client";

import { useState } from "react";
import {
  KeyRound,
  Loader2,
  ShieldCheck,
  Smartphone,
} from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  createTotp,
  rotateRecoveryCodes,
  verifyTotpEnrollment,
} from "@/lib/actions/auth/mfa";

/**
 * Settings → Profile → Security section (TOTP authenticator-app enrollment).
 *
 * Flow: "Set up authenticator app" → createTotp() (duplicate-guarded) →
 * render the Stytch-issued `qr_code` image (server-generated — NO QR library)
 * + manual secret → verify one code (verifyTotpEnrollment) → show the
 * one-time recovery codes exactly once ("I saved these" confirmation).
 *
 * A member who already holds a TOTP registration (`totp_registration_id`) is
 * surfaced for management (rotate recovery codes) instead of creating a
 * duplicate instance (spec: duplicate guard; Stytch multi-registration
 * semantics are guarded here).
 *
 * SSOT: the qr_code, secret, and recovery codes are never persisted locally —
 * they exist only in Stytch and in this transient client state, which is
 * cleared when the user leaves the section.
 */
export function SecuritySection() {
  // idle → creating → qr (show qr+secret, verify) → recovery (codes shown
  // once) → enrolled | existing (manage registration).
  const [phase, setPhase] = useState<
    "idle" | "creating" | "qr" | "recovery" | "enrolled" | "existing"
  >("idle");
  const [error, setError] = useState<string | null>(null);
  const [qrCode, setQrCode] = useState<string | null>(null);
  const [secret, setSecret] = useState<string | null>(null);
  const [recoveryCodes, setRecoveryCodes] = useState<string[]>([]);
  const [existingRegistrationId, setExistingRegistrationId] = useState<
    string | null
  >(null);

  const [verifyCode, setVerifyCode] = useState("");
  const [isVerifying, setIsVerifying] = useState(false);
  const [isRotating, setIsRotating] = useState(false);

  const startEnrollment = async () => {
    setError(null);
    setPhase("creating");
    try {
      const result = await createTotp();
      if (!result.success) {
        setError(result.error || "No se pudo iniciar la configuración.");
        setPhase("idle");
        return;
      }

      if (result.data.status === "existing") {
        setExistingRegistrationId(result.data.totpRegistrationId);
        setPhase("existing");
        return;
      }

      setQrCode(result.data.qrCode ?? null);
      setSecret(result.data.secret ?? null);
      setRecoveryCodes(result.data.recoveryCodes ?? []);
      setPhase("qr");
    } catch {
      setError("No se pudo iniciar la configuración. Inténtalo de nuevo.");
      setPhase("idle");
    }
  };

  const submitVerification = async () => {
    const code = verifyCode.trim();
    if (code.length < 6) {
      setError("Ingresa el código de 6 dígitos de tu app de autenticación.");
      return;
    }
    setError(null);
    setIsVerifying(true);
    try {
      const result = await verifyTotpEnrollment({ code });
      if (!result.success) {
        setError(
          "El código no es válido. Verifica la hora de tu dispositivo e inténtalo de nuevo."
        );
        return;
      }
      // Recovery codes were returned by createTotp; show them exactly once.
      setPhase("recovery");
    } catch {
      setError("No se pudo verificar el código. Inténtalo de nuevo.");
    } finally {
      setIsVerifying(false);
    }
  };

  const confirmRecoveryCodes = () => {
    setRecoveryCodes([]);
    setVerifyCode("");
    setPhase("enrolled");
  };

  const rotateCodes = async () => {
    setError(null);
    setIsRotating(true);
    try {
      const result = await rotateRecoveryCodes();
      if (!result.success) {
        setError(
          result.error || "No se pudieron rotar los códigos de recuperación."
        );
        return;
      }
      setRecoveryCodes(result.data.recoveryCodes);
      setPhase("recovery");
    } catch {
      setError("No se pudieron rotar los códigos de recuperación.");
    } finally {
      setIsRotating(false);
    }
  };

  return (
    <section className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
      <header className="space-y-1">
        <div className="flex items-center gap-2">
          <ShieldCheck className="h-5 w-5 text-emerald-600" aria-hidden="true" />
          <p className="text-xs font-semibold uppercase tracking-[0.2em] text-slate-500">
            Security
          </p>
        </div>
        <h3 className="text-xl font-semibold text-slate-900">
          Two-step verification (authenticator app)
        </h3>
        <p className="text-sm text-slate-600">
          Añade una app de autenticación (TOTP) como segundo factor. Los
          códigos se generan en tu dispositivo; el secreto se almacena
          únicamente en Stytch.
        </p>
      </header>

      <div className="mt-6 space-y-4">
        {phase === "idle" && (
          <Button
            onClick={() => void startEnrollment()}
            className="bg-emerald-500 text-white hover:bg-emerald-600"
          >
            <Smartphone className="mr-2 h-4 w-4" aria-hidden="true" />
            Configurar app de autenticación
          </Button>
        )}

        {phase === "creating" && (
          <p className="flex items-center gap-2 text-sm text-slate-600">
            <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
            Creando registro TOTP…
          </p>
        )}

        {phase === "existing" && (
          <div className="space-y-4">
            <div className="rounded-xl border border-blue-200 bg-blue-50 px-4 py-3">
              <p className="text-sm font-medium text-blue-900">
                Ya tienes una app de autenticación configurada.
              </p>
              <p className="mt-1 text-xs text-blue-700">
                Registro TOTP: <code>{existingRegistrationId}</code> — no se
                creó una instancia duplicada.
              </p>
            </div>
            <Button
              variant="outline"
              onClick={() => void rotateCodes()}
              disabled={isRotating}
            >
              {isRotating ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" aria-hidden="true" />
              ) : (
                <KeyRound className="mr-2 h-4 w-4" aria-hidden="true" />
              )}
              Rotar códigos de recuperación
            </Button>
          </div>
        )}

        {phase === "qr" && qrCode && (
          <div className="space-y-4">
            <div className="grid gap-4 sm:grid-cols-[auto_1fr] sm:items-center">
              {/* Server-generated QR image from Stytch — rendered as-is, no QR
                  library (design verdict #2). */}
              {/* eslint-disable-next-line @next/next/no-img-element */}
              <img
                src={qrCode}
                alt="Código QR para la app de autenticación"
                width={180}
                height={180}
                className="rounded-xl border border-slate-200"
              />
              <div className="space-y-1 text-sm">
                <p className="font-medium text-slate-900">
                  Escanea el código QR o ingresa el secreto manualmente
                </p>
                {secret ? (
                  <p className="font-mono text-xs text-slate-600">{secret}</p>
                ) : null}
                <p className="text-xs text-slate-500">
                  Usa Google Authenticator, 1Password, Authy o cualquier app
                  compatible con TOTP.
                </p>
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="totp-verify-code" className="text-sm font-medium">
                Código de verificación
              </Label>
              <div className="flex flex-col gap-2 sm:flex-row">
                <Input
                  id="totp-verify-code"
                  type="text"
                  inputMode="numeric"
                  maxLength={6}
                  placeholder="Código de 6 dígitos"
                  value={verifyCode}
                  onChange={(e) => setVerifyCode(e.target.value)}
                  disabled={isVerifying}
                  className="sm:max-w-[200px]"
                />
                <Button
                  onClick={() => void submitVerification()}
                  disabled={verifyCode.trim().length < 6 || isVerifying}
                  className="bg-emerald-500 text-white hover:bg-emerald-600"
                >
                  {isVerifying ? (
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" aria-hidden="true" />
                  ) : null}
                  Verificar e inscribir
                </Button>
              </div>
            </div>
          </div>
        )}

        {phase === "recovery" && recoveryCodes.length > 0 && (
          <div className="space-y-4">
            <div className="rounded-xl border border-amber-200 bg-amber-50 px-4 py-3">
              <p className="text-sm font-semibold text-amber-900">
                Guarda estos códigos de recuperación
              </p>
              <p className="mt-1 text-xs text-amber-800">
                Se muestran una sola vez. Úsalos si pierdes el acceso a tu app
                de autenticación. No se vuelven a mostrar.
              </p>
              <div className="mt-3 grid grid-cols-2 gap-2">
                {recoveryCodes.map((code) => (
                  <code
                    key={code}
                    className="rounded-lg bg-white px-2 py-1 text-center font-mono text-xs font-semibold text-amber-900"
                  >
                    {code}
                  </code>
                ))}
              </div>
            </div>
            <Button
              onClick={confirmRecoveryCodes}
              className="bg-emerald-500 text-white hover:bg-emerald-600"
            >
              Guardé estos códigos
            </Button>
          </div>
        )}

        {phase === "enrolled" && (
          <div className="rounded-xl border border-emerald-200 bg-emerald-50 px-4 py-3">
            <p className="text-sm font-medium text-emerald-900">
              Verificación en dos pasos activada. ✓
            </p>
            <p className="mt-1 text-xs text-emerald-700">
              A partir de ahora deberás ingresar un código de tu app de
              autenticación al iniciar sesión.
            </p>
          </div>
        )}

        {error ? (
          <p className="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-xs font-medium text-red-700">
            {error}
          </p>
        ) : null}
      </div>
    </section>
  );
}
