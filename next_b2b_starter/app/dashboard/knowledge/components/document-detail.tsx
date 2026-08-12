"use client";

import { useState } from "react";
import {
  FileText,
  Lock,
  RotateCw,
  Pencil,
  Trash2,
  Loader2,
  ArrowLeft,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Alert, AlertDescription } from "@/components/ui/alert";
import type { Document, DocumentVisibility } from "@/lib/models/document.model";
import { DocumentHelpers } from "@/lib/models/document.model";
import { ui } from "@/lib/copy/ui";
import { cn } from "@/lib/utils";

interface DocumentDetailProps {
  document: Document;
  canManage?: boolean;
  onRename: (documentId: number, title: string) => Promise<void>;
  onReprocess: (documentId: number) => Promise<void>;
  onDelete: (documentId: number) => Promise<void>;
  onChangeVisibility: (documentId: number, visibility: DocumentVisibility) => Promise<void>;
  onBack?: () => void;
}

/** Restricted state: shown to members without org:manage when they open a link
 *  to an admin_only document. NEVER reveals the document title. */
export function RestrictedDocumentState({ onBack }: { onBack?: () => void }) {
  return (
    <div className="flex h-full flex-col items-center justify-center p-8 text-center">
      <div className="flex h-14 w-14 items-center justify-center rounded-full" style={{ backgroundColor: "#fef2f2" }}>
        <Lock className="h-6 w-6" style={{ color: "#dc2626" }} />
      </div>
      <h2 className="mt-4 text-base font-semibold" style={{ color: "#111827" }}>
        {ui.knowledge.documentRestricted}
      </h2>
      <p className="mt-2 max-w-sm text-sm" style={{ color: "#6b7280" }}>
        {ui.knowledge.documentRestrictedDesc}
      </p>
      {onBack && (
        <Button variant="outline" size="sm" className="mt-6 gap-2" onClick={onBack}>
          <ArrowLeft className="h-4 w-4" />
          {ui.knowledge.modeDocs}
        </Button>
      )}
    </div>
  );
}

