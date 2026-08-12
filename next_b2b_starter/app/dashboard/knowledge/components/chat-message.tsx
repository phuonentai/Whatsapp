"use client";

import { Sparkles } from "lucide-react";
import type {
  ChatMessage as ChatMessageType,
  SimilarDocument,
} from "@/lib/models/cognitive.model";
import { ChatHelpers } from "@/lib/models/cognitive.model";
import { Markdown } from "@/components/common/markdown";
import { DocumentSources } from "./document-sources";
import { tpl, ui } from "@/lib/copy/ui";

interface ChatMessageProps {
  message: ChatMessageType;
  sources?: SimilarDocument[];
  streaming?: boolean;
  onSourceClick?: (documentId: number) => void;
}

/** Rewrites [n] markers into citation-anchor links (#fuente-n) so the shared
 *  markdown renderer styles them as chips. Out-of-range markers stay literal. */
function markCitations(content: string, sourceCount: number): string {
  if (sourceCount === 0) return content;
  return content.replace(/\[(\d+)\]/g, (match, n) =>
    Number(n) >= 1 && Number(n) <= sourceCount ? `[${n}](#fuente-${n})` : match
  );
}

export function ChatMessage({ message, sources, streaming = false, onSourceClick }: ChatMessageProps) {
  const isUser = message.role === "user";

  if (isUser) {
    return (
      <div className="flex justify-end">
        <div
          className="max-w-[70%] rounded-lg rounded-br-sm px-3 py-2"
          style={{ backgroundColor: "#5b21b6", color: "white" }}
        >
          <p className="text-sm whitespace-pre-wrap">{message.content}</p>
          <p className="text-xs mt-1" style={{ color: "rgba(255,255,255,0.7)" }}>
            {ChatHelpers.formatTimestamp(message.createdAt)}
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="flex gap-2">
      <div
        className="h-7 w-7 rounded-full flex items-center justify-center flex-shrink-0"
        style={{ backgroundColor: "#ede9fe" }}
      >
        <Sparkles className="h-3.5 w-3.5" style={{ color: "#7c3aed" }} />
      </div>
      <div aria-live="polite" className="max-w-[70%]">
        <div
          className="rounded-lg rounded-tl-sm px-3 py-2"
          style={{ backgroundColor: "#f3f4f6" }}
        >
          <Markdown
            content={markCitations(message.content, sources?.length ?? 0)}
            citationCount={sources?.length ?? 0}
            showCopyButton
          />
          {streaming && <span className="ml-0.5 inline-block h-3.5 w-0.5 align-middle animate-pulse" style={{ backgroundColor: "#7c3aed" }} />}
          <p className="text-xs mt-1" style={{ color: "#6b7280" }}>
            {ChatHelpers.formatTimestamp(message.createdAt)}
            {message.tokensUsed > 0 && (
              <span
                className="ml-2 cursor-help underline decoration-dotted underline-offset-2"
                title={tpl(ui.knowledge.tokensUsed, { n: message.tokensUsed })}
              >
                {tpl(ui.knowledge.tokensUsed, { n: message.tokensUsed })}
              </span>
            )}
          </p>
        </div>
        {sources && sources.length > 0 && (
          <DocumentSources sources={sources} onSourceClick={onSourceClick} />
        )}
      </div>
    </div>
  );
}

export function TypingIndicator() {
  return (
    <div className="flex gap-2">
      <div
        className="h-7 w-7 rounded-full flex items-center justify-center flex-shrink-0"
        style={{ backgroundColor: "#ede9fe" }}
      >
        <Sparkles className="h-3.5 w-3.5" style={{ color: "#7c3aed" }} />
      </div>
      <div className="rounded-lg rounded-tl-sm px-3 py-2" style={{ backgroundColor: "#f3f4f6" }}>
        <div className="flex gap-1">
          <span className="h-2 w-2 rounded-full animate-bounce" style={{ backgroundColor: "#9ca3af", animationDelay: "0ms" }} />
          <span className="h-2 w-2 rounded-full animate-bounce" style={{ backgroundColor: "#9ca3af", animationDelay: "150ms" }} />
          <span className="h-2 w-2 rounded-full animate-bounce" style={{ backgroundColor: "#9ca3af", animationDelay: "300ms" }} />
        </div>
      </div>
    </div>
  );
}
