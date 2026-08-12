"use client";

import { useState } from "react";
import { toast } from "sonner";
import { ui } from "@/lib/copy/ui";
import { useFollowUpSettingsQuery } from "@/lib/hooks/queries/use-procurement-queries";
import type { FollowUpSettingsDto } from "@/lib/api/api/dto/procurement.dto";
import { useUpdateFollowUpSettings } from "@/lib/hooks/mutations/use-procurement-mutations";
import { ErrorState } from "@/components/common/error-state";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";

export function FollowUpSettingsPanel() {
  const { data: settings, isLoading, isError, refetch, isRefetching } =
    useFollowUpSettingsQuery();

  if (isLoading) return <div className="text-gray-500">{ui.procurement.loading}</div>;
  if (isError) {
    return (
      <ErrorState
        title={ui.procurement.errorLoading}
        description=""
        onRetry={() => refetch()}
        isRetrying={isRefetching}
      />
    );
  }

  // The form mounts once the settings are loaded (keyed remount keeps the
  // local state in sync without effects).
  return <FollowUpSettingsForm key={settings ? "loaded" : "loading"} settings={settings!} />;
}

function FollowUpSettingsForm({ settings }: { settings: FollowUpSettingsDto }) {
  const mutation = useUpdateFollowUpSettings();
  const [enabled, setEnabled] = useState(settings?.Enabled ?? false);
  const [deadlineHours, setDeadlineHours] = useState(settings?.DeadlineHours ?? 4);
  const [maxNudges, setMaxNudges] = useState(settings?.MaxNudges ?? 1);
  const [template, setTemplate] = useState(
    settings?.MessageTemplate ??
      "Hola [proveedor], te recordamos la cotización pendiente de hoy. Quedamos atentos."
  );
  const [deadlineError, setDeadlineError] = useState<string | null>(null);
  const [nudgesError, setNudgesError] = useState<string | null>(null);

  const preview = template.replace("[proveedor]", "Distribuciones Andinas SAS");

  const onSave = async () => {
    const dh = Number(deadlineHours);
    const mn = Number(maxNudges);
    let ok = true;
    if (dh < 1 || dh > 168) {
      setDeadlineError("Las horas de plazo deben estar entre 1 y 168.");
      ok = false;
    } else setDeadlineError(null);
    if (mn < 0 || mn > 5) {
      setNudgesError("El número de recordatorios debe estar entre 0 y 5.");
      ok = false;
    } else setNudgesError(null);
    if (!ok) return;
    try {
      await mutation.mutateAsync({
        enabled,
        deadline_hours: dh,
        max_nudges: mn,
        message_template: template,
      });
      toast.success(ui.procurement.followUpSaved);
    } catch {
      // mutation onError toasts the Spanish error
    }
  };

  return (
    <div className="space-y-4">
      <div>
        <h2 className="text-lg font-semibold">{ui.procurement.followUpTitle}</h2>
        <p className="text-sm text-gray-500">{ui.procurement.followUpHint}</p>
      </div>

      <div className="flex items-center justify-between rounded-md border p-3">
        <Label>{ui.procurement.followUpEnabled}</Label>
        <Switch checked={enabled} onCheckedChange={setEnabled} />
      </div>

      <div className="grid gap-4 sm:grid-cols-2">
        <div className="space-y-1.5">
          <Label htmlFor="deadline-hours">{ui.procurement.followUpDeadline}</Label>
          <Input
            id="deadline-hours"
            type="number"
            min={1}
            max={168}
            value={deadlineHours}
            onChange={(e) => setDeadlineHours(Number(e.target.value))}
          />
          {deadlineError && <p className="text-sm text-red-600">{deadlineError}</p>}
        </div>
        <div className="space-y-1.5">
          <Label htmlFor="max-nudges">{ui.procurement.followUpMaxNudges}</Label>
          <Input
            id="max-nudges"
            type="number"
            min={0}
            max={5}
            value={maxNudges}
            onChange={(e) => setMaxNudges(Number(e.target.value))}
          />
          {nudgesError && <p className="text-sm text-red-600">{nudgesError}</p>}
        </div>
      </div>

      <div className="space-y-1.5">
        <Label htmlFor="followup-template">{ui.procurement.followUpTemplate}</Label>
        <textarea
          id="followup-template"
          rows={3}
          className="w-full rounded-md border border-gray-300 p-2 text-sm"
          value={template}
          onChange={(e) => setTemplate(e.target.value)}
        />
      </div>

      <div className="rounded-md bg-gray-50 p-3">
        <p className="text-xs font-medium text-gray-500">{ui.procurement.followUpTemplatePreview}</p>
        <p className="mt-1 text-sm text-gray-700">{preview}</p>
      </div>

      <div className="flex justify-end">
        <Button onClick={onSave} disabled={mutation.isPending}>
          {ui.common.save}
        </Button>
      </div>
    </div>
  );
}
