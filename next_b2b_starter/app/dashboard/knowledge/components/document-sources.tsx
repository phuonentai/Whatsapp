"use client";

import { useState } from "react";
import {
  FileText,
  FileImage,
  FileVideo,
  FileAudio,
  FileSpreadsheet,
  FileArchive,
  ChevronDown,
  ExternalLink,
} from "lucide-react";
import type { SimilarDocument } from "@/lib/models/cognitive.model";
import type { Document } from "@/lib/models/document.model";
import { DocumentHelpers } from "@/lib/models/document.model";
import { useDocuments } from "@/lib/hooks/queries/use-documents-query";
import { cn } from "@/lib/utils";
import { ui, tpl } from "@/lib/copy/ui";

interface DocumentSourcesProps {
  sources: SimilarDocument[];
  onSourceClick?: (documentId: number) => void;
}

function documentIcon(doc: Document | undefined) {
  const contentType = doc?.contentType ?? "";
  const fileName = doc?.fileName ?? "";
  const name = fileName.toLowerCase();

  if (contentType.startsWith("image/")) return FileImage;
  if (contentType.startsWith("video/")) return FileVideo;
  if (contentType.startsWith("audio/")) return FileAudio;
  if (contentType.includes("spreadsheet") || /\.(csv|xlsx?|ods)$/.test(name)) return FileSpreadsheet;
  if (contentType.includes("zip") || contentType.includes("archive") || /\.(zip|rar|7z|tar|gz)$/.test(name)) return FileArchive;
  return FileText;
}

export function DocumentSources({ sources, onSourceClick }: DocumentSourcesProps) {
  const [isOpen, setIsOpen] = useState(false);
  const documents = useDocuments();

  if (!sources || sources.length === 0) return null;

  const documentsById = new Map<number, Document>(
    documents.map((doc) => [doc.id, doc])
  );
  const countLabel =
    sources.length === 1
      ? ui.knowledge.referencedSource
      : tpl(ui.knowledge.referencedSources, { n: sources.length });

  return (
    <div className="mt-2 overflow-hidden rounded-xl border" style={{ borderColor: "#e5e7eb", backgroundColor: "white" }}>
      <button
        onClick={() => setIsOpen(!isOpen)}
        aria-expanded={isOpen}
        className="flex w-full items-center justify-between px-3 py-2 text-xs transition-colors hover:bg-gray-50"
        style={{ color: "#4b5563" }}
      >
        <span className="flex items-center gap-1.5">
          <FileText className="h-3.5 w-3.5" />
          {countLabel}
        </span>
        <ChevronDown
          className={cn(
            "h-3.5 w-3.5 transition-transform duration-200",
            isOpen && "rotate-180"
          )}
        />
      </button>

      {isOpen && (
        <div className="space-y-2 border-t p-3" style={{ borderColor: "#f3f4f6" }}>
          {sources.map((source, index) => {
            const doc = documentsById.get(source.documentId);
            const Icon = documentIcon(doc);
            const available = Boolean(doc);
            const inner = (
              <>
                <div className="flex items-center justify-between gap-2">
                  <span className="flex min-w-0 items-center gap-1.5 text-xs font-medium" style={{ color: "#374151" }}>
                    <Icon className="h-3.5 w-3.5 flex-shrink-0" style={{ color: "#7c3aed" }} />
                    <span className="truncate">
                      {available ? doc!.title : ui.knowledge.documentUnavailable}
                    </span>
                  </span>
                  <span className="flex items-center gap-2 flex-shrink-0">
                    <span className="text-xs font-medium" style={{ color: "#6b7280" }}>
                      {DocumentHelpers.getSimilarityLabel(source.similarityScore)}
                    </span>
                    {available && onSourceClick && (
                      <ExternalLink className="h-3 w-3" style={{ color: "#9ca3af" }} />
                    )}
                  </span>
                </div>
                <p className="mt-2 line-clamp-3 text-xs leading-relaxed" style={{ color: "#4b5563" }}>
                  {source.contentPreview}
                </p>
              </>
            );

            const card = (
              <div className="rounded-lg p-3" style={{ backgroundColor: "#f9fafb" }}>
                {inner}
              </div>
            );

            return available && onSourceClick ? (
              <button
                key={source.id}
                id={`fuente-${index + 1}`}
                onClick={() => onSourceClick(source.documentId)}
                className="block w-full text-left rounded-lg p-3 transition-colors hover:bg-violet-50 focus-visible:outline focus-visible:outline-2 focus-visible:outline-violet-500"
                style={{ backgroundColor: "#f9fafb" }}
                title={doc!.title}
              >
                {inner}
              </button>
            ) : (
              <div key={source.id} id={`fuente-${index + 1}`}>{card}</div>
            );
          })}
        </div>
      )}
    </div>
  );
}
