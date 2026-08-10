"use client";

import { useRef, useState } from "react";
import { toast } from "sonner";
import { Loader2, Upload, Download } from "lucide-react";
import { crmRepository, type ImportSummaryDto } from "@/lib/api/api/repositories/crm-repository";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";

interface ContactImportDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function ContactImportDialog({ open, onOpenChange }: ContactImportDialogProps) {
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [file, setFile] = useState<File | null>(null);
  const [isImporting, setIsImporting] = useState(false);
  const [summary, setSummary] = useState<ImportSummaryDto | null>(null);

  const handleTemplateDownload = async () => {
    try {
      await crmRepository.downloadImportTemplate();
      toast.success("Plantilla descargada");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Error al descargar la plantilla");
    }
  };

  const handleImport = async () => {
    if (!file) {
      toast.error("Selecciona un archivo CSV");
      return;
    }
    setIsImporting(true);
    setSummary(null);
    try {
      const result = await crmRepository.importContacts(file);
      setSummary(result);
      setFile(null);
      if (fileInputRef.current) fileInputRef.current.value = "";
      toast.success("Importación completada");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Error al importar contactos");
    } finally {
      setIsImporting(false);
    }
  };

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!isImporting) {
          if (!next) {
            setSummary(null);
            setFile(null);
          }
          onOpenChange(next);
        }
      }}
    >
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Importar contactos</DialogTitle>
          <DialogDescription>
            Carga un archivo CSV con el formato de la plantilla. Los teléfonos ya existentes se omiten sin modificar.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <Button type="button" variant="outline" onClick={handleTemplateDownload} className="w-full">
            <Download className="mr-2 h-4 w-4" />
            Descargar plantilla
          </Button>

          <input
            ref={fileInputRef}
            type="file"
            accept=".csv,text/csv"
            onChange={(e) => setFile(e.target.files?.[0] ?? null)}
            className="block w-full text-sm text-gray-600 file:mr-3 file:rounded file:border-0 file:bg-blue-50 file:px-3 file:py-2 file:text-sm file:font-medium file:text-blue-700 hover:file:bg-blue-100"
          />

          {summary && (
            <div className="rounded border p-3 text-sm space-y-1">
              <p className="font-medium">Resumen de la importación</p>
              <p className="text-green-700">Importados: {summary.importados}</p>
              <p className="text-yellow-700">Omitidos (teléfono duplicado): {summary.omitidos}</p>
              {summary.errores.length > 0 && (
                <div className="mt-2">
                  <p className="text-red-700">Errores: {summary.errores.length}</p>
                  <ul className="ml-4 list-disc text-xs text-red-600 space-y-1">
                    {summary.errores.map((e) => (
                      <li key={e.fila}>
                        Fila {e.fila}: {e.razon}
                      </li>
                    ))}
                  </ul>
                </div>
              )}
            </div>
          )}
        </div>

        <DialogFooter className="gap-2 sm:gap-0">
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)} disabled={isImporting}>
            Cerrar
          </Button>
          <Button type="button" onClick={handleImport} disabled={isImporting || !file}>
            {isImporting ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : <Upload className="mr-2 h-4 w-4" />}
            {isImporting ? "Importando..." : "Importar"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
