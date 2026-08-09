"use client";

import { usePlaybooksQuery } from "@/lib/hooks/queries/use-playbooks-query";
import type { PlaybookGuionDto } from "@/lib/api/api/dto/playbook.dto";

interface QuickRepliesProps {
  conversationId: number;
  onSelect: (mensaje: string) => void;
}

export function QuickReplies({ conversationId, onSelect }: QuickRepliesProps) {
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
      {guiones.map((guion) => (
        <button
          key={guion.id}
          type="button"
          title={guion.mensaje}
          onClick={() => onSelect(guion.mensaje)}
          className="max-w-52 truncate rounded-full border border-gray-200 bg-white px-3 py-1 text-xs font-medium text-gray-700 transition hover:border-blue-300 hover:bg-blue-50 hover:text-blue-700"
        >
          {guion.titulo}
        </button>
      ))}
    </div>
  );
}
