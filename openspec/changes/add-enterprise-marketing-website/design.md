# Design: Enterprise marketing website (home, blog, academy)

## D0. Design system (from the approved Shuffle template)

Extracted from `next_b2b_starter/website/shuffle-ref/shuffle/src/html/index.html` (reference template):

- **Palette**: dark sections `slate-900` (nav, hero, features, footer) with `slate-800`/`slate-700` cards and `slate-300`/`slate-400` secondary text; light sections `white` / `slate-50`; single accent **`emerald-500`** (CTAs, highlights, icons) with tints `emerald-50`, `emerald-500/10`, `emerald-500/20`, `emerald-600` hover; destructive hints `red-500`.
- **Typography**: Inter (body, already loaded via `next/font/google` as `--font-sans`), **Sora** (headings — ADD via `next/font/google`, `variable: --font-heading`; wire `fontFamily.heading` in `tailwind.config.ts`). Headings `font-heading font-bold tracking-tight`; hero H1 `text-4xl sm:text-5xl lg:text-6xl`.
- **Layout rhythm**: `max-w-7xl mx-auto px-4 sm:px-6 lg:px-8`; section padding `py-20 lg:py-28`; nav `h-16`; footer `py-12`; radius `rounded-lg/xl/2xl`; shadows `shadow-lg shadow-emerald-500/25`.
- **Motion**: buttons `hover:scale-105 active:scale-95 transition-all`; badge pulse `animate-pulse`; card hover `hover:border-emerald-500/50 group-hover:bg-emerald-500/30`; scroll-reveal via `framer-motion` (`whileInView`, viewport once, stagger) — motion is progressive enhancement, content must be readable without JS.
- **Dark hero composition**: gradient `bg-gradient-to-br from-slate-900 via-slate-900 to-emerald-900/20` + right-side `bg-gradient-to-l from-emerald-500/10`; two-column: copy (badge, H1 with emerald span, lead, 2 CTAs, "integraciones" strip with `border-t border-slate-800`) + browser-frame chat mockup (`bg-slate-800 rounded-2xl border-slate-700`, macOS dots, split WhatsApp chat / Siigo invoice panels) + floating stat card (`-bottom-6 -left-6 bg-white rounded-xl shadow-xl`).

## D1. Routing map

```
app/
  (marketing)/
    layout.tsx            # marketing shell: <SiteHeader/> + children + <SiteFooter/>
    page.tsx              # / homepage (replaces app/page.tsx; old file removed)
    features/page.tsx     # /features
    pricing/page.tsx      # /pricing
    blog/
      page.tsx            # /blog index (category filter, cards)
      [slug]/page.tsx     # /blog/[slug] article
      feed.xml/route.ts   # /blog/feed.xml RSS (Route Handler)
    academy/
      page.tsx            # /academy hub (tracks + course cards)
      [course]/page.tsx   # /academy/[course] syllabus
      [course]/[lesson]/page.tsx   # /academy/[course]/[lesson]
    about/page.tsx  faq/page.tsx  security/page.tsx  privacy/page.tsx  terms/page.tsx
```
Root `app/layout.tsx` (providers, Inter, metadata) is unchanged except `metadataBase` placeholder retained. `app/page.tsx` is deleted; `app/robots.ts` unchanged; `app/sitemap.ts` rewritten.

## D2. Content model (build-time markdown, no new deps)

- `content/blog/<slug>.md` frontmatter: `title, description, date (ISO), author, category, tags[], cover (optional path in public/), draft?`. Body: CommonMark + GFM tables/lists (remark-gfm).
- `content/academy/<course-slug>/course.md` frontmatter: `title, description, level (Principiante|Intermedio|Avanzado), durationMinutes, track, order, cover?`. Lessons: `content/academy/<course-slug>/<NN>-<lesson-slug>.md` frontmatter: `title, description, order, durationMinutes`; body markdown.
- `lib/content.ts` contract (fs-based, server-only):
  - `getBlogPosts(): BlogPostMeta[]` (sorted desc, drafts excluded)
  - `getBlogPost(slug): { meta, body } | null`
  - `getBlogCategories(): string[]`
  - `getCourses(): CourseMeta[]`, `getCourse(slug): { meta, lessons: LessonMeta[] } | null`
  - `getLesson(courseSlug, lessonSlug): { meta, body, prev, next } | null`
  - `parseFrontmatter<T>(raw): { data: T; body: string }` — tiny regex splitter on `---` fences (no gray-matter dep)
  - types: `BlogPostMeta`, `CourseMeta`, `LessonMeta` exported from `lib/content/types.ts` (or same file)
