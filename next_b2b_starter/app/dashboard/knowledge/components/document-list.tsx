"use client";

import { useState } from "react";
import { Trash2, FileText, RefreshCcw, MoreVertical } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import type { Document } from "@/lib/models/document.model";
import { DocumentHelpers } from "@/lib/models/document.model";
import { cn } from "@/lib/utils";

interface DocumentListProps {
  documents: Document[];
  isLoading?: boolean;
  isFetching?: boolean;
  onDelete: (documentId: number) => Promise<void>;
  onRefresh: () => void;
  compact?: boolean;
}

function DocumentCard({
  document,
  onDelete,
  compact,
}: {
  document: Document;
  onDelete: (doc: Document) => void;
  compact?: boolean;
}) {
  const statusConfig = DocumentHelpers.getStatusConfig(document.status);

  return (
    <div
      className={cn(
        "group relative flex items-start gap-3 rounded-lg border transition-all",
        compact
          ? "border-transparent bg-transparent p-2 hover:bg-violet-50"
          : "border-gray-200 bg-white p-4 hover:border-violet-200 hover:shadow-sm"
      )}
    >
      <div
        className={cn(
          "flex shrink-0 items-center justify-center rounded-lg",
          compact ? "h-8 w-8" : "h-10 w-10"
        )}
        style={{ backgroundColor: "#fef2f2" }}
      >
        <FileText className={cn(compact ? "h-4 w-4" : "h-5 w-5")} style={{ color: "#ef4444" }} />
      </div>

      <div className="min-w-0 flex-1">
        <div className="flex items-start justify-between gap-2">
          <p className="truncate font-medium text-sm" style={{ color: "#111827" }}>{document.title}</p>
          {compact && (
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-6 w-6 shrink-0 text-gray-400 opacity-0 group-hover:opacity-100"
                >
                  <MoreVertical className="h-3 w-3" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem
                  className="text-red-600"
                  onClick={() => onDelete(document)}
                >
                  <Trash2 className="mr-2 h-4 w-4" />
                  Delete
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          )}
        </div>
        <p className="truncate text-xs" style={{ color: "#6b7280" }}>{document.fileName}</p>

        {!compact && (
          <div className="mt-4 flex items-center justify-between border-t border-gray-100 pt-3">
            <div className="flex items-center gap-2">
              <Badge
                  variant="outline"
                  className={cn(
                  "text-xs",
                  statusConfig.color,
                  statusConfig.bgColor
                  )}
              >
                  {statusConfig.label}
              </Badge>
            </div>
            <div className="flex items-center gap-2">
              <span className="text-xs" style={{ color: "#9ca3af" }}>
                {DocumentHelpers.formatFileSize(document.fileSize)}
              </span>
              <Button
                variant="ghost"
                size="icon"
                onClick={() => onDelete(document)}
                className="h-8 w-8 opacity-0 transition-opacity group-hover:opacity-100"
                style={{ color: "#9ca3af" }}
              >
                <Trash2 className="h-4 w-4" />
              </Button>
            </div>
          </div>
        )}
        
        {compact && (
           <div className="mt-1 flex items-center gap-2">
              <div className="h-1.5 w-1.5 rounded-full" style={{ backgroundColor: document.status === 'processed' ? "#10b981" : "#eab308" }} />
              <span className="text-[10px]" style={{ color: "#9ca3af" }}>
                {DocumentHelpers.formatFileSize(document.fileSize)}
              </span>
           </div>
        )}
      </div>
    </div>
  );
}

function DocumentListSkeleton({ compact }: { compact?: boolean }) {
  return (
    <div className={cn("grid gap-4", compact ? "grid-cols-1" : "sm:grid-cols-2 lg:grid-cols-3")}>
        {Array.from({ length: 6 }).map((_, i) => (
            <div key={i} className={cn("rounded-xl p-4", compact ? "flex gap-3" : "border border-gray-200 bg-white")}>
                <Skeleton className={cn("rounded-lg", compact ? "h-8 w-8" : "h-10 w-10")} />
                <div className="flex-1 space-y-2">
                    <Skeleton className="h-4 w-3/4" />
                    <Skeleton className="h-3 w-1/2" />
                </div>
            </div>
        ))}
    </div>
  );
}

