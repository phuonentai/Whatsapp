"use client";

import { useState } from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { Check, Copy } from "lucide-react";
import { cn } from "@/lib/utils";
import { ui } from "@/lib/copy/ui";

interface MarkdownProps {
  content: string;
  className?: string;
  /** Show a copy-to-clipboard button (assistant messages). */
  showCopyButton?: boolean;
  /**
   * When set, `[n](#fuente-n)` links are rendered as inline citation chips
   * (styled violet superscripts) instead of external links.
   */
  citationCount?: number;
}

/**
 * Shared markdown renderer for AI-generated content.
 *
 * - GFM (tables, task lists, strikethrough, autolinks) via `remark-gfm`.
 * - Deliberately NO `rehype-raw`: raw HTML in model output stays escaped text,
 *   never rendered as markup.
 * - Styled to the chat-bubble theme (gray bubble, violet accents).
 */
export function Markdown({ content, className, showCopyButton = false, citationCount = 0 }: MarkdownProps) {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    try {
      if (navigator.clipboard?.writeText) {
        await navigator.clipboard.writeText(content);
      } else {
        // jsdom / non-secure contexts: select-and-copy fallback.
        const textarea = document.createElement("textarea");
        textarea.value = content;
        textarea.style.position = "fixed";
        textarea.style.opacity = "0";
        document.body.appendChild(textarea);
        textarea.select();
        document.execCommand("copy");
        document.body.removeChild(textarea);
      }
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1500);
    } catch {
      // Clipboard unavailable — ignore, content is still visible.
    }
  };

  return (
    <div className={cn("group relative", className)}>
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        components={{
          p: ({ children }) => (
            <p className="my-1.5 text-sm leading-relaxed first:mt-0 last:mb-0" style={{ color: "#111827" }}>
              {children}
            </p>
          ),
          a: ({ children, href }) => {
            const citation = /^#fuente-(\d+)$/.exec(href ?? "");
            if (citation && Number(citation[1]) <= citationCount) {
              return (
                <a
                  href={href}
                  className="inline-flex items-center justify-center rounded-full px-1.5 py-0.5 text-[10px] font-semibold align-super transition-colors hover:opacity-80"
                  style={{ backgroundColor: "#ede9fe", color: "#6d28d9" }}
                >
                  {children}
                </a>
              );
            }
            return (
              <a
                href={href}
                target="_blank"
                rel="noreferrer noopener"
                className="underline underline-offset-2 hover:opacity-80"
                style={{ color: "#7c3aed" }}
              >
                {children}
              </a>
            );
          },
          ul: ({ children }) => (
            <ul className="my-1.5 list-disc pl-5 text-sm" style={{ color: "#111827" }}>
              {children}
            </ul>
          ),
          ol: ({ children }) => (
            <ol className="my-1.5 list-decimal pl-5 text-sm" style={{ color: "#111827" }}>
              {children}
            </ol>
          ),
          li: ({ children }) => <li className="my-0.5 leading-relaxed">{children}</li>,
          h1: ({ children }) => (
            <h1 className="mb-1 mt-2 text-base font-semibold" style={{ color: "#111827" }}>
              {children}
            </h1>
          ),
          h2: ({ children }) => (
            <h2 className="mb-1 mt-2 text-sm font-semibold" style={{ color: "#111827" }}>
              {children}
            </h2>
          ),
          h3: ({ children }) => (
            <h3 className="mb-1 mt-2 text-sm font-semibold" style={{ color: "#111827" }}>
              {children}
            </h3>
          ),
          h4: ({ children }) => (
            <h4 className="mb-1 mt-2 text-sm font-semibold" style={{ color: "#111827" }}>
              {children}
            </h4>
          ),
          strong: ({ children }) => <strong className="font-semibold">{children}</strong>,
          em: ({ children }) => <em>{children}</em>,
          del: ({ children }) => <del className="text-gray-400">{children}</del>,
          hr: () => <hr className="my-3 border-t" style={{ borderColor: "#e5e7eb" }} />,
          blockquote: ({ children }) => (
            <blockquote
              className="my-1.5 border-l-2 pl-3 text-sm italic"
              style={{ borderColor: "#c4b5fd", color: "#4b5563" }}
            >
              {children}
            </blockquote>
          ),
          pre: ({ children }) => (
            <pre className="my-2 overflow-x-auto rounded-lg bg-gray-900 p-3">{children}</pre>
          ),
          code: ({ className: codeClassName, children, node: _node, ...rest }) => {
            const isBlock = /language-/.test(codeClassName ?? "");
            if (isBlock) {
              return (
                <code
                  className={cn("block text-xs font-mono leading-relaxed text-gray-100", codeClassName)}
                  {...rest}
                >
                  {children}
                </code>
              );
            }
            return (
              <code
                className="rounded bg-gray-100 px-1.5 py-0.5 text-xs font-mono text-gray-800"
                {...rest}
              >
                {children}
              </code>
            );
          },
          table: ({ children }) => (
            <div className="my-2 overflow-x-auto">
              <table className="w-full border-collapse text-xs" style={{ color: "#111827" }}>
                {children}
              </table>
            </div>
          ),
          thead: ({ children }) => (
            <thead className="border-b" style={{ borderColor: "#d1d5db" }}>
              {children}
            </thead>
          ),
          th: ({ children }) => (
            <th className="border px-2 py-1 text-left font-semibold" style={{ borderColor: "#d1d5db" }}>
              {children}
            </th>
          ),
          td: ({ children }) => (
            <td className="border px-2 py-1" style={{ borderColor: "#d1d5db" }}>
              {children}
            </td>
          ),
        }}
      >
        {content}
      </ReactMarkdown>
      {showCopyButton && (
        <button
          type="button"
          onClick={handleCopy}
          aria-label={copied ? ui.knowledge.copied : ui.knowledge.copy}
          title={ui.knowledge.copy}
          className="absolute -right-1 -top-2 flex h-6 w-6 items-center justify-center rounded-md border bg-white opacity-0 shadow-sm transition-opacity hover:bg-gray-50 focus:opacity-100 group-hover:opacity-100"
          style={{ borderColor: "#e5e7eb", color: "#6b7280" }}
        >
          {copied ? <Check className="h-3 w-3" style={{ color: "#16a34a" }} /> : <Copy className="h-3 w-3" />}
        </button>
      )}
    </div>
  );
}
