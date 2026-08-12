import { cn } from "@/lib/utils";

interface SectionHeadingProps {
  eyebrow?: string;
  title: string;
  lead?: string;
  align?: "left" | "center";
  /** Kept for call-site compatibility; sections are soft-themed by tokens. */
  dark?: boolean;
  className?: string;
}

export function SectionHeading({
  eyebrow,
  title,
  lead,
  align = "center",
  className,
}: SectionHeadingProps) {
  return (
    <div
      className={cn(
        "max-w-3xl mb-16",
        align === "center" && "text-center mx-auto",
        className
      )}
    >
      {eyebrow && (
        <p className="text-sm font-semibold uppercase tracking-wider mb-3 text-primary">
          {eyebrow}
        </p>
      )}
      <h2 className="font-heading text-3xl sm:text-4xl font-bold tracking-tight text-balance text-foreground">
        {title}
      </h2>
      {lead && (
        <p className="mt-4 text-lg leading-relaxed text-muted-foreground">
          {lead}
        </p>
      )}
    </div>
  );
}