function EmptyDocumentsState({ compact }: { compact?: boolean }) {
  if (compact) {
     return (
        <div className="flex flex-col items-center justify-center py-8 text-center rounded-xl border border-dashed" style={{ backgroundColor: "rgba(249,250,251,0.5)", borderColor: "#e5e7eb" }}>
             <div className="p-2 rounded-full mb-2" style={{ backgroundColor: "#f3f4f6" }}>
                <FileText className="h-4 w-4" style={{ color: "#9ca3af" }} />
             </div>
             <p className="text-xs font-medium" style={{ color: "#6b7280" }}>No documents yet</p>
        </div>
     )
  }
  return (
    <div className="flex flex-col items-center justify-center rounded-2xl border-2 border-dashed px-6 py-16" style={{ borderColor: "#e5e7eb", backgroundColor: "rgba(249,250,251,0.5)" }}>
      <div className="flex h-16 w-16 items-center justify-center rounded-full" style={{ backgroundColor: "#f3f4f6" }}>
        <FileText className="h-8 w-8" style={{ color: "#9ca3af" }} />
      </div>
      <h3 className="mt-4 text-sm font-semibold" style={{ color: "#111827" }}>
        No documents yet
      </h3>
      <p className="mt-1 max-w-sm text-center text-sm" style={{ color: "#6b7280" }}>
        Upload your first PDF document to start building your knowledge base for
        AI-powered search.
      </p>
    </div>
  );
}

export function DocumentList({
  documents,
  isLoading = false,
  isFetching = false,
  onDelete,
  onRefresh,
  compact = false,
}: DocumentListProps) {
  const [deleteTarget, setDeleteTarget] = useState<Document | null>(null);
  const [isDeleting, setIsDeleting] = useState(false);

  const handleDelete = async () => {
    if (!deleteTarget) return;
    setIsDeleting(true);
    try {
      await onDelete(deleteTarget.id);
      setDeleteTarget(null);
    } finally {
      setIsDeleting(false);
    }
  };

  if (isLoading) {
    return <DocumentListSkeleton compact={compact} />;
  }

  if (documents.length === 0) {
    return <EmptyDocumentsState compact={compact} />;
  }

  return (
    <>
      <div className="mb-4 flex items-center justify-between">
        <p className="text-sm font-medium" style={{ color: "#4b5563" }}>
           {compact ? "Sources" : `${documents.length} document${documents.length !== 1 ? "s" : ""}`}
        </p>
        <Button
          variant="ghost"
          size="sm"
          onClick={onRefresh}
          disabled={isFetching}
          className="h-8 w-8 p-0"
          style={{ color: "#4b5563" }}
        >
          <RefreshCcw
            className={cn("h-4 w-4", isFetching && "animate-spin")}
          />
        </Button>
      </div>

      <div className={cn("grid gap-2", compact ? "grid-cols-1" : "sm:grid-cols-2 lg:grid-cols-3")}>
        {documents.map((doc) => (
          <DocumentCard key={doc.id} document={doc} onDelete={setDeleteTarget} compact={compact} />
        ))}
      </div>

      <Dialog open={!!deleteTarget} onOpenChange={() => setDeleteTarget(null)}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Delete Document</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete &quot;{deleteTarget?.title}&quot;?
              This action cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter className="gap-2 sm:gap-0">
            <Button
              variant="outline"
              onClick={() => setDeleteTarget(null)}
              disabled={isDeleting}
            >
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={handleDelete}
              disabled={isDeleting}
            >
              {isDeleting ? "Deleting..." : "Delete"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}

