"use client";

import { useState } from "react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Download, Trash2, ShieldCheck, Loader2 } from "lucide-react";
import { toast } from "sonner";
import { agentRepository } from "@/lib/api/api/repositories/agent-repository";

export function ComplianceSection() {
  const [contactId, setContactId] = useState("");
  const [isExporting, setIsExporting] = useState(false);
  const [isForgetting, setIsForgetting] = useState(false);
  const [exportResult, setExportResult] = useState<{ contactName: string; conversations: number } | null>(null);

  const handleExport = async () => {
    const id = Number(contactId);
    if (!Number.isInteger(id) || id <= 0) {
      toast.error("Ingresa un ID de contacto válido");
      return;
    }
    setIsExporting(true);
    try {
      const bundle = await agentRepository.exportContact(id);
      const blob = new Blob([JSON.stringify(bundle, null, 2)], { type: "application/json" });
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `contacto-${id}-export.json`;
      a.click();
      URL.revokeObjectURL(url);
      setExportResult({
        contactName: bundle.contact.display_name || bundle.contact.phone_number,
        conversations: bundle.conversations.length,
      });
      toast.success("Exportación generada (Habeas Data)");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Error al exportar el contacto");
    } finally {
      setIsExporting(false);
    }
  };

  const handleForget = async () => {
    const id = Number(contactId);
    if (!Number.isInteger(id) || id <= 0) {
      toast.error("Ingresa un ID de contacto válido");
      return;
    }
    if (!window.confirm("¿Anonimizar todos los datos personales de este contacto? Esta acción no se puede deshacer.")) {
      return;
    }
    setIsForgetting(true);
    try {
      await agentRepository.forgetContact(id);
      toast.success("Contacto anonimizado correctamente");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Error al anonimizar el contacto");
    } finally {
      setIsForgetting(false);
    }
  };

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <div className="flex items-center gap-3">
            <ShieldCheck className="h-6 w-6 text-gray-600" />
            <div>
              <CardTitle>Habeas Data (Ley 1581)</CardTitle>
              <CardDescription>
                Gestión de consentimiento, exportación y supresión de datos personales de contactos.
              </CardDescription>
            </div>
          </div>
        </CardHeader>
        <CardContent className="space-y-6">
          <Alert className="border border-blue-200 bg-blue-50">
            <AlertTitle>Qué hace el asistente por ti</AlertTitle>
            <AlertDescription>
              <ul className="ml-4 list-disc space-y-1 text-sm">
                <li>Solicita autorización de tratamiento de datos al primer contacto.</li>
                <li>Enmascara documentos, teléfonos y nombres antes de enviarlos a modelos de IA.</li>
                <li>Bloquea respuestas autónomas a contactos sin consentimiento.</li>
              </ul>
            </AlertDescription>
          </Alert>

          <div className="space-y-2">
            <Label className="text-sm font-medium">ID del contacto</Label>
            <Input
              type="number"
              placeholder="Ej: 42"
              value={contactId}
              onChange={(e) => setContactId(e.target.value)}
            />
            <p className="text-xs text-gray-500">
              El ID aparece en la URL de la conversación del CRM (contacto) o en la lista de contactos.
            </p>
          </div>

          {exportResult && (
            <Alert className="border border-emerald-200 bg-emerald-50">
              <AlertDescription>
                Exportación de {exportResult.contactName} generada: {exportResult.conversations} conversaciones incluidas.
              </AlertDescription>
            </Alert>
          )}

          <div className="flex flex-col gap-3 sm:flex-row">
            <Button
              onClick={handleExport}
              disabled={isExporting || !contactId}
              className="bg-gray-900 text-white hover:bg-gray-800"
            >
              {isExporting ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : <Download className="mr-2 h-4 w-4" />}
              Exportar datos del contacto
            </Button>
            <Button
              onClick={handleForget}
              disabled={isForgetting || !contactId}
              variant="outline"
              className="text-red-600 hover:bg-red-50"
            >
              {isForgetting ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : <Trash2 className="mr-2 h-4 w-4" />}
              Derecho al olvido (anonimizar)
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
