# VERDICT — add-enterprise-marketing-website

STATUS: APPROVED

## Staff Security Engineer

- **MED — Content-loader path containment must be explicit (required design note).** `getBlogPost(slug)`, `getCourse(slug)`, `getLesson(courseSlug, lessonSlug)` receive URL-derived params. The design must mandate: validate slugs against a strict charset (`[a-z0-9-]`), resolve the target path and verify it stays inside the `content/` root (reject `..`, absolute paths, and any traversal), and `notFound()` on any miss. As written, the design does not state this contract; the implementation reportedly renders a branded 404 for unknown slugs, which implies handling exists — but containment must be codified, not incidental, because these loaders run server-side on unauthenticated request paths.
- **LOW — Markdown trust boundary (residual risk, acceptable).** `prose.tsx` renders `react-markdown` + `remark-gfm` with no `rehype-raw` (raw HTML is escaped by default). The content tree is repo-authored and build-time static. Risk is acceptable ONLY while content remains a trusted, in-repo, non-user-generated tree; if any future path accepts user or webhook-derived markdown, this boundary changes and sanitization becomes mandatory. Codify "no rehype-raw, content is a trusted tree" in the design.
- **LOW — Custom `parseFrontmatter` regex splitter.** Hand-rolled `---`-fence parsing is fragile but operates only on trusted in-repo files at build time. Residual risk accepted; note that any move to user-generated content must replace it with a real parser (gray-matter) and schema validation.
- **INFO — No new secrets, no new auth surface.** Public unauthenticated routes; Stytch flows untouched; no new env vars; JSON-LD via `dangerouslySetInnerHTML` carries only repo-authored static data. `NEXT_PUBLIC_*` additions must remain non-secret (existing convention). No RBAC/tenant-isolation concerns apply — no data-plane code is introduced.

## Staff DBA

- **N/A — No database surface.** This change introduces no migrations, no SQLC queries, no transaction boundaries, no indexes, no schema. No findings; no residual database risk.

## SRE

- **LOW-MED — Build-time dependency on Google Fonts must be documented.** `next/font/google` fetches at build; a network failure at build time can fail the production build. `font-display: swap` mitigates runtime, not build. Record the mitigation (retry, or vendor the font files) and the expected failure mode in the design; not a blocker.
- **LOW — Motion must respect `prefers-reduced-motion`.** `framer-motion` scroll reveals are decorative; design already states content must be readable without JS (progressive enhancement). Codify reduced-motion handling and keep reveals below the fold non-blocking for rendering.
- **LOW — Observability is optional and deferred (residual).** No RUM/analytics in scope; CSP already permits GTM for a later conversion-tracking add. Acceptable; note that conversion measurement is a post-launch dependency for the marketing goals.
- **INFO — Deploy checklist items, not design defects.** `metadataBase` is the `yourdomain.com` placeholder and OG images are absent; both must be resolved at deployment or social/SEO output is wrong. Content updates require a rebuild (no ISR) — acceptable for the stated scope, document as an operational constraint.
- **INFO — Rollback is clean.** Git-revert only, no infra/tenant state. Verified consistent with the proposal's rollback section.

## Notes

- Findings are codification items for the design doc, not blockers to the delivered behaviour. The change is frontend-only with a minimal attack surface; the two items with enforcement consequence (path containment, markdown trust boundary) are LOW residual risk today but MUST be re-evaluated if content ever becomes user-generated.