export function DocumentDetail({
  document,
  canManage = true,
  onRename,
  onReprocess,
  onDelete,
  onChangeVisibility,
  onBack,
}: DocumentDetailProps) {
  const statusConfig = DocumentHelpers.getStatusConfig(document.status);
  const [isRenaming, setIsRenaming] = useState(false);
  const [renameValue, setRenameValue] = useState("");
  const [renameOpen, setRenameOpen] = useState(false);
  const [reprocessOpen, setReprocessOpen] = useState(false);
  const [isReprocessing, setIsReprocessing] = useState(false);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);

  const isAdminOnly = document.visibility === "admin_only";

  const handleRename = async () => {
    if (!renameValue.trim()) return;
    setIsRenaming(true);
    try {
      await onRename(document.id, renameValue.trim());
      setRenameOpen(false);
    } finally {
      setIsRenaming(false);
    }
  };

  const handleReprocess = async () => {
    setIsReprocessing(true);
    try {
      await onReprocess(document.id);
      setReprocessOpen(false);
    } finally {
      setIsReprocessing(false);
    }
  };

  const handleDelete = async () => {
    setIsDeleting(true);
    try {
      await onDelete(document.id);
      setDeleteOpen(false);
    } finally {
      setIsDeleting(false);
    }
  };

  return (
    <div className="flex h-full flex-col min-h-0">
      <div className="h-14 px-4 border-b border-gray-200 flex items-center justify-between flex-shrink-0 bg-white">
        <div className="flex min-w-0 items-center gap-3">
          {onBack && (
            <Button variant="ghost" size="icon" className="h-8 w-8" onClick={onBack} aria-label={ui.knowledge.modeDocs}>
              <ArrowLeft className="h-4 w-4" />
            </Button>
          )}
          <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg" style={{ backgroundColor: "#fef2f2" }}>
            <FileText className="h-4 w-4" style={{ color: "#ef4444" }} />
          </div>
          <p className="truncate text-sm font-medium" style={{ color: "#111827" }}>{document.title}</p>
        </div>
        <div className="flex items-center gap-2">
          {isAdminOnly && (
            <span
              className="flex items-center gap-1 rounded-full px-2 py-1 text-[10px] font-medium"
              style={{ backgroundColor: "#f3f4f6", color: "#4b5563" }}
            >
              <Lock className="h-3 w-3" />
              {ui.knowledge.visibilityAdminOnly}
            </span>
          )}
          {canManage && (
            <>
              <Button variant="outline" size="sm" className="h-8 gap-1.5 text-xs" onClick={() => { setRenameValue(document.title); setRenameOpen(true); }}>
                <Pencil className="h-3.5 w-3.5" />
                {ui.knowledge.rename}
              </Button>
              <Button variant="outline" size="sm" className="h-8 gap-1.5 text-xs" onClick={() => setReprocessOpen(true)}>
                <RotateCw className="h-3.5 w-3.5" />
                {ui.knowledge.reprocess}
              </Button>
              <Button variant="destructive" size="sm" className="h-8 gap-1.5 text-xs" onClick={() => setDeleteOpen(true)}>
                <Trash2 className="h-3.5 w-3.5" />
                {ui.knowledge.delete}
              </Button>
            </>
          )}
        </div>
      </div>

      <div className="flex-1 min-h-0 overflow-y-auto p-6">
        <div className="space-y-4">
          <div className="rounded-xl border border-gray-200 bg-white p-5">
            <h3 className="text-sm font-semibold" style={{ color: "#111827" }}>Metadata</h3>
            <dl className="mt-3 space-y-2 text-sm">
              <div className="flex items-center justify-between gap-4">
                <dt style={{ color: "#6b7280" }}>Archivo</dt>
                <dd className="truncate" style={{ color: "#111827" }}>{document.fileName}</dd>
              </div>
              <div className="flex items-center justify-between gap-4">
                <dt style={{ color: "#6b7280" }}>Tamaño</dt>
                <dd style={{ color: "#111827" }}>{DocumentHelpers.formatFileSize(document.fileSize)}</dd>
              </div>
              <div className="flex items-center justify-between gap-4">
                <dt style={{ color: "#6b7280" }}>Subido</dt>
                <dd style={{ color: "#111827" }}>{DocumentHelpers.formatDate(document.createdAt)}</dd>
              </div>
              <div className="flex items-center justify-between gap-4">
                <dt style={{ color: "#6b7280" }}>Estado</dt>
                <dd>
                  <Badge variant="outline" className={cn("text-xs", statusConfig.color, statusConfig.bgColor)}>
                    {statusConfig.label}
                  </Badge>
                </dd>
              </div>
              <div className="flex items-center justify-between gap-4">
                <dt style={{ color: "#6b7280" }}>Visibilidad</dt>
                <dd className="flex items-center gap-1.5" style={{ color: "#111827" }}>
                  {isAdminOnly && <Lock className="h-3 w-3" style={{ color: "#4b5563" }} />}
                  {DocumentHelpers.formatVisibility(document.visibility)}
                </dd>
              </div>
            </dl>

            {canManage && (
              <div className="mt-5 border-t border-gray-100 pt-4">
                <Label htmlFor="doc-visibility" className="text-xs" style={{ color: "#6b7280" }}>
                  {ui.knowledge.visibilityWorkspace} / {ui.knowledge.visibilityAdminOnly}
                </Label>
                <div className="mt-2 flex gap-2">
                  <Button
                    variant={!isAdminOnly ? "default" : "outline"}
                    size="sm"
                    className={cn("h-8 text-xs", !isAdminOnly && "bg-gray-900 hover:bg-gray-800")}
                    onClick={() => onChangeVisibility(document.id, "workspace")}
                  >
                    {ui.knowledge.visibilityWorkspace}
                  </Button>
                  <Button
                    variant={isAdminOnly ? "default" : "outline"}
                    size="sm"
                    className={cn("h-8 text-xs", isAdminOnly && "bg-gray-900 hover:bg-gray-800")}
                    onClick={() => onChangeVisibility(document.id, "admin_only")}
                  >
                    {ui.knowledge.visibilityAdminOnly}
                  </Button>
                </div>
                <p className="mt-2 text-xs" style={{ color: "#9ca3af" }}>
                  {ui.knowledge.visibilityAdminOnly}
                  {" — "}
                  {ui.knowledge.documentRestrictedDesc}
                </p>
              </div>
            )}
          </div>

          {document.status === "failed" && canManage && (
            <Alert variant="default" className="border-amber-200 bg-amber-50">
              <AlertDescription className="text-sm text-amber-900">
                {ui.knowledge.reprocess}
              </AlertDescription>
            </Alert>
          )}
        </div>
      </div>

      {/* Rename dialog */}
      <Dialog open={renameOpen} onOpenChange={setRenameOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>{ui.knowledge.rename}</DialogTitle>
          </DialogHeader>
          <div className="space-y-2">
            <Label htmlFor="rename-title">{ui.knowledge.modeDocs}</Label>
            <Input
              id="rename-title"
              value={renameValue}
              onChange={(e) => setRenameValue(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") handleRename();
              }}
            />
          </div>
          <DialogFooter className="gap-2 sm:gap-0">
            <Button variant="outline" onClick={() => setRenameOpen(false)} disabled={isRenaming}>
              Cancelar
            </Button>
            <Button onClick={handleRename} disabled={isRenaming || !renameValue.trim()} className="bg-gray-900 hover:bg-gray-800">
              {isRenaming ? "Guardando…" : ui.knowledge.rename}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Reprocess dialog */}
      <Dialog open={reprocessOpen} onOpenChange={setReprocessOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>{ui.knowledge.reprocess}</DialogTitle>
            <DialogDescription>
              ¿Reprocesar &quot;{document.title}&quot;? Se volverá a extraer el texto y se regenerarán los chunks (no se vuelve a subir el archivo).
            </DialogDescription>
          </DialogHeader>
          <DialogFooter className="gap-2 sm:gap-0">
            <Button variant="outline" onClick={() => setReprocessOpen(false)} disabled={isReprocessing}>
              Cancelar
            </Button>
            <Button onClick={handleReprocess} disabled={isReprocessing} className="gap-2 bg-gray-900 hover:bg-gray-800">
              {isReprocessing ? <Loader2 className="h-4 w-4 animate-spin" /> : <RotateCw className="h-4 w-4" />}
              {ui.knowledge.reprocess}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete dialog */}
      <Dialog open={deleteOpen} onOpenChange={setDeleteOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>{ui.knowledge.delete}</DialogTitle>
            <DialogDescription>
              ¿Eliminar &quot;{document.title}&quot;? Esta acción no se puede deshacer y las citas a este documento dejarán de estar disponibles.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter className="gap-2 sm:gap-0">
            <Button variant="outline" onClick={() => setDeleteOpen(false)} disabled={isDeleting}>
              Cancelar
            </Button>
            <Button variant="destructive" onClick={handleDelete} disabled={isDeleting}>
              {isDeleting ? "Eliminando…" : ui.knowledge.delete}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
