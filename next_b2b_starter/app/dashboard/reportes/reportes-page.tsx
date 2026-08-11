"use client";

import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  Bar,
  BarChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";

import { analyticsRepository } from "@/lib/api/api/repositories/analytics-repository";
import { queryKeys } from "@/lib/hooks/queries/query-keys";
import { useModule } from "@/lib/hooks/use-entitlement";
import { usePermissions } from "@/lib/hooks/use-permissions";
import { PERMISSIONS } from "@/lib/auth/permissions";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { ErrorState } from "@/components/common/error-state";

function formatCOP(value: number): string {
  return new Intl.NumberFormat("es-CO", {
    style: "currency",
    currency: "COP",
    maximumFractionDigits: 0,
  }).format(value);
}

function formatDate(periodo: string): string {
  const [y, m, d] = periodo.split("-");
  if (!y || !m || !d) return periodo;
  return `${d}/${m}/${y}`;
}

export function ReportesPage() {
  const [period, setPeriod] = useState<"week" | "month">("month");
  const [days, setDays] = useState(30);
  const { hasPermission } = usePermissions();
  const analyticsModule = useModule("analytics");

  const canViewInvoices = hasPermission(PERMISSIONS.INVOICE_VIEW);
  const canViewDeals = hasPermission("deal:view");
  const canViewContacts = hasPermission("contact:view");

  const revenueQuery = useQuery({
    queryKey: [...queryKeys.modules.all, "analytics", "revenue", period],
    queryFn: () => analyticsRepository.revenue({ period }),
    enabled: analyticsModule.enabled && canViewInvoices,
    staleTime: 60 * 1000,
  });

  const topCustomersQuery = useQuery({
    queryKey: [...queryKeys.modules.all, "analytics", "top-customers"],
    queryFn: () => analyticsRepository.topCustomers(10),
    enabled: analyticsModule.enabled && canViewInvoices,
    staleTime: 60 * 1000,
  });

  const funnelQuery = useQuery({
    queryKey: [...queryKeys.modules.all, "analytics", "funnel"],
    queryFn: () => analyticsRepository.funnel(),
    enabled: analyticsModule.enabled && canViewDeals,
    staleTime: 60 * 1000,
  });

  const inactiveQuery = useQuery({
    queryKey: [...queryKeys.modules.all, "analytics", "inactive", days],
    queryFn: () => analyticsRepository.inactiveContacts(days),
    enabled: analyticsModule.enabled && canViewContacts,
    staleTime: 60 * 1000,
  });

  const totalRevenue = useMemo(
    () => (revenueQuery.data ?? []).reduce((sum, p) => sum + p.monto_total, 0),
    [revenueQuery.data]
  );

  const funnelData = useMemo(() => {
    if (!funnelQuery.data) return [];
    const rows = funnelQuery.data.etapas.map((e) => ({
      name: e.etapa,
      cantidad: e.cantidad,
      monto: e.monto_total,
    }));
    if (funnelQuery.data.otras_pipelines) {
      rows.push({
        name: "Otras pipelines",
        cantidad: funnelQuery.data.otras_pipelines.cantidad,
        monto: funnelQuery.data.otras_pipelines.monto_total,
      });
    }
    return rows;
  }, [funnelQuery.data]);

  if (!analyticsModule.enabled) {
    return (
      <div className="p-6">
        <h1 className="text-2xl font-bold mb-4">Reportes</h1>
        <p className="text-gray-600">
          El módulo de Reportes no está disponible en tu plan actual.
        </p>
      </div>
    );
  }

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Reportes de ventas</h1>
          <p className="text-sm text-gray-500">
            Ventas facturadas (facturas válidas), clientes, embudo e inactividad.
          </p>
        </div>
        <div className="flex gap-2">
          {(["week", "month"] as const).map((p) => (
            <button
              key={p}
              onClick={() => setPeriod(p)}
              className={`px-4 py-2 rounded-md text-sm font-medium ${
                period === p ? "bg-blue-600 text-white" : "bg-gray-100 text-gray-700"
              }`}
            >
              {p === "week" ? "Semana" : "Mes"}
            </button>
          ))}
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <Card>
          <CardHeader>
            <CardTitle className="text-sm text-gray-500">Ventas facturadas</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-bold">{formatCOP(totalRevenue)}</p>
            <p className="text-xs text-gray-400">
              {period === "week" ? "Últimas semanas" : "Últimos meses"} (COP)
            </p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle className="text-sm text-gray-500">Top cliente</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-bold truncate">
              {topCustomersQuery.data?.[0]?.nombre ?? "—"}
            </p>
            <p className="text-xs text-gray-400">
              {topCustomersQuery.data?.[0]
                ? formatCOP(topCustomersQuery.data[0].monto_total)
                : "Sin facturas válidas"}
            </p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle className="text-sm text-gray-500">Negocios ganados</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-bold">
              {funnelQuery.data?.ganado?.cantidad ?? 0}
            </p>
            <p className="text-xs text-gray-400">
              {funnelQuery.data?.ganado
                ? formatCOP(funnelQuery.data.ganado.monto_total)
                : "Sin negocios ganados"}
            </p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle className="text-sm text-gray-500">Contactos inactivos</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-bold">
              {inactiveQuery.data?.filter((c) => c.clasificacion === "inactivo").length ??
                0}
            </p>
            <p className="text-xs text-gray-400">
              Sin mensajes en {days} días (WhatsApp)
            </p>
          </CardContent>
        </Card>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <Card>
          <CardHeader>
            <CardTitle>Ventas facturadas por {period === "week" ? "semana" : "mes"}</CardTitle>
          </CardHeader>
          <CardContent className="h-72">
            {canViewInvoices && revenueQuery.isError && (
              <ErrorState
                title="Error al cargar las ventas"
                description="No se pudieron cargar las ventas facturadas."
                onRetry={() => revenueQuery.refetch()}
                isRetrying={revenueQuery.isRefetching}
              />
            )}
            {canViewInvoices && revenueQuery.isLoading && (
              <p className="text-gray-500">Cargando...</p>
            )}
            {canViewInvoices &&
              !revenueQuery.isLoading &&
              (revenueQuery.data?.length ? (
                <ResponsiveContainer width="100%" height="100%">
                  <BarChart
                    data={revenueQuery.data.map((p) => ({
                      name: formatDate(p.periodo),
                      monto: p.monto_total,
                    }))}
                  >
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis dataKey="name" />
                    <YAxis tickFormatter={(v: number) => `${Math.round(v / 1000000)}M`} />
                    <Tooltip formatter={(v) => formatCOP(Number(v))} />
                    <Bar dataKey="monto" fill="#2563eb" name="Facturado (COP)" />
                  </BarChart>
                </ResponsiveContainer>
              ) : (
                <p className="text-gray-500">
                  Sin datos para el periodo seleccionado.
                </p>
              ))}
            {!canViewInvoices && (
              <p className="text-gray-500">No tienes permiso para ver facturación.</p>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Embudo de negocios</CardTitle>
          </CardHeader>
          <CardContent className="h-72 overflow-auto">
            {canViewDeals && funnelQuery.isError && (
              <ErrorState
                title="Error al cargar el embudo"
                description="No se pudo cargar el embudo de negocios."
                onRetry={() => funnelQuery.refetch()}
                isRetrying={funnelQuery.isRefetching}
              />
            )}
            {canViewDeals && funnelQuery.isLoading && (
              <p className="text-gray-500">Cargando...</p>
            )}
            {canViewDeals && !funnelQuery.isLoading && (
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b text-left text-gray-500">
                    <th className="py-2">Etapa</th>
                    <th className="py-2 text-right">Negocios</th>
                    <th className="py-2 text-right">Monto (COP)</th>
                  </tr>
                </thead>
                <tbody>
                  {funnelData.length === 0 && (
                    <tr>
                      <td colSpan={3} className="py-4 text-gray-500">
                        Sin datos de pipeline.
                      </td>
                    </tr>
                  )}
                  {funnelData.map((row) => (
                    <tr key={row.name} className="border-b">
                      <td className="py-2">{row.name}</td>
                      <td className="py-2 text-right">{row.cantidad}</td>
                      <td className="py-2 text-right">{formatCOP(row.monto)}</td>
                    </tr>
                  ))}
                  {funnelQuery.data?.ganado && (
                    <tr className="bg-green-50">
                      <td className="py-2 font-medium">Ganado</td>
                      <td className="py-2 text-right font-medium">
                        {funnelQuery.data.ganado.cantidad}
                      </td>
                      <td className="py-2 text-right font-medium">
                        {formatCOP(funnelQuery.data.ganado.monto_total)}
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            )}
            {!canViewDeals && (
              <p className="text-gray-500">No tienes permiso para ver negocios.</p>
            )}
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle>Contactos inactivos (WhatsApp)</CardTitle>
          <select
            value={days}
            onChange={(e) => setDays(Number(e.target.value))}
            className="rounded-md border px-3 py-1.5 text-sm"
          >
            {[7, 15, 30, 60, 90].map((d) => (
              <option key={d} value={d}>
                {d} días
              </option>
            ))}
          </select>
        </CardHeader>
        <CardContent>
          {canViewContacts && inactiveQuery.isError && (
            <ErrorState
              title="Error al cargar los contactos inactivos"
              description="No se pudieron cargar los contactos inactivos."
              onRetry={() => inactiveQuery.refetch()}
              isRetrying={inactiveQuery.isRefetching}
            />
          )}
          {canViewContacts && inactiveQuery.isLoading && (
            <p className="text-gray-500">Cargando...</p>
          )}
          {canViewContacts &&
            !inactiveQuery.isLoading &&
            (inactiveQuery.data?.length ? (
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
                {inactiveQuery.data.map((c) => (
                  <div
                    key={c.telefono}
                    className="rounded-md border p-3 text-sm flex justify-between items-center"
                  >
                    <div>
                      <p className="font-medium">{c.nombre || "Sin nombre"}</p>
                      <p className="text-gray-500">{c.telefono}</p>
                    </div>
                    <span
                      className={`rounded-full px-2 py-0.5 text-xs ${
                        c.clasificacion === "inactivo"
                          ? "bg-orange-100 text-orange-700"
                          : "bg-gray-100 text-gray-600"
                      }`}
                    >
                      {c.clasificacion === "inactivo" ? "Inactivo" : "Sin actividad"}
                    </span>
                  </div>
                ))}
              </div>
            ) : (
              <p className="text-gray-500">
                Sin contactos inactivos para el umbral seleccionado.
              </p>
            ))}
          {!canViewContacts && (
            <p className="text-gray-500">No tienes permiso para ver contactos.</p>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
