## Why

Connecting WhatsApp currently ends in a static green "WhatsApp connected" box with no next step. Users don't learn that messages arrive going forward (not a backfill), don't know how to verify the connection, aren't introduced to consent obligations under Ley 1581, and aren't guided into the inbox or the AI copilot. This dead-end is the biggest drop-off point after purchase.

## What Changes

- Replace the post-connect dead-end with a next-steps onboarding flow on the WhatsApp configuration surface:
  1. **Send a test message** — instruct the user to message their business number from their personal phone and watch it arrive in the inbox (uses the existing inbound webhook path; no new API).
  2. **Set expectations** — a plain-language statement that only new messages arrive from the moment of connection; historical chat is not backfilled.
  3. **Consent note (Ley 1581)** — explain that contacts may be asked for data-treatment consent automatically, with a link to the compliance section.
  4. **First look** — a guided CTA into the inbox.
  5. **Enable the assistant** — a CTA to agent settings to turn on the WhatsApp copilot.
- The next-steps card renders after a successful connect and can be dismissed; it links to the existing inbox, compliance, and agent-settings surfaces.
- Copy resolves through the typed copy layer from `standardize-spanish-first-copy` (Spanish-first).

## Capabilities

### New Capabilities
- `whatsapp-post-connect-onboarding`: the post-connect next-steps flow (test message, go-forward expectations, consent note, inbox first look, assistant CTA).

### Modified Capabilities
- `whatsapp-config-frontend`: the WhatsApp settings view SHALL render the post-connect next-steps flow after a successful connect instead of terminating at the connected banner.

## Impact

- Frontend: `app/dashboard/settings/components/whatsapp-config-section.tsx` (post-connect state), new `components/whatsapp/post-connect-steps.tsx`, copy layer additions.
- Backend: none. The test-message path uses the existing inbound webhook and CRM conversation pipeline; no new endpoint, DB, or Stytch contract change.
- Consent behavior: unchanged — the consent state machine in `whatsapp-compliance` (Ley 1581) is not modified; this change only surfaces expectations and links to the compliance section.
- Rollback: revert the frontend commit in Git; no Stytch tenant policy or provider configuration is changed, so no external rollback applies.
- Non-Goals: no historical-chat backfill (the API does not provide it and this change explicitly does not attempt it), no new send-test-message endpoint, no changes to the consent state machine, no automated messaging behavior change, no local credential or token storage.
