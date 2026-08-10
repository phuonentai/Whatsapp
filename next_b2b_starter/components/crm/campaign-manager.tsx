"use client";

import { useState } from "react";
import { toast } from "sonner";
import {
  useAiBuild,
  useCreateCampaign,
  useCreateSegment,
  useDeleteSegment,
  useLaunchCampaign,
} from "@/lib/hooks/mutations/use-campaign-mutations";
import {
  useCampaignsQuery,
  useCampaignRecipientsQuery,
  useSegmentsQuery,
} from "@/lib/hooks/queries/use-campaign-queries";
import { campaignRepository } from "@/lib/api/api/repositories/campaign-repository";
import type { CampaignDto, EvalResultDto, SegmentDto, SegmentFilter } from "@/lib/api/api/dto/campaign.dto";

// Preset templates the SMB actually uses; each maps to a whitelisted filter spec.
const PRESETS: { label: string; spec: SegmentFilter[] }[] = [
  { label: "Clientes", spec: [{ field: "lead_status", op: "eq", value: "cliente" }] },
  { label: "Contactados este mes", spec: [{ field: "recency_days", op: "lte", value: 30 }] },
  { label: "Leads nuevos", spec: [{ field: "lead_status", op: "eq", value: "nuevo" }] },
  { label: "Calificados", spec: [{ field: "lead_status", op: "eq", value: "calificado" }] },
];

