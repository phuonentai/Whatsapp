"use client";
import { useTagsQuery } from "@/lib/hooks/queries/use-crm-queries";
import { useCreateTag, useDeleteTag } from "@/lib/hooks/mutations/use-crm-mutations";
import { useState } from "react";

export function TagManager() {
  const { data: tags, isLoading } = useTagsQuery();
  const createTag = useCreateTag();
  const deleteTag = useDeleteTag();
  const [nombre, setNombre] = useState("");
  const [color, setColor] = useState("#3B82F6");

  if (isLoading) return <div className="text-gray-500">Cargando etiquetas...</div>;

  const handleCreate = () => {
    if (!nombre.trim()) return;
    createTag.mutate({ nombre: nombre.trim(), color });
    setNombre("");
  };

  return (
    <div>
      <div className="flex gap-2 mb-4">
        <input value={nombre} onChange={(e) => setNombre(e.target.value)}
          placeholder="Nombre de etiqueta" className="border rounded px-3 py-2 w-64" />
        <input type="color" value={color} onChange={(e) => setColor(e.target.value)}
          className="w-10 h-10 rounded border cursor-pointer" />
        <button onClick={handleCreate} className="bg-blue-600 text-white px-4 py-2 rounded text-sm">
          Agregar
        </button>
      </div>
      <div className="flex flex-wrap gap-2">
        {tags?.map((t) => (
          <div key={t.id} className="flex items-center gap-2 px-3 py-1 rounded-full text-sm text-white"
            style={{ backgroundColor: t.color || "#6B7280" }}>
            {t.nombre}
            <button onClick={() => deleteTag.mutate(t.id)} className="ml-1 hover:text-gray-200">×</button>
          </div>
        ))}
        {(!tags || tags.length === 0) && (
          <div className="text-gray-400">Sin etiquetas. Crea la primera.</div>
        )}
      </div>
    </div>
  );
}
