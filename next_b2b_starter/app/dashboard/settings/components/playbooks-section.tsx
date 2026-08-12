"use client";

import { useState } from "react";
import { toast } from "sonner";
import { Store, CheckCircle2, RotateCcw } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { ErrorState } from "@/components/common/error-state";
import { usePlaybooksQuery } from "@/lib/hooks/queries/use-playbooks-query";
import { useApplyPlaybook, useResetPlaybook } from "@/lib/hooks/mutations/use-playbook-mutations";
import type { PlaybookDto } from "@/lib/api/api/dto/playbook.dto";

export function PlaybookSetupCard() {
  const { data: playbooks, isLoading, isError, refetch, isRefetching } = usePlaybooksQuery();
  const applyPlaybook = useApplyPlaybook();
  const resetPlaybook = useResetPlaybook();
  const [confirmReset, setConfirmReset] = useState<string | null>(null);

  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Plantillas de negocio</CardTitle>
        </CardHeader>
        <CardContent className="text-sm text-slate-500">Cargando plantillas...</CardContent>
      </Card>
    );
  }

  if (isError) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Plantillas de negocio</CardTitle>
        </CardHeader>
        <CardContent>
          <ErrorState
            title="Error al cargar las plantillas"
            description="No se pudieron cargar las plantillas. Inténtalo de nuevo."
            onRetry={() => refetch()}
            isRetrying={isRefetching}
          />
        </CardContent>
      </Card>
    );
  }

  const catalog = playbooks ?? [];
  const applied = catalog.filter((p) => p.applied);

  if (applied.length === 0 && catalog.length === 0) {
    return null;
  }

  const handleApply = async (key: string) => {
    try {
      await applyPlaybook.mutateAsync(key);
      toast.success("Plantilla aplicada", {
        description: "Tu pipeline, etiquetas y guiones están listos para usar.",
      });
    } catch (error) {
      toast.error("No se pudo aplicar la plantilla", {
        description: error instanceof Error ? error.message : undefined,
      });
    }
  };

  const handleReset = async (key: string) => {
    try {
      await resetPlaybook.mutateAsync(key);
      setConfirmReset(null);
      toast.success("Plantilla reiniciada", {
        description: "Se eliminaron los datos sembrados por la plantilla.",
      });
    } catch (error) {
      toast.error("No se pudo reiniciar la plantilla", {
        description: error instanceof Error ? error.message : undefined,
      });
    }
  };

  return (
    <Card className="border-slate-200">
      <CardHeader>
        <div className="flex items-center gap-2">
          <Store className="h-5 w-5 text-slate-600" />
          <CardTitle className="text-base">
            {applied.length > 0 ? "Tu plantilla de negocio" : "¿Qué tipo de negocio es el tuyo?"}
          </CardTitle>
        </div>
        <CardDescription>
          {applied.length > 0
            ? "Pipeline, etiquetas y guiones listos para tu operación."
            : "Elige una plantilla y tu espacio queda listo en un clic: pipeline por vertical, etiquetas, configuración de módulos y respuestas rápidas para WhatsApp."}
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-3">
        {catalog.map((playbook) => (
          <PlaybookRow
            key={playbook.key}
            playbook={playbook}
            onApply={handleApply}
            onReset={handleReset}
            isApplying={applyPlaybook.isPending}
            confirmReset={confirmReset}
            setConfirmReset={setConfirmReset}
          />
        ))}
      </CardContent>
    </Card>
  );
}

function PlaybookRow({
  playbook,
  onApply,
  onReset,
  isApplying,
  confirmReset,
  setConfirmReset,
}: {
  playbook: PlaybookDto;
  onApply: (key: string) => void;
  onReset: (key: string) => void;
  isApplying: boolean;
  confirmReset: string | null;
  setConfirmReset: (key: string | null) => void;
}) {
  const isConfirming = confirmReset === playbook.key;

  return (
    <div className="flex items-start justify-between gap-4 rounded-xl border border-slate-100 bg-slate-50/60 p-4">
      <div className="min-w-0 space-y-1">
        <div className="flex items-center gap-2">
          <p className="text-sm font-semibold text-slate-900">{playbook.name}</p>
          {playbook.applied ? (
            <Badge variant="default" className="bg-green-600">
              <CheckCircle2 className="mr-1 h-3 w-3" /> Activa
            </Badge>
          ) : null}
        </div>
        <p className="text-sm text-slate-600 line-clamp-2">{playbook.description}</p>
        {playbook.applied && playbook.guiones && playbook.guiones.length > 0 ? (
          <p className="text-xs text-slate-500">
            {playbook.guiones.length} guiones listos en la bandeja de entrada
          </p>
        ) : null}
      </div>
      <div className="flex shrink-0 items-center gap-2">
        {playbook.applied ? (
          isConfirming ? (
            <>
              <Button
                variant="destructive"
                size="sm"
                onClick={() => onReset(playbook.key)}
                disabled={isApplying}
              >
                Confirmar
              </Button>
              <Button variant="ghost" size="sm" onClick={() => setConfirmReset(null)}>
                Cancelar
              </Button>
            </>
          ) : (
            <Button
              variant="outline"
              size="sm"
              onClick={() => setConfirmReset(playbook.key)}
              disabled={isApplying}
            >
              <RotateCcw className="mr-1 h-3 w-3" /> Reiniciar
            </Button>
          )
        ) : (
          <Button
            size="sm"
            className="bg-emerald-500 text-white hover:bg-emerald-600"
            onClick={() => onApply(playbook.key)}
            disabled={isApplying}
          >
            Usar plantilla
          </Button>
        )}
      </div>
    </div>
  );
}
