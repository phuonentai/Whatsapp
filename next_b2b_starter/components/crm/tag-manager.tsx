"use client";
import { useTagsQuery } from "@/lib/hooks/queries/use-crm-queries";
import { useCreateTag, useDeleteTag } from "@/lib/hooks/mutations/use-crm-mutations";
import { useState } from "react";
import { ConfirmDialog } from "@/components/crm/confirm-dialog";

export function TagManager() {
  const { data: tags, isLoading } = useTagsQuery();
  const createTag = useCreateTag();
  const deleteTag = useDeleteTag();
  const [showForm, setShowForm] = useState(false);
  const [nombre, setNombre] = useState("");
  const [color, setColor] = useState("#3B82F6");
  const [deleting, setDeleting] = useState<{ id: number; nombre: string } | null>(null);

  if (isLoading) return <div className="text-gray-500">Cargando etiquetas...</div>;

  const handleCreate = () => {
    if (!nombre.trim()) return;
    createTag.mutate(
      { nombre: nombre.trim(), color },
      {
        onSuccess: () => {
          setNombre("");
          setColor("#3B82F6");
          setShowForm(false);
        },
      }
    );
  };

  const handleDelete = () => {
    if (!deleting) return;
    deleteTag.mutate(deleting.id, {
      onSuccess: () => setDeleting(null),
    });
  };

  return (
    <div>
      <div className="mb-4">
        {showForm ? (
          <div className="flex flex-wrap gap-2">
            <input
              name="nombre"
              value={nombre}
              onChange={(e) => setNombre(e.target.value)}
              placeholder="Nombre de etiqueta"
              className="border rounded px-3 py-2 w-64"
            />
            <input
              name="color"
              type="color"
              value={color}
              onChange={(e) => setColor(e.target.value)}
              className="w-10 h-10 rounded border cursor-pointer"
            />
            <button
              onClick={handleCreate}
              disabled={createTag.isPending}
              className="bg-gray-900 text-white px-4 py-2 rounded text-sm"
            >
              Guardar
            </button>
            <button
              onClick={() => setShowForm(false)}
              className="border px-4 py-2 rounded text-sm text-gray-600"
            >
              Cancelar
            </button>
          </div>
        ) : (
          <button
            onClick={() => setShowForm(true)}
            className="bg-gray-900 text-white px-4 py-2 rounded text-sm"
          >
            Nueva etiqueta
          </button>
        )}
      </div>
      <div className="flex flex-wrap gap-2" data-testid="tag-list">
        {tags?.map((t) => (
          <div
            key={t.id}
            className="flex items-center gap-2 px-3 py-1 rounded-full text-sm text-white"
            style={{ backgroundColor: t.color || "#6B7280" }}
          >
            {t.nombre}
            <button
              aria-label="Eliminar"
              onClick={() => setDeleting({ id: t.id, nombre: t.nombre })}
              className="ml-1 hover:text-gray-200"
            >
              ×
            </button>
          </div>
        ))}
        {(!tags || tags.length === 0) && (
          <div className="text-gray-400">Sin etiquetas. Crea la primera.</div>
        )}
      </div>

      <ConfirmDialog
        open={Boolean(deleting)}
        onOpenChange={(next) => !next && setDeleting(null)}
        title="Eliminar etiqueta"
        description={`¿Estás seguro de eliminar la etiqueta ${deleting?.nombre ?? ""}?`}
        confirmLabel="Eliminar"
        loading={deleteTag.isPending}
        onConfirm={handleDelete}
      />
    </div>
  );
}