export function CampaignManager() {
  const { data: segments, isLoading } = useSegmentsQuery();
  const { data: campaigns } = useCampaignsQuery();

  const createSegment = useCreateSegment();
  const deleteSegment = useDeleteSegment();
  const createCampaign = useCreateCampaign();
  const launchCampaign = useLaunchCampaign();
  const aiBuild = useAiBuild();

  const [nombre, setNombre] = useState("");
  const [spec, setSpec] = useState<SegmentFilter[]>([]);
  const [preview, setPreview] = useState<EvalResultDto | null>(null);
  const [previewing, setPreviewing] = useState(false);

  const [aiText, setAiText] = useState("");
  const [aiSpec, setAiSpec] = useState<SegmentFilter[] | null>(null);

  const [campaignNombre, setCampaignNombre] = useState("");
  const [campaignSegmentId, setCampaignSegmentId] = useState<number>(0);
  const [expandedRecipients, setExpandedRecipients] = useState<number | null>(null);

  if (isLoading) return <div className="text-gray-500">Cargando campañas...</div>;

  const runPreview = async (filters: SegmentFilter[]) => {
    setPreviewing(true);
    try {
      const res = await campaignRepository.previewSpec(filters);
      setPreview(res);
    } catch {
      toast.error("No se pudo calcular la vista previa.");
    } finally {
      setPreviewing(false);
    }
  };

  const handleSaveSegment = (name: string, filters: SegmentFilter[]) => {
    createSegment.mutate(
      { nombre: name, filter_spec: filters },
      {
        onSuccess: () => {
          setNombre("");
          setSpec([]);
          setPreview(null);
          setAiSpec(null);
          setAiText("");
          toast.success("Segmento guardado");
        },
      }
    );
  };

  const handleSaveAiSegment = () => {
    if (!aiSpec || !aiText.trim()) return;
    const name = aiText.trim().slice(0, 100);
    handleSaveSegment(name, aiSpec);
  };

  const handleLaunch = (campaign: CampaignDto) => {
    launchCampaign.mutate(campaign.id, {
      onSuccess: () => toast.success(`Campaña "${campaign.nombre}" lanzada: audiencia capturada`),
    });
  };

  return (
    <div className="space-y-8">
      {/* AI audience builder */}
      <section className="border rounded-lg p-4">
        <h2 className="font-semibold mb-2">Audiencia con IA</h2>
        <p className="text-sm text-gray-600 mb-3">
          Describe tu audiencia y la IA armará los filtros. Revisa antes de guardar.
        </p>
        <div className="flex gap-2">
          <input
            name="ai_descripcion"
            value={aiText}
            onChange={(e) => setAiText(e.target.value)}
            placeholder="Ej: clientes mayoristas que escribieron este mes"
            className="border rounded px-3 py-2 flex-1"
          />
          <button
            onClick={() => aiBuild.mutate(aiText, { onSuccess: (r) => setAiSpec(r.filter_spec) })}
            disabled={aiBuild.isPending || !aiText.trim()}
            className="bg-gray-900 text-white px-4 py-2 rounded text-sm disabled:opacity-50"
          >
            {aiBuild.isPending ? "Generando..." : "Generar audiencia"}
          </button>
        </div>
        {aiBuild.isError && (
          <p className="text-red-600 text-sm mt-2">{String(aiBuild.error ?? "Error generando la audiencia.")}</p>
        )}
        {aiSpec && (
          <div className="mt-3 border-t pt-3">
            <pre className="bg-gray-50 rounded p-3 text-xs overflow-x-auto">{JSON.stringify(aiSpec, null, 2)}</pre>
            <div className="flex items-center gap-2 mt-3">
              <button
                onClick={handleSaveAiSegment}
                className="bg-gray-900 text-white px-4 py-2 rounded text-sm"
              >
                Guardar como segmento
              </button>
              <button
                onClick={() => runPreview(aiSpec)}
                disabled={previewing}
                className="border px-4 py-2 rounded text-sm text-gray-600"
              >
                {previewing ? "Calculando..." : "Ver vista previa"}
              </button>
              {preview && <PreviewBadge preview={preview} />}
            </div>
          </div>
        )}
      </section>

      {/* Segments */}
      <section className="border rounded-lg p-4">
        <h2 className="font-semibold mb-2">Segmentos</h2>
        <div className="flex flex-wrap gap-2 mb-3">
          {PRESETS.map((p) => (
            <button
              key={p.label}
              onClick={() => {
                setSpec(p.spec);
                setPreview(null);
              }}
              className={`border rounded px-3 py-1 text-xs ${
                JSON.stringify(spec) === JSON.stringify(p.spec) ? "bg-gray-900 text-white" : "bg-white text-gray-700"
              }`}
            >
              {p.label}
            </button>
          ))}
        </div>
        <div className="flex flex-wrap items-center gap-2 mb-3">
          <input
            name="segmento_nombre"
            value={nombre}
            onChange={(e) => setNombre(e.target.value)}
            placeholder="Nombre del segmento"
            className="border rounded px-3 py-2 w-56"
          />
          <button
            onClick={() => runPreview(spec)}
            disabled={previewing || spec.length === 0}
            className="border px-4 py-2 rounded text-sm text-gray-600 disabled:opacity-50"
          >
            {previewing ? "Calculando..." : "Vista previa"}
          </button>
          {preview && <PreviewBadge preview={preview} />}
          <button
            onClick={() => handleSaveSegment(nombre, spec)}
            disabled={!nombre.trim() || spec.length === 0}
            className="bg-gray-900 text-white px-4 py-2 rounded text-sm disabled:opacity-50"
          >
            Guardar segmento
          </button>
        </div>
        <pre className="bg-gray-50 rounded p-3 text-xs overflow-x-auto">
          {spec.length > 0 ? JSON.stringify(spec, null, 2) : "Selecciona un filtro o usa la IA."}
        </pre>

        <ul className="mt-4 space-y-2">
          {(segments ?? []).map((s) => (
            <li key={s.id} className="flex items-center justify-between border-b pb-2">
              <div>
                <div className="font-medium">{s.nombre}</div>
                <div className="text-xs text-gray-500">
                  {s.filter_spec.map((f) => `${f.field} ${f.op} ${JSON.stringify(f.value)}`).join(" · ")}
                </div>
              </div>
              <div className="flex gap-2">
                <button
                  onClick={() => runPreview(s.filter_spec)}
                  className="text-xs border rounded px-2 py-1 text-gray-600"
                >
                  Vista previa
                </button>
                <button
                  onClick={() => deleteSegment.mutate(s.id)}
                  className="text-xs border rounded px-2 py-1 text-red-600"
                >
                  Eliminar
                </button>
              </div>
            </li>
          ))}
          {(segments ?? []).length === 0 && <li className="text-sm text-gray-500">Aún no hay segmentos.</li>}
        </ul>
      </section>

      {/* Campaigns */}
      <section className="border rounded-lg p-4">
        <h2 className="font-semibold mb-2">Campañas</h2>
        <div className="flex flex-wrap items-center gap-2 mb-4">
          <input
            name="campana_nombre"
            value={campaignNombre}
            onChange={(e) => setCampaignNombre(e.target.value)}
            placeholder="Nombre de la campaña"
            className="border rounded px-3 py-2 w-56"
          />
          <select
            name="campana_segmento"
            value={campaignSegmentId}
            onChange={(e) => setCampaignSegmentId(Number(e.target.value))}
            className="border rounded px-3 py-2"
          >
            <option value={0}>Selecciona un segmento</option>
            {(segments ?? []).map((s) => (
              <option key={s.id} value={s.id}>
                {s.nombre}
              </option>
            ))}
          </select>
          <button
            onClick={() =>
              createCampaign.mutate(
                { nombre: campaignNombre.trim(), segment_id: campaignSegmentId },
                {
                  onSuccess: () => {
                    setCampaignNombre("");
                    setCampaignSegmentId(0);
                    toast.success("Campaña creada");
                  },
                }
              )
            }
            disabled={!campaignNombre.trim() || campaignSegmentId === 0}
            className="bg-gray-900 text-white px-4 py-2 rounded text-sm disabled:opacity-50"
          >
            Crear campaña
          </button>
        </div>

        <ul className="space-y-2">
          {(campaigns ?? []).map((c) => (
            <li key={c.id} className="border rounded p-3">
              <div className="flex items-center justify-between">
                <div>
                  <div className="font-medium">{c.nombre}</div>
                  <div className="text-xs text-gray-500">
                    Estado: {c.status === "ready" ? "Lista (audiencia capturada)" : "Borrador"} ·{" "}
                    {c.recipient_count} destinatarios
                  </div>
                </div>
                <div className="flex gap-2">
                  {c.status === "draft" && (
                    <button
                      onClick={() => handleLaunch(c)}
                      disabled={launchCampaign.isPending}
                      className="bg-gray-900 text-white px-3 py-1 rounded text-xs disabled:opacity-50"
                    >
                      Lanzar
                    </button>
                  )}
                  <button
                    onClick={() =>
                      setExpandedRecipients(expandedRecipients === c.id ? null : c.id)
                    }
                    className="text-xs border rounded px-2 py-1 text-gray-600"
                  >
                    {expandedRecipients === c.id ? "Ocultar destinatarios" : "Destinatarios"}
                  </button>
                </div>
              </div>
              {expandedRecipients === c.id && <RecipientList campaignId={c.id} />}
            </li>
          ))}
          {(campaigns ?? []).length === 0 && (
            <li className="text-sm text-gray-500">Aún no hay campañas. Crea una para capturar su audiencia.</li>
          )}
        </ul>
      </section>
    </div>
  );
}

function PreviewBadge({ preview }: { preview: EvalResultDto }) {
  return (
    <span className="text-xs text-gray-600">
      <strong className="text-gray-900">{preview.total}</strong> contactos
      {preview.excluded_by_gates > 0 && (
        <span className="text-amber-600"> · {preview.excluded_by_gates} excluidos por consentimiento</span>
      )}
    </span>
  );
}

function RecipientList({ campaignId }: { campaignId: number }) {
  const { data: recipients, isLoading } = useCampaignRecipientsQuery(campaignId, { limit: 100 });
  if (isLoading) return <div className="text-xs text-gray-500 mt-2">Cargando...</div>;
  return (
    <ul className="mt-2 text-xs space-y-1">
      {(recipients ?? []).map((r) => (
        <li key={r.id} className="flex justify-between border-b py-1">
          <span>{r.display_name || r.phone_number}</span>
          <span className={r.status === "pending" ? "text-gray-500" : "text-green-600"}>{r.status}</span>
        </li>
      ))}
      {(recipients ?? []).length === 0 && <li className="text-gray-500">Sin destinatarios todavía.</li>}
    </ul>
  );
}
