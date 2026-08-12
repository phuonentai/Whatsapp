import Link from "next/link";
import { copy } from "@/lib/copy/ui";
import { LogoMark } from "./site-header";

const PRODUCT_LINKS: { href: string }[] = [
  { href: "/features" },
  { href: "/pricing" },
  { href: "/blog" },
  { href: "/academy" },
];

const RESOURCE_LINKS: { href: string }[] = [
  { href: "/blog" },
  { href: "/academy" },
  { href: "/faq" },
  { href: "/security" },
];

const LEGAL_LINKS: { href: string }[] = [
  { href: "/privacy#habeas-data" },
  { href: "/terms" },
  { href: "/privacy" },
  { href: "/about" },
];

function SocialLinks() {
  return (
    <div className="flex gap-4">
      <a href="https://twitter.com" target="_blank" rel="noreferrer" className="text-slate-400 hover:text-white transition-colors" aria-label="X (Twitter)">
        <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 24 24" aria-hidden="true">
          <path d="M24 12.073c0-6.627-5.373-12-12-12s-12 5.373-12 12c0 5.99 4.388 10.954 10.125 11.854v-8.385H7.078v-3.47h3.047V9.43c0-3.007 1.792-4.669 4.533-4.669 1.312 0 2.686.235 2.686.235v2.953H15.83c-1.491 0-1.956.925-1.956 1.874v2.25h3.328l-.532 3.47h-2.796v8.385C19.612 23.027 24 18.062 24 12.073z" />
        </svg>
      </a>
      <a href="https://www.linkedin.com" target="_blank" rel="noreferrer" className="text-slate-400 hover:text-white transition-colors" aria-label="LinkedIn">
        <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 24 24" aria-hidden="true">
          <path d="M20.447 20.452h-3.554v-5.569c0-1.328-.027-3.037-1.852-3.037-1.853 0-2.136 1.445-2.136 2.939v5.667H9.351V9h3.414v1.561h.046c.477-.9 1.637-1.85 3.37-1.85 3.601 0 4.267 2.37 4.267 5.455v6.286zM5.337 7.433c-1.144 0-2.063-.926-2.063-2.065 0-1.138.92-2.063 2.063-2.063 1.14 0 2.064.925 2.064 2.063 0 1.139-.925 2.065-2.064 2.065zm1.782 13.019H3.555V9h3.564v11.452zM22.225 0H1.771C.792 0 0 .774 0 1.729v20.542C0 23.227.792 24 1.771 24h20.451C23.2 24 24 23.227 24 22.271V1.729C24 .774 23.2 0 22.222 0h.003z" />
        </svg>
      </a>
      <a href="https://www.instagram.com" target="_blank" rel="noreferrer" className="text-slate-400 hover:text-white transition-colors" aria-label="Instagram">
        <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 24 24" aria-hidden="true">
          <path d="M12 2.163c3.204 0 3.584.012 4.85.07 3.252.148 4.771 1.691 4.919 4.919.058 1.265.069 1.645.069 4.849 0 3.205-.012 3.584-.069 4.849-.149 3.225-1.664 4.771-4.919 4.919-1.266.058-1.644.07-4.85.07-3.204 0-3.584-.012-4.849-.07-3.26-.149-4.771-1.699-4.919-4.92-.058-1.265-.07-1.644-.07-4.849 0-3.204.013-3.583.07-4.849.149-3.227 1.664-4.771 4.919-4.919 1.266-.057 1.645-.069 4.849-.069zm0-2.163c-3.259 0-3.667.014-4.947.072-4.358.2-6.78 2.618-6.98 6.98-.059 1.281-.073 1.689-.073 4.948 0 3.259.014 3.668.072 4.948.2 4.358 2.618 6.78 6.98 6.98 1.281.058 1.689.072 4.948.072 3.259 0 3.668-.014 4.948-.072 4.354-.2 6.782-2.618 6.979-6.98.059-1.28.073-1.689.073-4.948 0-3.259-.014-3.667-.072-4.947-.196-4.354-2.617-6.78-6.979-6.98-1.281-.059-1.69-.073-4.949-.073zm0 5.838a6.162 6.162 0 100 12.324 6.162 6.162 0 000-12.324zm0 10.162a4 4 0 110-8 4 4 0 010 8zm6.406-11.845a1.44 1.44 0 100 2.881 1.44 1.44 0 000-2.881z" />
        </svg>
      </a>
    </div>
  );
}

export function SiteFooter() {
  return (
    <footer className="bg-slate-900 text-white py-12 border-t border-slate-800">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="grid md:grid-cols-4 gap-8">
          <div className="col-span-2">
            <div className="flex items-center gap-3 mb-4">
              <LogoMark />
              <span className="font-heading font-bold text-xl">NexoChat</span>
            </div>
            <p className="text-slate-400 text-sm max-w-sm">
              {copy("marketing", "footerTagline")}
            </p>
            <div className="mt-6">
              <SocialLinks />
            </div>
          </div>
          <div>
            <h4 className="font-semibold mb-4 text-white">{copy("marketing", "footerProduct")}</h4>
            <ul className="space-y-2 text-sm text-slate-400">
              {copy("marketing", "footerProductLinks").map((label, i) => (
                <li key={label}>
                  <Link href={PRODUCT_LINKS[i]?.href ?? "/"} className="hover:text-white transition-colors">
                    {label}
                  </Link>
                </li>
              ))}
            </ul>
          </div>
          <div>
            <h4 className="font-semibold mb-4 text-white">{copy("marketing", "footerResources")}</h4>
            <ul className="space-y-2 text-sm text-slate-400">
              {copy("marketing", "footerResourceLinks").map((label, i) => (
                <li key={label}>
                  <Link href={RESOURCE_LINKS[i]?.href ?? "/"} className="hover:text-white transition-colors">
                    {label}
                  </Link>
                </li>
              ))}
            </ul>
          </div>
          <div>
            <h4 className="font-semibold mb-4 text-white">{copy("marketing", "footerLegal")}</h4>
            <ul className="space-y-2 text-sm text-slate-400">
              {copy("marketing", "footerLegalLinks").map((label, i) => (
                <li key={label}>
                  <Link href={LEGAL_LINKS[i]?.href ?? "/"} className="hover:text-white transition-colors">
                    {label}
                  </Link>
                </li>
              ))}
            </ul>
          </div>
        </div>
        <div className="border-t border-slate-800 mt-12 pt-8 flex flex-col md:flex-row justify-between items-center gap-4">
          <p className="text-slate-400 text-sm">© 2026 NexoChat. {copy("marketing", "footerRights")}</p>
          <Link href="/security" className="text-slate-400 text-sm hover:text-white transition-colors">
            Estado del servicio
          </Link>
        </div>
      </div>
    </footer>
  );
}
