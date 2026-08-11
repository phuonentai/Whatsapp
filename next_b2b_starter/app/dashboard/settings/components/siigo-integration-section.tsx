"use client";

import { useState } from "react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Skeleton } from "@/components/ui/skeleton";
import {
  FileText,
  CheckCircle2,
  CircleDashed,
  Loader2,
  Lock,
  Pause,
  Play,
  RefreshCw,
  Receipt,
  Users,
  FlaskConical,
  Rocket,
} from "lucide-react";
import { useSiigoStatusQuery, useSiigoNumerationQuery, useImportPreviewQuery } from "@/lib/hooks/queries/use-siigo-queries";
import {
  useSiigoConnect,
  useRequestAssistedSetup,
  useConfirmNumeration,
  useImportConfirm,
  useTestInvoice,
  usePauseInvoicing,
  useResumeInvoicing,
  useActivateInvoicing,
} from "@/lib/hooks/mutations/use-siigo-mutations";
import type { ImportCounts, SiigoConnectionStatus } from "@/lib/models/siigo-connection.model";

const STEP_ORDER: { key: string; title: string; icon: typeof FileText; requiredStatus: SiigoConnectionStatus }[] = [
  { key: "conectar", title: "Conectar Siigo", icon: FileText, requiredStatus: "connected" },
  { key: "numeracion", title: "Numeración", icon: Receipt, requiredStatus: "numeracion_ok" },
  { key: "importar", title: "Importar clientes", icon: Users, requiredStatus: "sandbox_ok" },
  { key: "prueba", title: "Prueba en sandbox", icon: FlaskConical, requiredStatus: "sandbox_ok" },
  { key: "activar", title: "Activar", icon: Rocket, requiredStatus: "live" },
];

function statusIndex(status: SiigoConnectionStatus): number {
  switch (status) {
    case "none":
    case "awaiting_setup":
    case "invoicing_disabled":
      return 0;
    case "connected":
      return 1;
    case "numeracion_ok":
      return 2;
    case "sandbox_ok":
      return 3;
    case "live":
    case "paused":
      return STEP_ORDER.length;
  }
}

export function SiigoIntegrationSection() {
  const { data: status, isLoading, error, refetch } = useSiigoStatusQuery();

  if (isLoading) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-8 w-56" />
        <Skeleton className="h-64 rounded-xl" />
      </div>
    );
  }

  if (error || !status) {
    return (
      <Alert variant="destructive" className="border border-red-200 bg-red-50">
        <AlertTitle>No se pudo cargar la integración Siigo</AlertTitle>
        <AlertDescription>{error?.message ?? "Error al consultar el estado de la conexión."}</AlertDescription>
        <Button variant="outline" size="sm" className="mt-3" onClick={() => refetch()}>
          Reintentar
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
              <FileText className="h-6 w-6 text-gray-600" />
              <div>
                <CardTitle>Integración Siigo</CardTitle>
                <CardDescription>
                  Conecta tu Siigo para facturar electrónicamente desde WhatsApp.
                </CardDescription>
              </div>
            </div>
            <KillSwitch status={status.status} />
          </div>
        </CardHeader>
        <CardContent className="space-y-5">
          <StatusBanner status={status.status} />
          {status.status === "none" && <ConnectStep />}
          {status.status === "awaiting_setup" && <AwaitingSetupNotice />}
          {status.status === "connected" && <NumerationStep />}
          {status.status === "numeracion_ok" && <ImportStep />}
          {(status.status === "numeracion_ok" || status.status === "sandbox_ok") && (
            <SandboxAndActivateStep />
          )}
          {status.status === "paused" && (
            <Alert className="border border-gray-200 bg-gray-50">
              <AlertTitle>Facturación pausada</AlertTitle>
              <AlertDescription>
                No se emitirán facturas automáticas hasta que reanudes la facturación.
              </AlertDescription>
            </Alert>
          )}
          {status.status === "live" && <ActiveNotice />}
          <WizardProgress status={status.status} />
        </CardContent>
      </Card>
    </div>
  );
}

