"use client";

import { useState } from "react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Skeleton } from "@/components/ui/skeleton";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { Loader2, ServerCog } from "lucide-react";
import { useAdminConnectionsQuery } from "@/lib/hooks/queries/use-siigo-queries";
import { useAdminProvision } from "@/lib/hooks/mutations/use-siigo-mutations";
import type { AdminConnectionRow, SiigoConnectionStatus } from "@/lib/models/siigo-connection.model";

const STATUS_LABELS: Record<SiigoConnectionStatus, string> = {
  none: "Sin conexión",
  awaiting_setup: "Esperando configuración",
  connected: "Conectado",
  numeracion_ok: "Numeración confirmada",
  sandbox_ok: "Sandbox probado",
  live: "Activo",
  paused: "Pausado",
  invoicing_disabled: "Desactivado",
};

function StatusBadge({ status }: { status: SiigoConnectionStatus }) {
  const tone =
    status === "live"
      ? "bg-emerald-100 text-emerald-700"
      : status === "paused"
        ? "bg-gray-100 text-gray-600"
        : status === "awaiting_setup"
          ? "bg-amber-100 text-amber-700"
          : "bg-blue-100 text-blue-700";
  return <Badge variant="secondary" className={tone}>{STATUS_LABELS[status]}</Badge>;
}

function ProvisionForm({ row }: { row: AdminConnectionRow }) {
  const provision = useAdminProvision();
  const [clientId, setClientId] = useState("");
  const [clientSecret, setClientSecret] = useState("");
  const [nit, setNit] = useState("");

  const canSubmit = clientId.trim().length > 0 && clientSecret.trim().length > 0 && nit.trim().length > 0;

  return (
    <div className="space-y-2 rounded-lg border border-amber-200 bg-amber-50 p-3">
      <p className="text-xs font-medium text-amber-800">
        Provisionar credenciales para la organización {row.organization_id}
      </p>
      <div className="grid gap-2 sm:grid-cols-3">
        <Input placeholder="client_id" value={clientId} onChange={(e) => setClientId(e.target.value)} />
        <Input placeholder="client_secret" type="password" value={clientSecret} onChange={(e) => setClientSecret(e.target.value)} />
        <Input placeholder="NIT" value={nit} onChange={(e) => setNit(e.target.value)} />
      </div>
      {provision.error && <p className="text-xs text-red-600">{provision.error.message}</p>}
      <Button
        size="sm"
        onClick={() =>
          provision.mutate({
            organization_id: row.organization_id,
            client_id: clientId.trim(),
            client_secret: clientSecret.trim(),
            nit: nit.trim(),
          })
        }
        disabled={!canSubmit || provision.isPending}
      >
        {provision.isPending && <Loader2 className="mr-1 h-3 w-3 animate-spin" />}
        Provisionar
      </Button>
    </div>
  );
}

export function SiigoAdminView() {
  const { data: rows, isLoading, error, refetch } = useAdminConnectionsQuery();

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center gap-3">
          <ServerCog className="h-6 w-6 text-gray-600" />
          <div>
            <CardTitle>Onboarding de facturación Siigo</CardTitle>
            <CardDescription>Estado de conexión, numeración e importación por organización.</CardDescription>
          </div>
        </div>
      </CardHeader>
      <CardContent>
        {isLoading && <Skeleton className="h-40 w-full rounded-xl" />}

        {error && (
          <Alert variant="destructive" className="border border-red-200 bg-red-50">
            <AlertTitle>No se pudo cargar el listado</AlertTitle>
            <AlertDescription>{error.message}</AlertDescription>
            <Button variant="outline" size="sm" className="mt-3" onClick={() => refetch()}>
              Reintentar
            </Button>
          </Alert>
        )}

        {rows && rows.length === 0 && (
          <p className="py-8 text-center text-sm text-gray-500">
            Ninguna organización ha conectado Siigo todavía.
          </p>
        )}

        {rows && rows.length > 0 && (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Organización</TableHead>
                <TableHead>Estado</TableHead>
                <TableHead>NIT</TableHead>
                <TableHead>Numeración</TableHead>
                <TableHead>Última importación</TableHead>
                <TableHead>Error</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.map((row) => (
                <TableRow key={row.organization_id}>
                  <TableCell className="font-medium">{row.organization_id}</TableCell>
                  <TableCell><StatusBadge status={row.status} /></TableCell>
                  <TableCell>{row.nit ?? "—"}</TableCell>
                  <TableCell>
                    {row.numeration
                      ? `${row.numeration.prefijo ?? ""} ${row.numeration.next_number ?? ""}`.trim() || row.numeration.mode
                      : "—"}
                  </TableCell>
                  <TableCell>
                    {row.last_import_run
                      ? `${row.last_import_run.kind} · ${row.last_import_run.counts?.nuevos ?? 0} nuevos`
                      : "—"}
                  </TableCell>
                  <TableCell className="max-w-[200px] truncate text-xs text-red-600">
                    {row.last_error ?? "—"}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}

        {rows && rows.some((row) => row.status === "awaiting_setup") && (
          <div className="mt-5 space-y-3">
            <h4 className="text-sm font-semibold text-gray-900">Configuración asistida</h4>
            {rows
              .filter((row) => row.status === "awaiting_setup")
              .map((row) => (
                <ProvisionForm key={row.organization_id} row={row} />
              ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
