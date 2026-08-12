"use client";

import Link from "next/link";
import { useState } from "react";
import { Menu, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { copy } from "@/lib/copy/ui";

const NAV_LINKS = [
  { href: "/features", labelKey: "navFeatures" },
  { href: "/plataforma", labelKey: "navPlataforma" },
  { href: "/pricing", labelKey: "navPricing" },
  { href: "/blog", labelKey: "navBlog" },
  { href: "/academy", labelKey: "navAcademy" },
  { href: "/faq", labelKey: "navFaq" },
] as const;

export function LogoMark({ className = "w-8 h-8" }: { className?: string }) {
  // Brand anchor: emerald is the NexoChat/WhatsApp identity mark (single approved brand accent)
  return (
    <div className={`${className} bg-emerald-500 rounded-lg flex items-center justify-center shrink-0`}>
      <svg className="w-5 h-5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
        <path
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth="2"
          d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z"
        />
      </svg>
    </div>
  );
}

export function SiteHeader() {
  const [open, setOpen] = useState(false);

  return (
    <header className="sticky top-0 z-50 bg-slate-900/95 backdrop-blur supports-[backdrop-filter]:bg-slate-900/80 text-white border-b border-slate-800">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex items-center justify-between h-16">
          <Link href="/" className="flex items-center gap-3" aria-label="NexoChat — Inicio">
            <LogoMark />
            <span className="font-heading font-bold text-xl tracking-tight">NexoChat</span>
          </Link>
          <nav className="hidden md:flex items-center gap-8" aria-label="Principal">
            {NAV_LINKS.map((link) => (
              <Link
                key={link.href}
                href={link.href}
                className="text-sm text-slate-300 hover:text-white transition-colors"
              >
                {copy("marketing", link.labelKey)}
              </Link>
            ))}
          </nav>
          <div className="hidden md:flex items-center gap-4">
            <Link
              href="/auth"
              className="text-sm text-slate-300 hover:text-white transition-colors"
            >
              {copy("marketing", "signIn")}
            </Link>
            <Link href="/signup">
              <Button className="bg-emerald-500 hover:bg-emerald-600 text-white px-5 py-2.5 rounded-lg text-sm font-semibold">
                {copy("marketing", "tryFree")}
              </Button>
            </Link>
          </div>
          <button
            type="button"
            className="md:hidden p-2 text-slate-300 hover:text-white"
            aria-expanded={open}
            aria-label={open ? "Cerrar menú" : "Abrir menú"}
            onClick={() => setOpen((v) => !v)}
          >
            {open ? <X className="w-6 h-6" /> : <Menu className="w-6 h-6" />}
          </button>
        </div>
      </div>
      {open && (
        <div className="md:hidden bg-slate-800 border-t border-slate-700">
          <div className="px-4 py-4 space-y-3">
            {NAV_LINKS.map((link) => (
              <Link
                key={link.href}
                href={link.href}
                className="block text-slate-300 hover:text-white py-2"
                onClick={() => setOpen(false)}
              >
                {copy("marketing", link.labelKey)}
              </Link>
            ))}
            <Link
              href="/auth"
              className="block text-slate-300 hover:text-white py-2"
              onClick={() => setOpen(false)}
            >
              {copy("marketing", "signIn")}
            </Link>
            <Link href="/signup" onClick={() => setOpen(false)}>
              <Button className="w-full bg-emerald-500 hover:bg-emerald-600 text-white px-5 py-3 rounded-lg font-semibold mt-4">
                {copy("marketing", "tryFree")}
              </Button>
            </Link>
          </div>
        </div>
      )}
    </header>
  );
}
