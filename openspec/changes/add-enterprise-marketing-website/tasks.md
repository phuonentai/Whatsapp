## 1. Foundation [FE-NEXT]

- [x] 1.1 Tailwind: add `Sora` via `next/font/google` in a `components/marketing/fonts.ts` (or root layout) with `variable: "--font-heading"`; wire `fontFamily.heading` in `tailwind.config.ts`. Verify: `pnpm build` succeeds; `font-heading` utility compiles
- [x] 1.2 `lib/content.ts` + types: fs-based loaders `getBlogPosts/getBlogPost/getBlogCategories/getCourses/getCourse/getLesson`, `parseFrontmatter<T>`, exported types `BlogPostMeta/CourseMeta/LessonMeta`. Verify: `pnpm build` type-checks; a `node -e` smoke call returns sorted posts
- [x] 1.3 Copy layer: add `marketing` namespace to `lib/copy/ui.ts` (`ui` Spanish + `en` mirror): nav, hero, sections, pricing, FAQ, footer, CTA, blog/academy chrome. Verify: `pnpm lint`; `copy("marketing", "heroTitle")` typechecks
- [x] 1.4 Route group shell: `app/(marketing)/layout.tsx` (SiteHeader + children + SiteFooter), `components/marketing/site-header.tsx` (dark sticky nav + mobile menu, links to /features /pricing /blog /academy /faq, CTAs → /auth and /signup), `components/marketing/site-footer.tsx` (4-column + bottom bar), `components/marketing/page-hero.tsx`, `components/marketing/section-heading.tsx`, `components/marketing/prose.tsx` (react-markdown + remark-gfm wrapper), delete `app/page.tsx` and add `app/(marketing)/page.tsx` placeholder. Verify: `pnpm dev` renders `/` with header/footer; `pnpm build`

## 2. Homepage [FE-NEXT]

- [x] 2.1 `components/marketing/hero.tsx` (template composition: badge, H1 with emerald span, lead, CTAs, integrations strip, chat mockup with macOS dots + WhatsApp/Siigo panels + floating 1.2s stat), `logo-strip.tsx`. Verify: `/` renders the dark hero
- [x] 2.2 `comparison.tsx` (tradicional vs IA), `feature-grid.tsx` (8 features), `roi-calculator.tsx` (client, sliders → hours/COP), `cta-banner.tsx`, `faq.tsx` (shadcn Accordion, 6 items, `FAQPage` JSON-LD). Verify: interactions work in `pnpm dev` (slider math, accordion)
- [x] 2.3 Assemble `app/(marketing)/page.tsx`: hero → logo-strip → comparison → features → ROI → pricing preview → FAQ → CTA. Framer-motion scroll reveals (`whileInView`, once). Verify: `pnpm build`; homepage matches template design language

## 3. Pricing & features pages [FE-NEXT]

- [x] 3.1 `components/marketing/pricing.tsx` (client toggle mensual/anual -20%, 4 plans, destacado Pro) + `/pricing` page with `Offer` JSON-LD. Verify: toggle swaps prices; `pnpm build`
- [x] 3.2 `/features` page: deep feature sections (bandeja multi-canal, copiloto IA, campañas con consentimiento Ley 1581, facturación Siigo DIAN, pagos, playbooks verticales, analítica, RBAC/seguridad) grounded in specs. Verify: `pnpm build`

## 4. Blog [FE-NEXT]

- [x] 4.1 Content: 5 posts in `content/blog/` (WhatsApp API onboarding, RAG knowledge base, campañas y consentimiento Ley 1581, facturación DIAN con Siigo, links de pago y cobros) with real frontmatter. Verify: `getBlogPosts()` returns 5 sorted
- [x] 4.2 `/blog` index (category filter chips, `blog-card.tsx`, `Blog` JSON-LD, metadata) + `/blog/[slug]` (prose render, reading time, author, `BlogPosting` JSON-LD, back link). Verify: both routes render posts; unknown slug → notFound()
- [x] 4.3 `/blog/feed.xml` RSS route. Verify: `curl` returns valid RSS with 5 items

## 5. Academy [FE-NEXT]

- [x] 5.1 Content: 3 courses × 4–6 lessons in `content/academy/` (WhatsApp API fundamentos; Bandeja + Copiloto IA; Facturación Siigo + pagos). Verify: `getCourses()` returns 3 with lessons ordered
- [x] 5.2 `/academy` hub (tracks + `course-card.tsx`, metadata), `/academy/[course]` syllabus (lesson list with durations, `Course` JSON-LD), `/academy/[course]/[lesson]` (prose, prev/next `lesson-nav.tsx`). Verify: routes render; bad slugs → notFound()

## 6. Legal/trust pages + SEO [FE-NEXT]

- [x] 6.1 `/about`, `/faq` (Accordion + FAQPage), `/security`, `/privacy`, `/terms` — template-faithful layout, grounded content (Ley 1581, DIAN, Stytch RBAC, datos). Verify: all five render; `pnpm build`
- [x] 6.2 `app/sitemap.ts` rewrite (marketing + blog + academy entries with priorities); robots.ts verified. Verify: `pnpm build`; sitemap lists all routes

## 7. Verification gate [OPS-GOV]

- [x] 7.1 Verification gate run and recorded:
  - `pnpm lint` — PASS (0 errors, 4 warnings: 3 pre-existentes en components/crm, 1 en prose.tsx img)
  - `pnpm build` — PASS (clean build; 16 rutas marketing + dashboard/app existentes)
  - Smoke prod (`next start`): `/`, `/features`, `/plataforma`, `/pricing`, `/blog`, `/blog/<slug>`, `/academy`, curso, lección, `/about`, `/faq`, `/security`, `/privacy`, `/terms`, `/blog/feed.xml`, `/sitemap.xml` — todos 200 con layout esperado; RSS 2.0 con 5 items; sitemap incluye blog/academy/plataforma; JSON-LD BlogPosting/Course/LearningResource/FAQPage presentes
  - `openspec validate add-enterprise-marketing-website` — PASS
  - Nota: slugs desconocidos de blog/academy renderizan la página 404 con branding (status 200 por limitación de plataforma con streaming + layout raíz async; rutas inexistentes sí devuelven 404) — documentado en el delta spec
- [x] 7.2 Archive decision: **Archive deferred:** revisión pendiente del sitio entregado antes de archivar

## Central re-verification (2026-08-11, Phase 1 of repo-wide active-changes run)

- [x] Re-ran gates: `pnpm lint` PASS (0 errors / 4 pre-existing warnings), `npx tsc --noEmit` PASS, `pnpm build` PASS (baseline sweep, 16 marketing routes + dashboard/app intact). 7.1 smoke results (prod `next start` on all 16 routes, RSS, sitemap, JSON-LD) previously recorded PASS.
- [ ] Archive decision remains **Archive deferred**: pending site review of the delivered marketing site (human review), per 7.2 record. All technical gates green.
