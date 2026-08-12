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
import { ui, tpl } from "@/lib/copy/ui";

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
        <AlertTitle>{ui.agent.loadErrorTitle}</AlertTitle>
        <AlertDescription>
          {error.message || ui.agent.loadErrorBody}
        </AlertDescription>
        <Button variant="outline" size="sm" className="mt-3" onClick={() => refetch()}>
          {ui.common.retry}
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
            <Sparkles className="h-6 w-6 text-slate-600" />
            <div>
              <CardTitle>{ui.agent.settingsTitle}</CardTitle>
              <CardDescription>
                {ui.agent.settingsDescription}
              </CardDescription>
            </div>
          </div>
        </CardHeader>
        <CardContent className="space-y-6">
          <div className="grid gap-5 sm:grid-cols-2">
            <div className="space-y-2">
              <Label className="text-sm font-medium">{ui.agent.mode}</Label>
              <select
                className="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm"
                value={form.mode}
                onChange={(e) => set("mode", e.target.value as AgentSettings["mode"])}
                disabled={updateMutation.isPending}
              >
                <option value="copilot">{ui.agent.modeCopilot}</option>
                <option value="autopilot">{ui.agent.modeAutopilot}</option>
              </select>
              <p className="text-xs text-slate-500">
                {ui.agent.modeHint}
              </p>
            </div>

            <div className="space-y-2">
              <Label className="text-sm font-medium">{ui.agent.tone}</Label>
              <select
                className="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm"
                value={form.tone}
                onChange={(e) => set("tone", e.target.value as AgentSettings["tone"])}
                disabled={updateMutation.isPending}
              >
                <option value="formal">{ui.agent.toneFormal}</option>
                <option value="casual">{ui.agent.toneCasual}</option>
              </select>
              <p className="text-xs text-slate-500">{ui.agent.toneHint}</p>
            </div>
          </div>

          <div className="space-y-2">
            <Label className="text-sm font-medium">{ui.agent.brandVoice}</Label>
            <Input
              type="text"
              placeholder={ui.agent.brandVoicePlaceholder}
              value={form.brand_voice ?? ""}
              onChange={(e) => set("brand_voice", e.target.value)}
              disabled={updateMutation.isPending}
            />
          </div>

          <div className="grid gap-5 sm:grid-cols-3">
            <div className="space-y-2">
              <Label className="text-sm font-medium">{ui.agent.autopilotStart}</Label>
              <Input
                type="time"
                value={form.autopilot_start ?? ""}
                onChange={(e) => set("autopilot_start", e.target.value)}
                disabled={updateMutation.isPending}
              />
            </div>
            <div className="space-y-2">
              <Label className="text-sm font-medium">{ui.agent.autopilotEnd}</Label>
              <Input
                type="time"
                value={form.autopilot_end ?? ""}
                onChange={(e) => set("autopilot_end", e.target.value)}
                disabled={updateMutation.isPending}
              />
            </div>
            <div className="space-y-2">
              <Label className="text-sm font-medium">{ui.agent.timezone}</Label>
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
              <Label className="text-sm font-medium">{ui.agent.maxDailyMessages}</Label>
              <Input
                type="number"
                min={0}
                value={form.max_daily_messages}
                onChange={(e) => set("max_daily_messages", Number(e.target.value))}
                disabled={updateMutation.isPending}
              />
            </div>
            <div className="space-y-2">
              <Label className="text-sm font-medium">{ui.agent.maxDiscount}</Label>
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

          <div className="flex items-center justify-between rounded-lg border border-slate-200 bg-slate-50 p-4">
            <div>
              <Label className="text-sm font-medium">{ui.agent.killSwitch}</Label>
              <p className="text-xs text-slate-500">{ui.agent.killSwitchHint}</p>
            </div>
            <Switch
              checked={form.kill_switch}
              onCheckedChange={(v) => set("kill_switch", v)}
              disabled={updateMutation.isPending}
            />
          </div>

          <div className="rounded-lg border border-slate-200 p-4">
            <Label className="text-sm font-medium">{ui.agent.forbiddenTerms}</Label>
            <div className="mt-2 flex flex-wrap gap-2">
              {(form.guardrails.never?.forbidden_terms ?? []).map((term, i) => (
                <span key={`forbidden-${i}`} className="inline-flex items-center gap-1 rounded-full bg-red-50 px-2.5 py-1 text-xs text-red-700">
                  {term}
                  <button type="button" onClick={() => removeTerm("never", i)} aria-label={tpl(ui.agent.removeAria, { term })}>
                    <X className="h-3 w-3" />
                  </button>
                </span>
              ))}
            </div>
            <div className="mt-2 flex gap-2">
              <Input
                type="text"
                placeholder={ui.agent.forbiddenPlaceholder}
                value={newForbiddenTerm}
                onChange={(e) => setNewForbiddenTerm(e.target.value)}
                onKeyDown={(e) => { if (e.key === "Enter") { e.preventDefault(); addForbiddenTerm(); } }}
              />
              <Button type="button" variant="outline" onClick={addForbiddenTerm}>
                <Plus className="h-4 w-4" />
              </Button>
            </div>
          </div>

          <div className="rounded-lg border border-slate-200 p-4">
            <Label className="text-sm font-medium">{ui.agent.escalateTerms}</Label>
            <p className="text-xs text-slate-500">{ui.agent.escalateHint}</p>
            <div className="mt-2 flex flex-wrap gap-2">
              {(form.guardrails.escalate?.terms ?? []).map((term, i) => (
                <span key={`escalate-${i}`} className="inline-flex items-center gap-1 rounded-full bg-amber-50 px-2.5 py-1 text-xs text-amber-700">
                  {term}
                  <button type="button" onClick={() => removeTerm("escalate", i)} aria-label={tpl(ui.agent.removeAria, { term })}>
                    <X className="h-3 w-3" />
                  </button>
                </span>
              ))}
            </div>
            <div className="mt-2 flex gap-2">
              <Input
                type="text"
                placeholder={ui.agent.escalatePlaceholder}
                value={newEscalateTerm}
                onChange={(e) => setNewEscalateTerm(e.target.value)}
                onKeyDown={(e) => { if (e.key === "Enter") { e.preventDefault(); addEscalateTerm(); } }}
              />
              <Button type="button" variant="outline" onClick={addEscalateTerm}>
                <Plus className="h-4 w-4" />
              </Button>
            </div>
          </div>

          <div className="flex items-center justify-between rounded-lg border border-slate-200 bg-slate-50 p-4">
            <div>
              <Label className="text-sm font-medium">{ui.agent.consentRequired}</Label>
              <p className="text-xs text-slate-500">
                {ui.agent.consentHint}
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
              <Label className="text-sm font-medium">{ui.agent.consentMessage}</Label>
              <textarea
                className="min-h-24 w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm"
                value={form.consent_template ?? ""}
                onChange={(e) => set("consent_template", e.target.value)}
                disabled={updateMutation.isPending}
                placeholder={ui.agent.consentPlaceholder}
              />
            </div>
          )}

          <div>
            <Button
              onClick={handleSave}
              disabled={updateMutation.isPending}
              className="bg-emerald-500 text-white hover:bg-emerald-600"
            >
              {updateMutation.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              {ui.agent.saveConfig}
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