- `app/blog/feed.xml/route.ts`: RSS 2.0 with `title/description/link/pubDate` per post; `Content-Type: application/rss+xml`.
- Initial content: 5 blog posts, 3 academy courses × 4–6 lessons each (topics grounded in `openspec/specs/`, listed in tasks).

## D3. Component plan (`components/marketing/`)

Reuse: shadcn `button`, `badge`, `accordion` (FAQ), `card`, `separator`, `skeleton` (loading), `input` (search-free; newsletter is a CTA link). Icons: `lucide-react`. Motion: `framer-motion` (`motion.div`, `whileInView`, `viewport={{ once: true }}`).

- `site-header.tsx` ("use client" for mobile menu + scroll state): dark `bg-slate-900` sticky, logo mark (emerald rounded square + chat glyph), links (Producto→/features, Precios→/pricing, Blog→/blog, Academia→/academy, FAQ→/faq), CTA `Iniciar sesión` (→/auth) + `Probar gratis` (→/signup, emerald). Mobile: hamburger → slide-down panel.
- `site-footer.tsx`: 4-column dark footer (brand blurb + Producto / Recursos / Legal columns + socials), bottom bar with copyright + status link.
- `hero.tsx`: template composition D0 (server component; stat card static).
- `logo-strip.tsx`: "Integrado con" strip (text logos: Meta WhatsApp, Siigo, MercadoPago, Polar, Stytch — text-only, no trademark assets).
- `comparison.tsx`: before/after two-card section (red "Proceso tradicional" vs emerald "Con NexoChat IA").
- `feature-grid.tsx`: dark 4-col grid (Entrenamiento IA 60s, Facturación Siigo 1-clic, Pagos automatizados, Bandeja multi-agente, Campañas con consentimiento, Playbooks verticales, Analítica COP, RBAC Stytch) + props for eyebrow/title.
- `roi-calculator.tsx` ("use client"): range sliders (conversaciones diarias 50–1000, vendedores 1–20) → hours/week + COP lost; reuses shadcn `Slider` or native range styled as template.
- `pricing.tsx` ("use client"): 4 plans (Gratis $0 COP / Starter $39 USD / Pro $119 USD destacado / Enterprise Custom), monthly/annual toggle with `-20%`, template card styles.
- `faq.tsx`: shadcn `Accordion` + `FAQPage` JSON-LD prop.
- `cta-banner.tsx`: emerald gradient banner ("Recupera este tiempo ahora" → /signup).
- `section-heading.tsx`: eyebrow/title/lead block used across sections.
- `blog-card.tsx`, `course-card.tsx`, `lesson-nav.tsx` (prev/next + progress), `prose.tsx`: markdown renderer wrapper (react-markdown + remark-gfm + tailwind typography-like classes; no @tailwindcss/typography dep — hand-rolled prose classes).
- `page-hero.tsx`: compact dark hero for subpages (features/pricing/blog/academy/about…).

## D4. Copy

- Extend `lib/copy/ui.ts`: add `marketing` namespace to `ui` (Spanish) + `en` mirror: nav labels, hero, section headings, feature titles/descriptions, pricing, FAQ, footer, CTA, blog/academy chrome. Follow `tpl()` for interpolations (e.g. `roiHours({hours})`).
- Long-form markdown lives in `content/` (not ui.ts).

## D5. SEO & metadata

- `generateMetadata` on every marketing route: title template `%s | NexoChat`, description, OG/Twitter (site defaults from root layout), canonical `/…`.
- JSON-LD: `Organization` + `WebSite` already global; add `Blog` (index), `BlogPosting` (post, with author/datePublished/dateModified), `Course` + `LearningResource` (academy), `FAQPage` (/faq + homepage FAQ), `Product`/`Offer` (pricing, bestOffer).
- `sitemap.ts`: static marketing routes + generated blog/academy entries; priorities: home 1.0, pricing/features 0.8, blog/academy 0.7, legal 0.4.
- `app/head.tsx` untouched.

## D6. Open questions / decisions

- Landing replaces `app/page.tsx` (URL `/` unchanged). The not-found redirect behavior (dashboard-aware) is left as-is; marketing 404 is out of scope unless trivially improvable.
- Dark/light: marketing pages fixed to template palette (no next-themes toggle on marketing pages).
