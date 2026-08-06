"use client";

import { useState } from "react";
import { FileText, ChevronDown } from "lucide-react";
import type { SimilarDocument } from "@/lib/models/cognitive.model";
import { cn } from "@/lib/utils";

interface DocumentSourcesProps {
  sources: SimilarDocument[];
}

export function DocumentSources({ sources }: DocumentSourcesProps) {
  const [isOpen, setIsOpen] = useState(false);

  if (!sources || sources.length === 0) return null;

  return (
    <div className="mt-2 overflow-hidden rounded-xl border" style={{ borderColor: "#e5e7eb", backgroundColor: "white" }}>
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="flex w-full items-center justify-between px-3 py-2 text-xs transition-colors hover:bg-gray-50"
        style={{ color: "#4b5563" }}
      >
        <span className="flex items-center gap-1.5">
          <FileText className="h-3.5 w-3.5" />
          {sources.length} source{sources.length !== 1 ? "s" : ""} referenced
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
          {sources.map((source) => (
            <div key={source.id} className="rounded-lg p-3" style={{ backgroundColor: "#f9fafb" }}>
              <div className="flex items-center justify-between">
                <span className="text-xs font-medium" style={{ color: "#374151" }}>
                  Document #{source.documentId}
                </span>
                <div className="flex items-center gap-2">
                  <div className="h-1.5 w-14 overflow-hidden rounded-full" style={{ backgroundColor: "#e5e7eb" }}>
                    <div
                      className="h-full rounded-full transition-all"
                      style={{ width: `${source.similarityScore * 100}%`, backgroundColor: "#10b981" }}
                    />
                  </div>
                  <span className="text-xs font-medium" style={{ color: "#6b7280" }}>
                    {Math.round(source.similarityScore * 100)}%
                  </span>
                </div>
              </div>
              <p className="mt-2 line-clamp-3 text-xs leading-relaxed" style={{ color: "#4b5563" }}>
                {source.contentPreview}
              </p>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
