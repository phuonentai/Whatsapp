import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { cn } from "@/lib/utils";

/**
 * Server-rendered markdown for long-form content (blog posts, academy lessons).
 * GFM enabled (tables, task lists, strikethrough, autolinks). Hand-rolled prose
 * styling matching the marketing design language (no typography plugin dep).
 */
export function Prose({ content, className }: { content: string; className?: string }) {
  return (
    <div
      className={cn(
        "prose-marketing text-muted-foreground leading-relaxed",
        className
      )}
    >
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        components={{
          h2: (props) => (
            <h2
              className="font-heading text-2xl font-bold text-foreground mt-12 mb-4 tracking-tight"
              {...props}
            />
          ),
          h3: (props) => (
            <h3
              className="font-heading text-xl font-bold text-foreground mt-8 mb-3 tracking-tight"
              {...props}
            />
          ),
          h4: (props) => (
            <h4 className="text-lg font-bold text-foreground mt-6 mb-2" {...props} />
          ),
          p: (props) => <p className="my-4 text-[1.05rem]" {...props} />,
          a: (props) => (
            <a
              className="text-primary font-medium underline underline-offset-2 hover:text-primary/80"
              target={props.href?.startsWith("http") ? "_blank" : undefined}
              rel={props.href?.startsWith("http") ? "noreferrer" : undefined}
              {...props}
            />
          ),
          ul: (props) => <ul className="list-disc pl-6 my-4 space-y-2" {...props} />,
          ol: (props) => <ol className="list-decimal pl-6 my-4 space-y-2" {...props} />,
          li: (props) => <li className="leading-relaxed" {...props} />,
          blockquote: (props) => (
            <blockquote
              className="border-l-4 border-primary pl-4 my-6 text-muted-foreground italic"
              {...props}
            />
          ),
          code: (props) => (
            <code
              className="rounded bg-muted px-1.5 py-0.5 text-sm text-foreground font-mono"
              {...props}
            />
          ),
          pre: (props) => (
            <pre
              className="overflow-x-auto rounded-xl bg-foreground p-4 my-6 text-sm text-background"
              {...props}
            />
          ),
          table: (props) => (
            <div className="overflow-x-auto my-6">
              <table className="w-full text-sm border-collapse" {...props} />
            </div>
          ),
          th: (props) => (
            <th
              className="border border-border bg-muted px-3 py-2 text-left font-semibold text-foreground"
              {...props}
            />
          ),
          td: (props) => (
            <td className="border border-border px-3 py-2" {...props} />
          ),
          hr: (props) => <hr className="my-8 border-border" {...props} />,
          strong: (props) => <strong className="font-semibold text-foreground" {...props} />,
          img: (props) => <img className="rounded-xl my-6 max-w-full h-auto" alt={props.alt ?? ""} {...props} />,
        }}
      >
        {content}
      </ReactMarkdown>
    </div>
  );
}
