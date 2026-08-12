## Context

After the embedded-signup exchange succeeds, `whatsapp-config-section.tsx` swaps the connect empty-state for a green "WhatsApp connected" banner plus the raw config values. There is no guidance. The inbound pipeline (webhook → durable outbox → CRM conversation) already turns inbound messages into conversations, so a user messaging their own business number will produce a real conversation without any new backend.

## Goals / Non-Goals

**Goals:**
- Kill the post-connect dead-end with actionable next steps.
- Teach the "messages arrive going forward" reality.
- Surface Ley 1581 consent expectations and link compliance.
- Guide into the inbox and to enabling the copilot.

**Non-Goals:**
- No backend/test-message endpoint.
- No historical-chat backfill.
- No change to the consent state machine.
- No change to the connect flow itself (embedded signup path intact).

## Decisions

1. **Next-steps card, data-free.** `components/whatsapp/post-connect-steps.tsx` renders a checklist of the five items; it appears whenever a config is active and the user has not dismissed it (dismissal in localStorage). No new completion signals — the card is guidance, not tracking.
2. **Test message via inbound path, not an API.** Copy instructs the user to message the business number from their own phone; the existing webhook pipeline creates the conversation. This is honest and requires zero backend.
3. **Expectation copy is explicit:** only new messages arrive from the moment of connection; historical chats are not imported. This sets consent and data expectations and avoids a support loop.
4. **Consent note links, not re-implements.** The card summarizes that contacts may receive an automatic consent request (Ley 1581) and links to the compliance section; the `whatsapp-compliance` state machine is untouched.
5. **Dismissal state** stored client-side; card re-appears if config becomes inactive and re-activates.
6. **Copy layer** namespace `whatsapp` extended; Spanish-first.

## Risks / Trade-offs

- **Guidance-only card may be ignored or re-dismissed; acceptable — it replaces a true dead-end.**
- **Test-message flow depends on the user's phone; failure mode is user error, mitigated by copy ("check that you message the exact business number").**
- **Dismissal in localStorage is per-browser; acceptable for guidance UI.**
- **Dependency risk:** copy layer from `standardize-spanish-first-copy` must land first.
