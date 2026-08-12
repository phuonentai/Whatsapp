"use client";

import { Check, Eye, Pencil, RefreshCw, ShieldAlert, Sparkles, Users } from "lucide-react";
import type { EvalResultDto, SegmentFilter } from "@/lib/api/api/dto/campaign.dto";
import { ui, tpl } from "@/lib/copy/ui";

interface AudienceResultCardProps {
  spec: SegmentFilter[];
  preview?: EvalResultDto | null;
  isPreviewing?: boolean;
  isRegenerating?: boolean;
  onAccept: () => void;
  onEdit: () => void;
  onRegenerate: () => void;
  onPreview: () => void;
}

const OP_LABELS: Record<string, string> = {
  eq: ui.crm.opEq,
  neq: ui.crm.opNeq,
  lt: ui.crm.opLt,
  lte: ui.crm.opLte,
  gt: ui.crm.opGt,
  gte: ui.crm.opGte,
  contains: ui.crm.opContains,
  in: ui.crm.opIn,
};

function formatFilterValue(value: unknown): string {
  if (Array.isArray(value)) return value.map(String).join(", ");
  return String(value);
}

function formatFilter(filter: SegmentFilter): string {
  const op = OP_LABELS[filter.op] ?? filter.op;
  return `${filter.field} ${op} ${formatFilterValue(filter.value)}`;
}

const actionButtonClass =
  "inline-flex items-center gap-1.5 rounded px-3 py-1.5 text-xs font-medium transition-colors disabled:opacity-50";

/**
 * Structured display for the AI audience-builder result. Replaces the raw
 * `<pre>` JSON dump: criteria as chips, estimated audience stat, consent
 * exclusion notice, and accept / edit / regenerate actions.
 */
export function AudienceResultCard({
  spec,
  preview,
  isPreviewing = false,
  isRegenerating = false,
  onAccept,
  onEdit,
  onRegenerate,
  onPreview,
}: AudienceResultCardProps) {
  const excludedByConsent = (preview?.excluded_by_gates ?? 0) > 0;

  return (
    <div
      data-testid="audience-result-card"
      className="mt-3 space-y-3 rounded-lg border border-gray-200 bg-white p-4"
    >
      <div className="flex items-center gap-2">
        <Sparkles className="h-4 w-4" style={{ color: "#7c3aed" }} />
        <h3 className="text-sm font-semibold text-gray-900">{ui.crm.aiResultTitle}</h3>
      </div>

      <div className="space-y-1.5">
        <p className="text-xs font-medium uppercase tracking-wider text-gray-500">
          {ui.crm.criteria}
        </p>
        <div className="flex flex-wrap gap-1.5">
          {spec.map((filter, i) => (
            <span
              key={`${filter.field}-${filter.op}-${i}`}
              className="rounded-full border border-violet-200 bg-violet-50 px-2.5 py-1 text-xs font-medium text-violet-800"
            >
              {formatFilter(filter)}
            </span>
          ))}
        </div>
      </div>

      <div className="flex items-center gap-2 rounded-lg bg-gray-50 px-3 py-2.5">
        <Users className="h-4 w-4" style={{ color: "#6b7280" }} />
        <span className="text-xs text-gray-500">{ui.crm.estimatedAudience}:</span>
        <span className="text-sm font-semibold text-gray-900">
          {preview ? preview.total : "—"}
        </span>
        <span className="text-xs text-gray-500">{ui.crm.contacts}</span>
        {excludedByConsent && (
          <span className="text-xs font-medium text-amber-600">
            · {tpl(ui.crm.excludedByConsent, { n: preview!.excluded_by_gates })}
          </span>
        )}
      </div>

      <div
        role="note"
        className="flex items-start gap-2 rounded-lg border border-amber-200 bg-amber-50 px-3 py-2"
      >
        <ShieldAlert className="mt-0.5 h-3.5 w-3.5 flex-shrink-0" style={{ color: "#b45309" }} />
        <p className="text-xs leading-relaxed" style={{ color: "#92400e" }}>
          {ui.crm.consentNotice}
        </p>
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <button
          type="button"
          onClick={onAccept}
          className={actionButtonClass}
          style={{ backgroundColor: "#111827", color: "white" }}
        >
          <Check className="h-3.5 w-3.5" />
          {ui.crm.acceptSegment}
        </button>
        <button
          type="button"
          onClick={onEdit}
          className={actionButtonClass}
          style={{ border: "1px solid #e5e7eb", color: "#374151" }}
        >
          <Pencil className="h-3.5 w-3.5" />
          {ui.crm.editDescription}
        </button>
        <button
          type="button"
          onClick={onRegenerate}
          disabled={isRegenerating}
          className={actionButtonClass}
          style={{ border: "1px solid #e5e7eb", color: "#374151" }}
        >
          <RefreshCw className={isRegenerating ? "h-3.5 w-3.5 animate-spin" : "h-3.5 w-3.5"} />
          {ui.crm.regenerate}
        </button>
        <button
          type="button"
          onClick={onPreview}
          disabled={isPreviewing}
          className={actionButtonClass}
          style={{ border: "1px solid #e5e7eb", color: "#374151" }}
        >
          <Eye className="h-3.5 w-3.5" />
          {isPreviewing ? ui.crm.previewing : ui.crm.preview}
        </button>
      </div>
    </div>
  );
}
