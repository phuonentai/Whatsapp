# Proposal: Enterprise marketing website (home, blog, academy) for NexoChat

## Why

The public landing page (`next_b2b_starter/app/page.tsx`) is the generic starter hero ("Welcome to NexoChat — a modern Next.js starter") with no product messaging, no pricing, no blog, and no learning content. NexoChat — a WhatsApp Business API CRM for the Colombian market (AI copilot, Siigo DIAN invoicing, PSE/Nequi payment links, multi-agent inbox, Ley 1581 compliance) — has no way to market, educate, or acquire signups through its own site. The `sitemap.ts` already advertises `/about`, `/faq`, `/security`, `/privacy`, `/terms` that do not exist.

This change delivers an enterprise-grade marketing website in the exact design language of the approved reference template (`next_b2b_starter/website/shuffle-20260811-1759-14517.zip`, a Shuffle "ChatFlow CRM" Tailwind template: dark `slate-900` sections + `emerald-500` accent, Inter body + Sora headings, WhatsApp-chat hero mockup, before/after comparison, 4-column feature grid, ROI slider calculator, 4-plan pricing with monthly/annual toggle, FAQ accordion, dark 4-column footer), plus a **blog** and an **academy** (structured course/lesson learning hub) — reusing the existing shadcn/ui kit, layout providers, copy layer, and Tailwind theme as much as possible.

## What Changes

- **Route group `app/(marketing)/`** hosting public pages (URLs unchanged by the group): `/` (rewritten homepage), `/features`, `/pricing`, `/blog`, `/blog/[slug]`, `/academy`, `/academy/[course]`, `/academy/[course]/[lesson]`, `/about`, `/faq`, `/security`, `/privacy`, `/terms`. Existing `app/page.tsx` is replaced by the group's homepage; `app/(marketing)/layout.tsx` renders the marketing header/footer around the root layout's provider stack (Stytch/Theme/Auth/Query — unchanged).
- **Design system, template-faithful**: extend `tailwind.config.ts` with `fontFamily.heading` (Sora via `next/font/google`), keep the existing shadcn tokens untouched; marketing components use explicit `slate-*`/`emerald-*` utilities exactly as the reference template does. New `components/marketing/` directory: `site-header`, `site-footer`, `hero`, `logo-strip`, `comparison`, `feature-grid`, `roi-calculator` (client, range sliders), `pricing` (client, interval toggle), `faq`, `cta-banner`, `blog-card`, `course-card`, `lesson-nav`, `section-heading`. Reuses shadcn `button`, `badge`, `accordion`, `card`, `separator`, `skeleton`, `input`; icons from `lucide-react`; motion via `framer-motion` (already a dependency, currently unused) for 2026-standard scroll reveals.
- **Blog**: markdown content in `content/blog/*.md` (frontmatter: title, description, date, author, category, tags, cover) rendered with the existing `react-markdown` + `remark-gfm` stack via a new fs-based `lib/content.ts` loader (no new dependencies, no CMS). `/blog` index with category filter + search-free cards; `/blog/[slug]` article page with reading time, author, JSON-LD `BlogPosting`; `/blog/feed.xml` RSS route; initial 4–6 SEO-grounded posts (WhatsApp API onboarding, RAG knowledge base, campaign consent Ley 1581, DIAN invoicing, payment links, team RBAC).
- **Academy**: markdown courses in `content/academy/<course>/` (course frontmatter: title, description, level, duration, track; lessons as ordered files). `/academy` hub grouped by track; `/academy/[course]` syllabus page with per-lesson metadata; `/academy/[course]/[lesson]` lesson page with markdown body, prev/next navigation, and `Course`/`LearningResource` JSON-LD. Initial 3 courses (WhatsApp API fundamentos, Bandeja + Copiloto IA, Facturación Siigo + pagos).
- **Copy layer**: extend `lib/copy/ui.ts` with a `marketing` namespace (Spanish-first, `en` mirror, `tpl()` for interpolations) following the existing typed pattern; long-form article/course bodies live in the markdown content tree, not the copy layer.
- **SEO**: per-route `generateMetadata` (title/description/OG), `JsonLd` additions (`Organization`, `WebSite` exist; add `Blog`, `Course`, `FAQPage` on relevant pages), `sitemap.ts` rewritten to include all marketing routes + blog/academy entries with per-route priorities, `robots.ts` kept (allow `/`, disallow `/api`, `/dashboard`). Root `metadata.metadataBase` stays a placeholder (deployment URL is out of scope).
- **Legal/trust pages**: `/privacy`, `/terms`, `/security`, `/about`, `/faq` — real pages (template-faithful layout) with grounded content (Ley 1581, DIAN, Stytch RBAC, hosting posture from the existing specs).

## Capabilities

### New Capabilities

- `marketing-website`: public enterprise marketing site — template-faithful design system, homepage sections, pricing, blog (index/post/RSS), academy (hub/course/lesson), legal pages, marketing copy namespace, SEO/structured data, content loader contract

### Modified Capabilities

- (none — no existing capability's behaviour changes; the landing route is replaced, which no spec currently governs)

## Impact

- **Frontend only**: `next_b2b_starter/` — new `app/(marketing)/` routes, `components/marketing/`, `content/`, `lib/content.ts`, `lib/copy/ui.ts` extension, `tailwind.config.ts` font addition, `app/sitemap.ts` rewrite, `app/page.tsx` removal. No backend, no DB migration, no SQLC, no auth/RBAC, no billing changes.
- **Dependencies**: none new. Sora font via `next/font/google` (self-hosted at build); markdown via existing `react-markdown`/`remark-gfm`; motion via existing `framer-motion`.
- **Auth**: no Stytch flow changes. Public routes remain unauthenticated; `/signup` CTA targets are the existing signup flow.
- **Ops**: `pnpm dev`/`pnpm build`/`pnpm lint` in `next_b2b_starter/`; content is build-time static (no runtime DB).
- **Rollback**: Git — revert the change (route group, components, content, config). No DB or Stytch tenant state involved; nothing to roll back outside the repo. The previous `app/page.tsx` is recoverable from Git history.
- **Non-Goals**: no CMS/headless content management (content is build-time markdown); no newsletter/lead capture persistence (CTAs link to `/signup` only); no auth-gated academy (content is public); no i18n routing (Spanish-first site, matching the product's copy convention; `en` mirror exists in the copy layer for future use); no dark/light marketing theme switching (the reference design is fixed dark-hero/light-body; app theme system untouched); no blog search, pagination is capped at 12 posts per page (static generation).

## Assumptions

- The Shuffle template (`shuffle-20260811-1759-14517.zip`) is the approved design/color reference: slate-900 + emerald-500, Inter + Sora, section composition as extracted into the design doc. Where template copy is Spanish, we reuse its structure but rewrite copy for NexoChat's actual feature set (grounded in `openspec/specs/`).
- Exact plan names/prices on `/pricing` are marketing display values (Free/Starter/Pro/Enterprise per the template and the product's tier vocabulary); live Polar catalog pricing is out of scope for the marketing site.
- Blog/academy author identities use the product team (e.g., "Equipo NexoChat") — no real personal names are fabricated.
- `content/` markdown is authored as part of this change (not user-generated).
