# marketing-website Specification

## Purpose
TBD - created by archiving change add-enterprise-marketing-website. Update Purpose after archive.
## Requirements
### Requirement: Marketing website route group

The system SHALL serve public marketing pages under an `app/(marketing)` route group with URLs: `/`, `/features`, `/pricing`, `/blog`, `/blog/[slug]`, `/academy`, `/academy/[course]`, `/academy/[course]/[lesson]`, `/about`, `/faq`, `/security`, `/privacy`, `/terms`. The group layout SHALL render the marketing site header and footer around the existing root layout provider stack (Stytch, theme, auth, query). The previous `app/page.tsx` landing SHALL be replaced by the group homepage.

#### Scenario: Public access to all marketing routes

- **WHEN** an unauthenticated visitor requests any marketing route listed above
- **THEN** the route SHALL render HTTP 200 with the marketing shell (header, footer) and no authentication redirect

#### Scenario: Landing replaced

- **WHEN** a visitor requests `/`
- **THEN** the system SHALL render the new marketing homepage, not the starter landing

### Requirement: Template-faithful design system

Marketing pages SHALL use the approved reference design: dark `slate-900` sections with `emerald-500` accent, Inter body (existing) plus Sora headings, `max-w-7xl` containers, and the template's section compositions (dark hero with chat mockup, before/after comparison, feature grid, ROI calculator, pricing cards with interval toggle, FAQ accordion, 4-column footer). The existing shadcn theme tokens (CSS variables) SHALL NOT be altered for the app shell; marketing components SHALL use explicit Tailwind utilities.

#### Scenario: Hero renders template composition

- **WHEN** the homepage renders
- **THEN** it SHALL include the dark gradient hero with a WhatsApp-chat/Siigo mockup and a floating stat card, matching the reference template structure

#### Scenario: App shell tokens untouched

- **WHEN** the marketing site is built
- **THEN** `app/globals.css` CSS variables and `tailwind.config.ts` color tokens SHALL remain unchanged except for the added heading font family

### Requirement: Blog with static content

The system SHALL provide a blog: markdown posts in `content/blog/*.md` with frontmatter (`title`, `description`, `date`, `author`, `category`, `tags`, optional `cover`), a `/blog` index with category filtering, `/blog/[slug]` article pages rendering the markdown body (react-markdown + remark-gfm), reading time, and an RSS feed at `/blog/feed.xml`. Unknown slugs SHALL return 404.

#### Scenario: Post list and detail

- **WHEN** a visitor requests `/blog`
- **THEN** the index SHALL list published posts sorted by date descending with category filter chips
- **AND** requesting `/blog/<existing-slug>` SHALL render the article with metadata

#### Scenario: Unknown blog slug

- **WHEN** a visitor requests `/blog/<unknown-slug>`
- **THEN** the system SHALL render the branded 404 page instead of an article
- **AND** the response SHALL NOT index the URL (no article content is served); the HTTP status code follows the platform's not-found handling (known limitation: with streaming enabled and an async root layout, the boundary renders with 200; route-level misses return 404)

#### Scenario: RSS feed

- **WHEN** a visitor requests `/blog/feed.xml`
- **THEN** the system SHALL return an RSS 2.0 document (`application/rss+xml`) with an `<item>` per published post

### Requirement: Academy with structured courses

The system SHALL provide an academy: courses in `content/academy/<course>/course.md` and ordered lessons in `content/academy/<course>/<NN>-<slug>.md` with frontmatter (`title`, `description`, `level`, `durationMinutes`, `track`), a `/academy` hub grouped by track, `/academy/[course]` syllabus pages, and `/academy/[course]/[lesson]` lesson pages with prev/next navigation. Unknown course or lesson slugs SHALL return 404.

#### Scenario: Hub, syllabus, and lesson

- **WHEN** a visitor requests `/academy`
- **THEN** the hub SHALL list courses grouped by track
- **AND** requesting `/academy/<existing-course>` SHALL render the syllabus with ordered lessons
- **AND** requesting `/academy/<existing-course>/<existing-lesson>` SHALL render the lesson with prev/next navigation

#### Scenario: Unknown academy slug

- **WHEN** a visitor requests an unknown `/academy/<course>` or `/academy/<course>/<lesson>`
- **THEN** the system SHALL render the branded 404 page instead of a syllabus or lesson (status per platform not-found handling, see blog scenario)

### Requirement: Marketing copy in the typed copy layer

Marketing UI strings SHALL live in a `marketing` namespace of `lib/copy/ui.ts`, Spanish-first with an English mirror, following the existing typed `copy()`/`tpl()` pattern. Long-form blog/academy bodies SHALL live in the markdown content tree.

#### Scenario: Typed access

- **WHEN** code accesses a marketing string via `copy("marketing", "<key>")`
- **THEN** it SHALL typecheck against the Spanish namespace with English fallback semantics matching the existing layer

### Requirement: SEO and structured data

Every marketing route SHALL export `generateMetadata` (title template `%s | NexoChat`, description, OG). The system SHALL emit JSON-LD: `Blog` on the blog index, `BlogPosting` on article pages, `Course`/`LearningResource` on academy pages, `FAQPage` on FAQ surfaces, `Offer` on pricing. `app/sitemap.ts` SHALL list all marketing routes including blog and academy entries with per-route priorities.

#### Scenario: Sitemap covers marketing routes

- **WHEN** `/sitemap.xml` is generated
- **THEN** it SHALL include `/`, `/features`, `/pricing`, `/blog`, each published post, `/academy`, each course and lesson, and the legal pages with declared priorities

#### Scenario: Article structured data

- **WHEN** a blog article page renders
- **THEN** the document SHALL include a `BlogPosting` JSON-LD node with `headline`, `datePublished`, and `author`

