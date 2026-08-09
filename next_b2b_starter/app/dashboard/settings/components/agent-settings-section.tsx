"use client";

import { useMemo, useState } from "react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Loader2, Sparkles, Plus, X } from "lucide-react";
import { useAgentSettingsQuery } from "@/lib/hooks/queries/use-agent-settings-query";
import { useUpdateAgentSettings } from "@/lib/hooks/mutations/use-update-agent-settings";
import type { AgentSettings } from "@/lib/models/agent.model";

export function AgentSettingsSection() {
  const { data: settings, isLoading, error, refetch } = useAgentSettingsQuery();
  const updateMutation = useUpdateAgentSettings();

  // Local edits overlay the server settings; no effect-sync needed.
  const [edits, setEdits] = useState<Partial<AgentSettings>>({});

  const form = useMemo<AgentSettings | null>(() => {
    if (!settings) return null;
    return {
      ...settings,
      ...edits,
      guardrails: {
        never: {
          max_discount_percent: edits.guardrails?.never?.max_discount_percent ?? settings.guardrails?.never?.max_discount_percent ?? 10,
          forbidden_terms: edits.guardrails?.never?.forbidden_terms ?? settings.guardrails?.never?.forbidden_terms ?? [],
        },
        escalate: {
          terms: edits.guardrails?.escalate?.terms ?? settings.guardrails?.escalate?.terms ?? [],
        },
      },
    };
  }, [settings, edits]);

  const [newForbiddenTerm, setNewForbiddenTerm] = useState("");
  const [newEscalateTerm, setNewEscalateTerm] = useState("");

  if (isLoading) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-96 rounded-xl" />
      </div>
    );
  }

  if (error && !settings) {
    return (
      <Alert variant="destructive" className="border border-red-200 bg-red-50">
        <AlertTitle>Error al cargar</AlertTitle>
        <AlertDescription>
          {error.message || "No se pudo cargar la configuración del asistente IA."}
        </AlertDescription>
        <Button variant="outline" size="sm" className="mt-3" onClick={() => refetch()}>
          Reintentar
        </Button>
      </Alert>
    );
  }

  if (!form) return null;

  const set = <K extends keyof AgentSettings>(key: K, value: AgentSettings[K]) =>
    setEdits((prev) => ({ ...prev, [key]: value }));

  const addForbiddenTerm = () => {
    const term = newForbiddenTerm.trim();
    if (!term) return;
    set("guardrails", {
      ...form.guardrails,
      never: { ...form.guardrails.never, forbidden_terms: [...(form.guardrails.never?.forbidden_terms ?? []), term] },
    });
    setNewForbiddenTerm("");
  };

  const addEscalateTerm = () => {
    const term = newEscalateTerm.trim();
    if (!term) return;
    set("guardrails", {
      ...form.guardrails,
      escalate: { ...form.guardrails.escalate, terms: [...(form.guardrails.escalate?.terms ?? []), term] },
    });
    setNewEscalateTerm("");
  };

  const removeTerm = (list: "never" | "escalate", index: number) => {
    const current = list === "never"
      ? (form.guardrails.never?.forbidden_terms ?? [])
      : (form.guardrails.escalate?.terms ?? []);
    const next = current.filter((_, i) => i !== index);
    if (list === "never") {
      set("guardrails", { ...form.guardrails, never: { ...form.guardrails.never, forbidden_terms: next } });
    } else {
      set("guardrails", { ...form.guardrails, escalate: { ...form.guardrails.escalate, terms: next } });
    }
  };

  const handleSave = async () => {
    if (!form) return;
    try {
      await updateMutation.mutateAsync(form);
      setEdits({});
    } catch {
      // toast handled by mutation
    }
  };

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <div className="flex items-center gap-3">
            <Sparkles className="h-6 w-6 text-gray-600" />
            <div>
              <CardTitle>Asistente IA de WhatsApp</CardTitle>
              <CardDescription>
                Configura cómo el agente redacta y responde mensajes en tus conversaciones.
              </CardDescription>
            </div>
          </div>
        </CardHeader>
        <CardContent className="space-y-6">
          <div className="grid gap-5 sm:grid-cols-2">
            <div className="space-y-2">
              <Label className="text-sm font-medium">Modo</Label>
              <select
                className="w-full rounded-md border border-gray-300 bg-white px-3 py-2 text-sm"
                value={form.mode}
                onChange={(e) => set("mode", e.target.value as AgentSettings["mode"])}
                disabled={updateMutation.isPending}
              >
                <option value="copilot">Copiloto (sugiere respuestas, un humano aprueba)</option>
                <option value="autopilot">Autopiloto (responde solo dentro de la ventana)</option>
              </select>
              <p className="text-xs text-gray-500">
                Copiloto redacta borradores. Autopiloto envía respuestas autónomas solo si todas las reglas se cumplen.
              </p>
            </div>

            <div className="space-y-2">
              <Label className="text-sm font-medium">Tono de voz</Label>
              <select
                className="w-full rounded-md border border-gray-300 bg-white px-3 py-2 text-sm"
                value={form.tone}
                onChange={(e) => set("tone", e.target.value as AgentSettings["tone"])}
                disabled={updateMutation.isPending}
              >
                <option value="formal">Formal (usted)</option>
                <option value="casual">Informal (tú/vos)</option>
              </select>
              <p className="text-xs text-gray-500">Registro usado en las respuestas generadas.</p>
            </div>
          </div>

          <div className="space-y-2">
            <Label className="text-sm font-medium">Voz de la marca</Label>
            <Input
              type="text"
              placeholder="Ej: Atendemos con calidez y rapidez, siempre en español colombiano."
              value={form.brand_voice ?? ""}
              onChange={(e) => set("brand_voice", e.target.value)}
              disabled={updateMutation.isPending}
            />
          </div>

          <div className="grid gap-5 sm:grid-cols-3">
            <div className="space-y-2">
              <Label className="text-sm font-medium">Ventana autopiloto (inicio)</Label>
              <Input
                type="time"
                value={form.autopilot_start ?? ""}
                onChange={(e) => set("autopilot_start", e.target.value)}
                disabled={updateMutation.isPending}
              />
            </div>
            <div className="space-y-2">
              <Label className="text-sm font-medium">Ventana autopiloto (fin)</Label>
              <Input
                type="time"
                value={form.autopilot_end ?? ""}
                onChange={(e) => set("autopilot_end", e.target.value)}
                disabled={updateMutation.isPending}
              />
            </div>
            <div className="space-y-2">
              <Label className="text-sm font-medium">Zona horaria</Label>
              <Input
                type="text"
                value={form.timezone ?? "America/Bogota"}
                onChange={(e) => set("timezone", e.target.value)}
                disabled={updateMutation.isPending}
              />
            </div>
          </div>

          <div className="grid gap-5 sm:grid-cols-2">
            <div className="space-y-2">
              <Label className="text-sm font-medium">Límite de mensajes diarios (0 = ilimitado)</Label>
              <Input
                type="number"
                min={0}
                value={form.max_daily_messages}
                onChange={(e) => set("max_daily_messages", Number(e.target.value))}
                disabled={updateMutation.isPending}
              />
            </div>
            <div className="space-y-2">
              <Label className="text-sm font-medium">Descuento máximo permitido (%)</Label>
              <Input
                type="number"
                min={0}
                value={form.guardrails.never?.max_discount_percent ?? 10}
                onChange={(e) =>
                  set("guardrails", {
                    ...form.guardrails,
                    never: { ...form.guardrails.never, max_discount_percent: Number(e.target.value) },
                  })
                }
                disabled={updateMutation.isPending}
              />
            </div>
          </div>

          <div className="flex items-center justify-between rounded-lg border border-gray-200 bg-gray-50 p-4">
            <div>
              <Label className="text-sm font-medium">Interruptor de emergencia</Label>
              <p className="text-xs text-gray-500">Bloquea todos los envíos del agente inmediatamente.</p>
            </div>
            <Switch
              checked={form.kill_switch}
              onCheckedChange={(v) => set("kill_switch", v)}
              disabled={updateMutation.isPending}
            />
          </div>

          <div className="rounded-lg border border-gray-200 p-4">
            <Label className="text-sm font-medium">Términos prohibidos (nunca usarlos en respuestas)</Label>
            <div className="mt-2 flex flex-wrap gap-2">
              {(form.guardrails.never?.forbidden_terms ?? []).map((term, i) => (
                <span key={`forbidden-${i}`} className="inline-flex items-center gap-1 rounded-full bg-red-50 px-2.5 py-1 text-xs text-red-700">
                  {term}
                  <button type="button" onClick={() => removeTerm("never", i)} aria-label={`Quitar ${term}`}>
                    <X className="h-3 w-3" />
                  </button>
                </span>
              ))}
            </div>
            <div className="mt-2 flex gap-2">
              <Input
                type="text"
                placeholder="Ej: garantía total, envío gratis"
                value={newForbiddenTerm}
                onChange={(e) => setNewForbiddenTerm(e.target.value)}
                onKeyDown={(e) => { if (e.key === "Enter") { e.preventDefault(); addForbiddenTerm(); } }}
              />
              <Button type="button" variant="outline" onClick={addForbiddenTerm}>
                <Plus className="h-4 w-4" />
              </Button>
            </div>
          </div>

          <div className="rounded-lg border border-gray-200 p-4">
            <Label className="text-sm font-medium">Temas de escalamiento (derivar a un humano)</Label>
            <p className="text-xs text-gray-500">Si el cliente menciona estos temas, el agente nunca responde solo.</p>
            <div className="mt-2 flex flex-wrap gap-2">
              {(form.guardrails.escalate?.terms ?? []).map((term, i) => (
                <span key={`escalate-${i}`} className="inline-flex items-center gap-1 rounded-full bg-amber-50 px-2.5 py-1 text-xs text-amber-700">
                  {term}
                  <button type="button" onClick={() => removeTerm("escalate", i)} aria-label={`Quitar ${term}`}>
                    <X className="h-3 w-3" />
                  </button>
                </span>
              ))}
            </div>
            <div className="mt-2 flex gap-2">
              <Input
                type="text"
                placeholder="Ej: abogado, legal, garantía, demanda"
                value={newEscalateTerm}
                onChange={(e) => setNewEscalateTerm(e.target.value)}
                onKeyDown={(e) => { if (e.key === "Enter") { e.preventDefault(); addEscalateTerm(); } }}
              />
              <Button type="button" variant="outline" onClick={addEscalateTerm}>
                <Plus className="h-4 w-4" />
              </Button>
            </div>
          </div>

          <div className="flex items-center justify-between rounded-lg border border-gray-200 bg-gray-50 p-4">
            <div>
              <Label className="text-sm font-medium">Consentimiento requerido (Ley 1581)</Label>
              <p className="text-xs text-gray-500">
                Solicita autorización de tratamiento de datos antes de responder autónomamente.
              </p>
            </div>
            <Switch
              checked={form.consent_required}
              onCheckedChange={(v) => set("consent_required", v)}
              disabled={updateMutation.isPending}
            />
          </div>

          {form.consent_required && (
            <div className="space-y-2">
              <Label className="text-sm font-medium">Mensaje de consentimiento</Label>
              <textarea
                className="min-h-24 w-full rounded-md border border-gray-300 bg-white px-3 py-2 text-sm"
                value={form.consent_template ?? ""}
                onChange={(e) => set("consent_template", e.target.value)}
                disabled={updateMutation.isPending}
                placeholder="Hola, para atenderte necesitamos tu autorización para el tratamiento de tus datos personales conforme a la Ley 1581. ¿Nos autorizas? (Responde sí o acepto)."
              />
            </div>
          )}

          <div>
            <Button
              onClick={handleSave}
              disabled={updateMutation.isPending}
              className="bg-gray-900 text-white hover:bg-gray-800"
            >
              {updateMutation.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              Guardar configuración
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
