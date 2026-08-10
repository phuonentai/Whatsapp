"use client";

import { useState } from "react";
import { toast } from "sonner";
import { useTagsQuery, useEntityTagsQuery } from "@/lib/hooks/queries/use-crm-queries";
import { useTagEntity, useUntagEntity } from "@/lib/hooks/mutations/use-crm-mutations";

export function TagPicker({ entityType, entityId }: { entityType: string; entityId: number }) {
  const { data: tags } = useTagsQuery();
  const { data: entityTags } = useEntityTagsQuery(entityType, entityId);
  const tagEntity = useTagEntity();
  const untagEntity = useUntagEntity();
  const [selecting, setSelecting] = useState(false);

  const attachedIds = new Set(entityTags?.map((t) => t.id) ?? []);
  const available = (tags ?? []).filter((t) => !attachedIds.has(t.id));

  const handleAttach = (tagId: number) => {
    tagEntity.mutate(
      { entityType, entityId, tagId },
      { onSuccess: () => toast.success("Etiqueta asignada") }
    );
  };

  const handleDetach = (tagId: number) => {
    untagEntity.mutate(
      { entityType, entityId, tagId },
      { onSuccess: () => toast.success("Etiqueta removida") }
    );
  };

  return (
    <div className="mt-2">
      <div className="flex flex-wrap gap-2 mb-2">
        {(entityTags ?? []).map((tag) => (
          <span
            key={tag.id}
            data-testid="entity-tag"
            className="inline-flex items-center gap-1 px-2 py-1 rounded text-xs"
            style={{ backgroundColor: tag.color || "#e5e7eb", color: "#1f2937" }}
          >
            {tag.nombre}
            <button
              aria-label="Quitar"
              onClick={() => handleDetach(tag.id)}
              className="text-gray-500 hover:text-red-600"
            >
              ×
            </button>
          </span>
        ))}
      </div>
      {selecting ? (
        <div className="flex gap-2 items-center">
          <select
            aria-label="Seleccionar etiqueta"
            value=""
            onChange={(e) => {
              const id = Number(e.target.value);
              if (id) handleAttach(id);
              setSelecting(false);
            }}
            className="border rounded px-2 py-1 text-sm"
          >
            <option value="">Elegir etiqueta...</option>
            {available.map((tag) => (
              <option key={tag.id} value={tag.id}>
                {tag.nombre}
              </option>
            ))}
          </select>
          <button onClick={() => setSelecting(false)} className="text-gray-500 text-sm hover:underline">
            Cancelar
          </button>
        </div>
      ) : (
        <button
          onClick={() => setSelecting(true)}
          disabled={available.length === 0}
          className="text-blue-600 hover:underline text-sm disabled:text-gray-400"
        >
          {available.length === 0 ? "Sin etiquetas disponibles" : "Asignar etiqueta"}
        </button>
      )}
    </div>
  );
}