function StatusBanner({ status }: { status: SiigoConnectionStatus }) {
  switch (status) {
    case "none":
      return (
        <Alert className="border border-blue-200 bg-blue-50">
          <AlertTitle>Conecta Siigo para facturar</AlertTitle>
          <AlertDescription>
            Cuando un negocio llegue a la etapa facturado, la plataforma emitirá la factura
            electrónica automáticamente.
          </AlertDescription>
        </Alert>
      );
    case "awaiting_setup":
      return (
        <Alert className="border border-amber-200 bg-amber-50">
          <AlertTitle>Tu equipo está configurando tu facturación</AlertTitle>
          <AlertDescription>
            Recibimos tu solicitud. Un administrador configurará tu conexión Siigo.
          </AlertDescription>
        </Alert>
      );
    case "invoicing_disabled":
      return <DisabledNotice />;
    default:
      return null;
  }
}

function KillSwitch({ status }: { status: SiigoConnectionStatus }) {
  const pause = usePauseInvoicing();
  const resume = useResumeInvoicing();

  if (status === "live") {
    return (
      <Button variant="outline" size="sm" onClick={() => pause.mutate()} disabled={pause.isPending}>
        {pause.isPending ? <Loader2 className="mr-1 h-4 w-4 animate-spin" /> : <Pause className="mr-1 h-4 w-4" />}
        Pausar
      </Button>
    );
  }
  if (status === "paused") {
    return (
      <Button variant="outline" size="sm" onClick={() => resume.mutate()} disabled={resume.isPending}>
        {resume.isPending ? <Loader2 className="mr-1 h-4 w-4 animate-spin" /> : <Play className="mr-1 h-4 w-4" />}
        Reanudar
      </Button>
    );
  }
  return null;
}

