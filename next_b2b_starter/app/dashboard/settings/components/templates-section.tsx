"use client";

import { useMemo, useState } from "react";
import { CheckCircle2, Clock, FileText, MessageSquarePlus, PauseCircle, RefreshCw, Send, Trash2, XCircle } from "lucide-react";
import { toast } from "sonner";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { StatusChip } from "@/components/ui/status-chip";
import { Textarea } from "@/components/ui/textarea";
import { ui } from "@/lib/copy/ui";
import {
  WhatsAppTemplate,
  WhatsAppTemplateInput,
  TemplateStatus,
} from "@/lib/models/whatsapp-template.model";
import {
  useCreateWhatsAppTemplate,
  useDeleteWhatsAppTemplate,
  useSubmitWhatsAppTemplate,
  useSyncWhatsAppTemplate,
  useUpdateWhatsAppTemplate,
  useWhatsAppTemplatesQuery,
} from "@/lib/hooks/queries/use-whatsapp-template-queries";

const STATUS_ORDER: TemplateStatus[] = [
  "draft",
  "submitted",
  "approved",
  "rejected",
  "paused",
];

const STATUS_LABEL: Record<TemplateStatus, string> = {
  draft: ui.templates.statusDraft,
  submitted: ui.templates.statusSubmitted,
  approved: ui.templates.statusApproved,
  rejected: ui.templates.statusRejected,
  paused: ui.templates.statusPaused,
};

function statusChip(status: TemplateStatus): {
  tone: "emerald" | "red" | "amber" | "gray";
  icon: typeof FileText;
} {
  switch (status) {
    case "approved":
      return { tone: "emerald", icon: CheckCircle2 };
    case "rejected":
      return { tone: "red", icon: XCircle };
    case "submitted":
      return { tone: "amber", icon: Clock };
    case "paused":
      return { tone: "gray", icon: PauseCircle };
    case "draft":
    default:
      return { tone: "gray", icon: FileText };
  }
}

function countParams(body: string): number {
  const matches = body.match(/\{\{\s*(\d+)\s*\}\}/g) ?? [];
  let max = 0;
  for (const m of matches) {
    const idx = parseInt(m.replace(/[^\d]/g, ""), 10);
    if (!Number.isNaN(idx) && idx > max) max = idx;
  }
  return max;
}

const EMPTY_FORM: WhatsAppTemplateInput = {
  name: "",
  category: "",
  language: "es",
  body: "",
};

