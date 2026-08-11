## 1. Next-steps card

- [ ] 1.1 [FE-NEXT] Create `components/whatsapp/post-connect-steps.tsx` with the five guidance items (test message, go-forward expectations, consent note, inbox link, assistant CTA) and client-side dismissal state.
- [ ] 1.2 [FE-NEXT] Render the card in `whatsapp-config-section.tsx` whenever the configuration is active and not dismissed; clear dismissal when config deactivates so it reappears on reactivation.
- [ ] 1.3 [FE-NEXT] Add copy keys under `lib/copy` namespace `whatsapp` for the next-steps card items and links.

## 2. Tests

- [ ] 2.1 [FE-NEXT] Add component tests for: card render after active config, dismiss persistence, reappearance on reactivation, and that no test-message API is invoked.
- [ ] 2.2 [FE-NEXT] Update WhatsApp-config section tests to cover the post-connect state.

## 3. Verification

- [ ] 3.1 Run `pnpm lint` in `next_b2b_starter` — must pass.
- [ ] 3.2 Run `pnpm build` in `next_b2b_starter` — must pass.
- [ ] 3.3 Run affected WhatsApp-config and post-connect component tests — must pass.
- [ ] 3.4 Confirm no new backend endpoint/route was introduced (grep for send-test/test-message handlers) — none must exist.
- [ ] 3.5 Record results and archive decision in this file after completion.