function ConnectStep() {
  const connect = useSiigoConnect();
  const assisted = useRequestAssistedSetup();
  const [clientId, setClientId] = useState("");
  const [clientSecret, setClientSecret] = useState("");
  const [nit, setNit] = useState("");

  const canSubmit = clientId.trim().length > 0 && clientSecret.trim().length > 0 && nit.trim().length > 0;

  return (
    <div className="space-y-4 rounded-xl border border-gray-200 bg-gray-50 p-5">
      <div>
        <h3 className="text-base font-semibold text-gray-900">Paso 1 — Conecta tu cuenta Siigo</h3>
        <p className="mt-1 text-sm text-gray-500">
          Ingresa las credenciales API de tu empresa en Siigo (Portal de desarrolladores) y el NIT de tu empresa.
        </p>
      </div>

      {connect.error && (
        <Alert variant="destructive" className="border border-red-200 bg-red-50">
          <AlertDescription>{connect.error.message}</AlertDescription>
        </Alert>
      )}

      <div className="grid gap-4 sm:grid-cols-3">
        <div className="space-y-2">
          <Label htmlFor="siigo-client-id">Client ID</Label>
          <Input
            id="siigo-client-id"
            type="text"
            placeholder="client_id"
            value={clientId}
            onChange={(e) => setClientId(e.target.value)}
            disabled={connect.isPending}
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="siigo-client-secret">Client Secret</Label>
          <Input
            id="siigo-client-secret"
            type="password"
            placeholder="client_secret"
            value={clientSecret}
            onChange={(e) => setClientSecret(e.target.value)}
            disabled={connect.isPending}
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="siigo-nit">NIT de tu empresa</Label>
          <Input
            id="siigo-nit"
            type="text"
            placeholder="900.123.456-7"
            value={nit}
            onChange={(e) => setNit(e.target.value)}
            disabled={connect.isPending}
          />
        </div>
      </div>

      <div className="flex flex-wrap items-center gap-3">
        <Button
          onClick={() => connect.mutate({ client_id: clientId.trim(), client_secret: clientSecret.trim(), nit: nit.trim() })}
          disabled={!canSubmit || connect.isPending}
          className="bg-emerald-600 text-white hover:bg-emerald-700"
        >
          {connect.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
          Conectar Siigo
        </Button>
        <Button variant="ghost" size="sm" onClick={() => assisted.mutate()} disabled={assisted.isPending}>
          {assisted.isPending && <Loader2 className="mr-1 h-4 w-4 animate-spin" />}
          No tengo Siigo — solicitar configuración asistida
        </Button>
      </div>
    </div>
  );
}

function AwaitingSetupNotice() {
  return (
    <Alert className="border border-amber-200 bg-amber-50">
      <AlertTitle className="flex items-center gap-2">
        <RefreshCw className="h-4 w-4" />
        Configuración en curso
      </AlertTitle>
      <AlertDescription>
        Un administrador está provisionando tus credenciales Siigo. Esta página se actualizará automáticamente.
      </AlertDescription>
    </Alert>
  );
}

function NumerationStep() {
  const numeration = useSiigoNumerationQuery({ enabled: true });
  const confirm = useConfirmNumeration();

  return (
    <div className="space-y-4 rounded-xl border border-gray-200 bg-gray-50 p-5">
      <div>
        <h3 className="text-base font-semibold text-gray-900">Paso 2 — Confirma tu numeración DIAN</h3>
        <p className="mt-1 text-sm text-gray-500">
          La facturación continúa la numeración de tu empresa. Confirma antes de emitir la primera factura.
        </p>
      </div>

      {numeration.isLoading && <Skeleton className="h-10 w-64" />}

      {numeration.data && (
        <div className="rounded-lg border border-gray-200 bg-white p-4 text-sm">
          {numeration.data.mode === "auto" && (
            <p className="text-gray-700">
              Siigo asignará el consecutivo automáticamente bajo la resolución activa de tu empresa.
              <span className="mt-1 block text-xs text-gray-500">
                {numeration.data.prefijo && <>Prefijo: {numeration.data.prefijo} · </>}
                {numeration.data.next_number && <>Siguiente número: {numeration.data.next_number}</>}
              </span>
            </p>
          )}
          {numeration.data.mode === "manual" && (
            <p className="text-gray-700">
              Siguiente número: <strong>{numeration.data.next_number ?? "—"}</strong>
              {numeration.data.prefijo && <> · Prefijo: {numeration.data.prefijo}</>}
            </p>
          )}
        </div>
      )}

      {confirm.error && (
        <Alert variant="destructive" className="border border-red-200 bg-red-50">
          <AlertDescription>{confirm.error.message}</AlertDescription>
        </Alert>
      )}

      <Button onClick={() => confirm.mutate()} disabled={confirm.isPending || numeration.isLoading}>
        {confirm.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
        Confirmar numeración
      </Button>
    </div>
  );
}

function ImportStep() {
  const preview = useImportPreviewQuery();
  const confirm = useImportConfirm();
  const [previewResult, setPreviewResult] = useState<ImportCounts | null>(null);
  const [result, setResult] = useState<string | null>(null);

  const handlePreview = async () => {
    setResult(null);
    const res = await preview.refetch();
    if (res.data) {
      setPreviewResult(res.data);
    }
  };

  const handleConfirm = async () => {
    try {
      const res = await confirm.mutateAsync();
      if (res) {
        setResult(
          `${res.nuevos} nuevos · ${res.existentes} existentes · ${res.duplicados} duplicados · ${res.sin_nit} sin NIT`,
        );
      }
    } catch {
      // error handled by mutation onError
    }
  };

  const counts = previewResult ?? preview.data;

  return (
    <div className="space-y-4 rounded-xl border border-gray-200 bg-gray-50 p-5">
      <div>
        <h3 className="text-base font-semibold text-gray-900">Paso 3 — Importa tus clientes</h3>
        <p className="mt-1 text-sm text-gray-500">
          Importa los clientes de Siigo como empresas y contactos del CRM (dedupe por NIT).
        </p>
      </div>

      {!counts && !preview.isFetching && (
        <Button variant="outline" onClick={handlePreview}>
          Ver vista previa
        </Button>
      )}

      {preview.isFetching && (
        <div className="flex items-center gap-2 text-sm text-gray-500">
          <Loader2 className="h-4 w-4 animate-spin" /> Consultando clientes en Siigo…
        </div>
      )}

      {counts && (
        <div className="rounded-lg border border-gray-200 bg-white p-4 text-sm">
          <p className="font-medium text-gray-900">{counts.total} clientes encontrados</p>
          <ul className="mt-2 grid gap-1 text-gray-600 sm:grid-cols-2">
            <li>Nuevos: {counts.nuevos}</li>
            <li>Existentes: {counts.existentes}</li>
            <li>Duplicados por NIT: {counts.duplicados}</li>
            <li>Sin NIT: {counts.sin_nit}</li>
          </ul>
          {result && <p className="mt-2 text-xs text-emerald-700">Importación completada: {result}</p>}
          {confirm.error && (
            <p className="mt-2 text-xs text-red-600">{confirm.error.message}</p>
          )}
          <Button className="mt-3" onClick={handleConfirm} disabled={confirm.isPending}>
            {confirm.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            Confirmar importación
          </Button>
        </div>
      )}
    </div>
  );
}

function SandboxAndActivateStep() {
  const test = useTestInvoice();
  const activate = useActivateInvoicing();
  const [testResult, setTestResult] = useState<string | null>(null);

  const handleTest = async () => {
    try {
      const res = await test.mutateAsync();
      if (res) {
        setTestResult(`Factura ${res.invoice_id ?? ""} — estado: ${res.status}`);
      }
    } catch {
      // error handled by mutation onError
    }
  };

  return (
    <div className="space-y-4">
      <div className="rounded-xl border border-gray-200 bg-gray-50 p-5">
        <h3 className="text-base font-semibold text-gray-900">Paso 4 — Prueba en sandbox</h3>
        <p className="mt-1 text-sm text-gray-500">
          Crea una factura de prueba en el ambiente sandbox para validar el flujo completo.
        </p>
        <div className="mt-3 flex flex-wrap items-center gap-3">
          <Button variant="outline" onClick={handleTest} disabled={test.isPending}>
            {test.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            Crear factura de prueba
          </Button>
          {testResult && <span className="text-sm text-emerald-700">{testResult}</span>}
        </div>
        {test.error && (
          <p className="mt-2 text-sm text-red-600">{test.error.message}</p>
        )}
      </div>

      <div className="rounded-xl border border-gray-200 bg-gray-50 p-5">
        <h3 className="text-base font-semibold text-gray-900">Paso 5 — Activar facturación</h3>
        <p className="mt-1 text-sm text-gray-500">
          Al activar, los negocios en etapa facturado emitirán factura electrónica automáticamente.
          La plantilla de WhatsApp <code className="rounded bg-gray-100 px-1">factura_lista</code> debe estar
          aprobada en Meta para enviar la notificación.
        </p>
        <Button
          className="mt-3 bg-emerald-600 text-white hover:bg-emerald-700"
          onClick={() => activate.mutate()}
          disabled={activate.isPending}
        >
          {activate.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
          Activar facturación
        </Button>
        {activate.error && <p className="mt-2 text-sm text-red-600">{activate.error.message}</p>}
      </div>
    </div>
  );
}

function ActiveNotice() {
  return (
    <Alert className="border border-emerald-200 bg-emerald-50">
      <AlertTitle className="flex items-center gap-2">
        <CheckCircle2 className="h-4 w-4 text-emerald-600" />
        Facturación activa
      </AlertTitle>
      <AlertDescription>
        El flujo negocio → facturado → factura electrónica → notificación por WhatsApp está habilitado.
      </AlertDescription>
    </Alert>
  );
}

function DisabledNotice() {
  return (
    <Alert className="border border-gray-200 bg-gray-50">
      <AlertDescription>
        Facturación desactivada — activa con Siigo para emitir facturas electrónicas automáticas.
      </AlertDescription>
    </Alert>
  );
}

function WizardProgress({ status }: { status: SiigoConnectionStatus }) {
  const current = statusIndex(status);
  return (
    <div className="border-t border-gray-200 pt-4">
      <ol className="flex flex-wrap items-center gap-2">
        {STEP_ORDER.map((step, idx) => {
          const done = idx < current;
          const active = idx === current && status !== "live" && status !== "paused";
          const Icon = step.icon;
          return (
            <li key={step.key} className="flex items-center gap-2">
              <div
                className={`flex items-center gap-1.5 rounded-full border px-3 py-1 text-xs font-medium ${
                  done
                    ? "border-emerald-200 bg-emerald-50 text-emerald-700"
                    : active
                      ? "border-blue-200 bg-blue-50 text-blue-700"
                      : "border-gray-200 bg-gray-50 text-gray-400"
                }`}
              >
                {done ? <CheckCircle2 className="h-3.5 w-3.5" /> : active ? <CircleDashed className="h-3.5 w-3.5" /> : <Lock className="h-3.5 w-3.5" />}
                <Icon className="h-3.5 w-3.5" />
                {step.title}
              </div>
              {idx < STEP_ORDER.length - 1 && <span className="text-gray-300">→</span>}
            </li>
          );
        })}
      </ol>
    </div>
  );
}