export function TemplatesSection() {
  const { data: templates, isLoading, error, refetch } = useWhatsAppTemplatesQuery();
  const createMutation = useCreateWhatsAppTemplate();
  const updateMutation = useUpdateWhatsAppTemplate();
  const deleteMutation = useDeleteWhatsAppTemplate();
  const submitMutation = useSubmitWhatsAppTemplate();
  const syncMutation = useSyncWhatsAppTemplate();

  const [dialogOpen, setDialogOpen] = useState(false);
  const [editing, setEditing] = useState<WhatsAppTemplate | null>(null);
  const [form, setForm] = useState<WhatsAppTemplateInput>(EMPTY_FORM);
  const [validationError, setValidationError] = useState<string | null>(null);

  const sorted = useMemo(() => {
    if (!templates) return [];
    const order = (s: TemplateStatus) => STATUS_ORDER.indexOf(s);
    return [...templates].sort((a, b) => order(a.status) - order(b.status));
  }, [templates]);

  function openCreate() {
    setEditing(null);
    setForm(EMPTY_FORM);
    setValidationError(null);
    setDialogOpen(true);
  }

  function openEdit(t: WhatsAppTemplate) {
    setEditing(t);
    setForm({
      name: t.name,
      category: t.category,
      language: t.language,
      body: t.body,
    });
    setValidationError(null);
    setDialogOpen(true);
  }

  function validate(): string | null {
    if (!form.name.trim()) return ui.templates.nameRequired;
    if (!form.category.trim()) return ui.templates.categoryRequired;
    if (!form.language.trim()) return ui.templates.languageRequired;
    if (!form.body.trim()) return ui.templates.bodyRequired;
    return null;
  }

  async function handleSave() {
    const err = validate();
    if (err) {
      setValidationError(err);
      return;
    }
    setValidationError(null);
    try {
      if (editing) {
        await updateMutation.mutateAsync({ id: editing.id, input: form });
        toast.success(ui.templates.updateSuccess);
      } else {
        await createMutation.mutateAsync(form);
        toast.success(ui.templates.createSuccess);
      }
      setDialogOpen(false);
    } catch (e) {
      const msg =
        e instanceof Error ? e.message : String(e);
      toast.error(msg.includes("template_name_conflict")
        ? ui.templates.nameConflictError
        : ui.common.unexpectedError);
    }
  }

  async function handleDelete(t: WhatsAppTemplate) {
    if (t.status !== "draft") {
      toast.error(ui.templates.notDraftError);
      return;
    }
    try {
      await deleteMutation.mutateAsync(t.id);
      toast.success(ui.templates.deleteSuccess);
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e);
      toast.error(msg.includes("template_not_draft")
        ? ui.templates.notDraftError
        : ui.common.unexpectedError);
    }
  }

  async function handleSubmit(t: WhatsAppTemplate) {
    toast.promise(submitMutation.mutateAsync(t.id), {
      loading: ui.templates.submitInProgress,
      success: ui.templates.submitSuccess,
      error: ui.common.unexpectedError,
    });
  }

  async function handleSync(t: WhatsAppTemplate) {
    toast.promise(syncMutation.mutateAsync(t.id), {
      loading: ui.templates.syncInProgress,
      success: ui.templates.syncSuccess,
      error: ui.common.unexpectedError,
    });
  }

  const paramCount = countParams(form.body);

  return (
    <Card>
      <CardHeader className="flex flex-row items-start justify-between">
        <div>
          <CardTitle>{ui.templates.title}</CardTitle>
          <CardDescription>{ui.templates.description}</CardDescription>
        </div>
        <Button onClick={openCreate} size="sm">
          <MessageSquarePlus className="h-4 w-4 mr-1" />
          {ui.templates.createTitle}
        </Button>
      </CardHeader>
      <CardContent>
        {isLoading && (
          <div className="space-y-3">
            <Skeleton className="h-12 w-full" />
            <Skeleton className="h-12 w-full" />
            <Skeleton className="h-12 w-full" />
          </div>
        )}

        {error && !isLoading && (
          <Alert variant="destructive">
            <AlertTitle>{ui.templates.loadFailedTitle}</AlertTitle>
            <AlertDescription>
              {error instanceof Error ? error.message : ui.templates.loadFailedBody}
            </AlertDescription>
            <Button variant="outline" size="sm" className="mt-3" onClick={() => refetch()}>
              {ui.templates.retry}
            </Button>
          </Alert>
        )}

        {!isLoading && !error && sorted.length === 0 && (
          <div className="py-10 text-center">
            <p className="font-medium text-slate-900">{ui.templates.emptyTitle}</p>
            <p className="text-sm text-slate-500">{ui.templates.emptyBody}</p>
          </div>
        )}

        {!isLoading && !error && sorted.length > 0 && (
          <div className="space-y-3">
            {sorted.map((t) => {
              const chip = statusChip(t.status);
              const ChipIcon = chip.icon;
              return (
                <div
                  key={t.id}
                  className="flex items-center justify-between rounded-xl border border-slate-200 bg-white p-4 shadow-sm"
                >
                  <div className="min-w-0">
                    <div className="flex items-center gap-2">
                      <p className="font-medium text-slate-900">{t.name}</p>
                      <StatusChip tone={chip.tone} icon={ChipIcon}>
                        {STATUS_LABEL[t.status]}
                      </StatusChip>
                    </div>
                    <p className="text-sm text-slate-600 truncate mt-1">
                      {t.body}
                    </p>
                    <p className="text-xs text-slate-500 mt-1">
                      {t.language} · {t.category} · {t.param_count} parámetros
                    </p>
                    {t.rejection_reason && t.status === "rejected" && (
                      <p className="text-xs text-red-600 mt-1">{t.rejection_reason}</p>
                    )}
                  </div>
                  <div className="flex items-center gap-1 shrink-0 ml-4">
                    {t.status === "draft" && (
                      <>
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => handleSubmit(t)}
                        >
                          <Send className="h-4 w-4 mr-1" />
                          {ui.templates.submit}
                        </Button>
                        <Button variant="outline" size="sm" onClick={() => openEdit(t)}>
                          {ui.templates.edit}
                        </Button>
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => handleDelete(t)}
                        >
                          <Trash2 className="h-4 w-4 text-red-500" />
                        </Button>
                      </>
                    )}
                    {(t.status === "submitted" || t.status === "approved" || t.status === "rejected" || t.status === "paused") && (
                      <Button variant="outline" size="sm" onClick={() => handleSync(t)}>
                        <RefreshCw className="h-4 w-4 mr-1" />
                        {ui.templates.sync}
                      </Button>
                    )}
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </CardContent>

      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              {editing ? ui.templates.editTitle : ui.templates.createTitle}
            </DialogTitle>
            <DialogDescription>
              {ui.templates.description}
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            {validationError && (
              <Alert variant="destructive">
                <AlertDescription>{validationError}</AlertDescription>
              </Alert>
            )}
            <div className="space-y-2">
              <Label htmlFor="tpl-name">{ui.templates.name}</Label>
              <Input
                id="tpl-name"
                value={form.name}
                placeholder={ui.templates.namePlaceholder}
                onChange={(e) => setForm({ ...form, name: e.target.value })}
              />
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-2">
                <Label htmlFor="tpl-category">{ui.templates.category}</Label>
                <Input
                  id="tpl-category"
                  value={form.category}
                  placeholder={ui.templates.categoryPlaceholder}
                  onChange={(e) => setForm({ ...form, category: e.target.value })}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="tpl-language">{ui.templates.language}</Label>
                <Input
                  id="tpl-language"
                  value={form.language}
                  placeholder={ui.templates.languagePlaceholder}
                  onChange={(e) => setForm({ ...form, language: e.target.value })}
                />
              </div>
            </div>
            <div className="space-y-2">
              <Label htmlFor="tpl-body">{ui.templates.body}</Label>
              <Textarea
                id="tpl-body"
                rows={4}
                value={form.body}
                placeholder={ui.templates.bodyPlaceholder}
                onChange={(e) => setForm({ ...form, body: e.target.value })}
              />
              <p className="text-xs text-slate-500">
                {ui.templates.paramCountHint.replace("{n}", String(paramCount))}
              </p>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDialogOpen(false)}>
              {ui.templates.cancel}
            </Button>
            <Button onClick={handleSave} disabled={createMutation.isPending || updateMutation.isPending}>
              {ui.templates.save}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </Card>
  );
}
