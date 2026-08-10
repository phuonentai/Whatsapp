"use client";

import { usePlaybooksQuery } from "@/lib/hooks/queries/use-playbooks-query";
import type { PlaybookGuionDto } from "@/lib/api/api/dto/playbook.dto";

interface QuickRepliesProps {
  conversationId: number;
  onSelect: (guion: PlaybookGuionDto) => void;
  sequenceActive: boolean;
  sequenceStep: number | null;
  sequenceTotal: number;
}

export function QuickReplies({
  conversationId,
  onSelect,
  sequenceActive,
  sequenceStep,
  sequenceTotal,
}: QuickRepliesProps) {
  const { data: playbooks, isLoading } = usePlaybooksQuery();

  if (isLoading || !conversationId) {
    return null;
  }

  const guiones: PlaybookGuionDto[] = (playbooks ?? []).flatMap(
    (pb) => (pb.applied && pb.guiones ? pb.guiones : [])
  );

  if (guiones.length === 0) {
    return null;
  }

  return (
    <div className="flex flex-wrap items-center gap-2 border-t border-gray-100 bg-gray-50/60 px-4 py-2">
      {sequenceActive ? (
        <span className="rounded-full bg-blue-100 px-3 py-1 text-xs font-semibold text-blue-700">
          Paso {sequenceStep} de {sequenceTotal}
        </span>
      ) : null}
      {guiones.map((guion) =>
        guion.pasos && guion.pasos.length > 0 ? (
          <button
            key={guion.id}
            type="button"
            title={guion.pasos.map((p) => p.titulo).join(" → ")}
            onClick={() => onSelect(guion)}
            className="flex max-w-60 items-center gap-1 truncate rounded-full border border-violet-200 bg-violet-50 px-3 py-1 text-xs font-medium text-violet-700 transition hover:border-violet-300 hover:bg-violet-100"
          >
            <span className="truncate">{guion.titulo}</span>
            <span className="shrink-0 rounded-full bg-violet-200 px-1.5 text-[10px] font-bold text-violet-800">
              {guion.pasos.length} pasos
            </span>
          </button>
        ) : (
          <button
            key={guion.id}
            type="button"
            title={guion.mensaje}
            onClick={() => onSelect(guion)}
            className="max-w-52 truncate rounded-full border border-gray-200 bg-white px-3 py-1 text-xs font-medium text-gray-700 transition hover:border-blue-300 hover:bg-blue-50 hover:text-blue-700"
          >
            {guion.titulo}
          </button>
        )
      )}
    </div>
  );
}
