"use client";

import { useEffect, useRef } from "react";

type Props = {
  user: string; // e.g. "info"
  domain: string; // e.g. "yourdomain.com"
  className?: string;
  children?: React.ReactNode; // link text, defaults to "Email us"
};

export function ObfuscatedEmail({ user, domain, className, children }: Props) {
  const ref = useRef<HTMLAnchorElement>(null);

  useEffect(() => {
    const a = ref.current;
    if (!a) return;
    const address = `${user}@${domain}`;
    a.href = `mailto:${address}`;
    if (!a.textContent || a.textContent.trim().length === 0) {
      a.textContent = address;
    }
  }, [user, domain]);

  return (
    <a ref={ref} className={className} rel="nofollow noopener noreferrer">
      {children ?? "Email us"}
    </a>
  );
}

