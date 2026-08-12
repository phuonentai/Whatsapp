## 1. Spike: PDF + link mechanics [OPS-GOV]

- [ ] 1.1 Verify whether the file-asset layer can serve public/shared URLs (or whether a public quote-view route is needed). Verify: findings recorded in tasks.md note + design.md Open Questions updated; fallback = minimal public HTML quote-view route
- [ ] 1.2 Spike PDF library: evaluate Go-native (gofpdf/maroto) vs HTML→PDF for Spanish glyph coverage (ñ, á, é), itemized table layout, and logo embedding; document tradeoff + chosen lib. Verify: findings recorded; `DocumentRenderer` interface unchanged by choice
- [ ] 1.3 Verify quote numbering counter precedent (repo has no local sequence util; invoicing uses provider-side numeration). Verify: findings recorded; approach in add-quote-documents task 3.2 confirmed or adjusted

## 2. Renderer + template registry [BE-DOMAIN]

- [ ] 2.1 Define `domain.DocumentData` (transport-free: quote items, totals, client info, branding snapshot key), `TemplateID`, `DocumentRenderer` + `TemplateRegistry` interfaces with NO PDF lib imports in domain. Verify: `go build ./...`; grep confirms domain purity
- [ ] 2.2 Implement `cotizacion` template layout + branding resolution via `DocumentBrandingProvider` (snapshot-key aware; default fallback). Verify: golden-file render test asserts logo, Spanish text, items, totals

## 3. PDF infra [BE-INFRA]

- [ ] 3.1 Implement renderer for chosen library: embedded Unicode font, header (logo/letterhead/colors), items table, subtotal/IVA/total, terms footer, validity line. Verify: render tests assert Spanish text + totals; `go vet ./...` EXIT 0
- [ ] 3.2 Store rendered PDFs via file-asset manager with quote purpose/category + link to quote row. Verify: unit tests assert asset created + linked

## 4. Shareable link / public view [BE-INFRA]

- [ ] 4.1 Implement shareable link mechanism per spike 1.1 (public asset URL or public quote-view route with unguessable token, expiry tied to `valid_until`). Verify: tests assert unguessable token + expiry behavior; expired link returns 410/expired state
- [ ] 4.2 Re-send endpoint `POST /api/quotes/:id/resend` (org:manage) using stored asset. Verify: handler test asserts re-send works without re-render when asset exists

## 5. Delivery on send transition [BE-DOMAIN]

- [ ] 5.1 Implement `DocumentSender` interface + `linkSender` (WhatsApp text message with link via existing `OutboundService.SendMessage`). Verify: unit tests with mock sender assert payload + non-fatal failure
- [ ] 5.2 Wire render → store → host → send into the `enviada` transition (render/store failure fails transition; send failure logged, state advances); add notified-status guard on `crm.quotes` (migration addition or payload field). Verify: transition tests cover render-fail (stays borrador), send-fail (enviada + logged), once-only notification

## 6. DI wiring [BE-INFRA]

- [ ] 6.1 Wire renderer, registry, branding provider, file-asset host, sender, resend handler in DI (named bindings per invoicing pattern). Verify: `go build ./...` EXIT 0; server boot defers to live env

## 7. Launch gate [OPS-GOV]

- [ ] 7.1 Run full gate: `make sqlc` (if any migration), `go build ./...`, `go vet ./...`, `make test`. Verify: all EXIT 0, results recorded here
- [ ] 7.2 Frontend regression (no FE changes expected): `pnpm lint` at baseline, `npx tsc --noEmit`, `pnpm build`. Verify: recorded here
