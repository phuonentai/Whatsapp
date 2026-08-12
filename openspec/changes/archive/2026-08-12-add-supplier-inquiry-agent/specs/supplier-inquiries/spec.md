## ADDED Requirements

### Requirement: Supplier registry with contact linkage

The system SHALL store suppliers org-scoped in `procurement.suppliers` (id, organization_id, contact_id, nit, delivery_days, min_order_amount, notes, is_active, timestamps), where `contact_id` SHALL reference `crm.contacts` with `ON DELETE RESTRICT` and rows SHALL be unique per `(organization_id, nit)`. Creating a supplier SHALL create or link the supplier's CRM contact in the same transaction (NIT via `tipo_documento = 'NIT'` and `numero_documento`), SHALL record the contact's consent as an org-declared grant (`consent_status = 'granted'`, `consented_at` set) with a `consent_grant` audit recording `basis = 'org_declared'` and the acting `stytch_member_id` (the organization is the data controller; for NIT persona jurídica contacts this is a documented departure from the subject-initiated consent pipeline; Habeas Data rights flow through the existing agent compliance endpoints), and SHALL audit `supplier_created`. Supplier writes SHALL require `org:manage`; reads SHALL require `org:view`.

#### Scenario: Create supplier creates linked contact with granted consent

- **WHEN** an `org:manage` member creates a supplier with a NIT via `POST /api/procurement/suppliers`
- **THEN** the system SHALL create a `crm.contacts` row with `tipo_documento = 'NIT'`, `numero_documento` set to the NIT, and `consent_status = 'granted'` with `consented_at` set in the same transaction
- **AND** SHALL insert the `procurement.suppliers` row referencing that contact
- **AND** SHALL audit `supplier_created` and `consent_grant` with `basis = 'org_declared'` and the acting `stytch_member_id`

#### Scenario: Duplicate NIT per organization rejected

- **WHEN** an `org:manage` member attempts to create a second supplier with an NIT already registered in the same organization
- **THEN** the system SHALL return HTTP 400 with a Spanish error
- **AND** SHALL NOT create a duplicate contact or supplier row

#### Scenario: Supplier contact deletion is restricted

- **WHEN** an attempt is made to delete a contact referenced by a `procurement.suppliers` row
- **THEN** the database SHALL reject the deletion (FK `ON DELETE RESTRICT`)
- **AND** the system SHALL return a Spanish error suggesting deactivation instead

#### Scenario: Supplier write denied without org:manage

- **WHEN** a member without `org:manage` attempts to create or update a supplier
- **THEN** the system SHALL return HTTP 403 with a Spanish error message

#### Scenario: Supplier list is org-scoped

- **WHEN** a member with `org:view` calls `GET /api/procurement/suppliers`
- **THEN** the response SHALL contain only suppliers of the member's organization

### Requirement: Product catalog with SKU tracking

The system SHALL store org-scoped products in `procurement.products` (id, organization_id, name, sku, unit, is_active, timestamps). Product writes SHALL require `org:manage`; reads SHALL require `org:view`. Deactivating a product SHALL keep its history (inquiry runs, responses, orders) intact.

#### Scenario: Create product

- **WHEN** an `org:manage` member creates a product via `POST /api/procurement/products`
- **THEN** the system SHALL persist the product with `is_active = true`

#### Scenario: Product deactivation preserves history

- **WHEN** an `org:manage` member sets `is_active = false` on a product referenced by past inquiry runs
- **THEN** the product row SHALL remain and past runs SHALL keep their reference

#### Scenario: Product read denied without org:view

- **WHEN** a member without `org:view` calls `GET /api/procurement/products`
- **THEN** the system SHALL return HTTP 403 with a Spanish error message

### Requirement: Inquiry run creation with AI-drafted messages

The system SHALL create inquiry runs via `POST /api/procurement/runs` with `product_ids`, `supplier_ids`, and a free-text `nota`, storing the run in `procurement.inquiry_runs` with status `draft`, `source = 'manual'`, and `schedule_ref` NULL. The system SHALL draft exactly one personalized Spanish WhatsApp inquiry message per supplier through the metered LLM client, one call per supplier (JSON contract `{"message": "..."}`); each draft SHALL greet the supplier by name, list the requested products (name × quantity from the run request), and ask for availability, price, and lead time. The drafting prompt SHALL contain no contact PII: the supplier display name is permitted for NIT contacts only, passed through the `whatsapp-compliance` masking decorator via a business-identity allowlist (persona jurídica — corporate identity, not personal data); documents and phone numbers SHALL always be masked, and non-NIT contacts SHALL use the `[NOMBRE]` placeholder. When the organization's AI credits are exhausted, the system SHALL transition the run to `escalated` and SHALL NOT make any unmetered LLM call.

#### Scenario: Run creation drafts one message per supplier

- **WHEN** an `org:manage` member creates a run for 2 suppliers and 3 products
- **THEN** the system SHALL make exactly 2 metered LLM calls (one per supplier)
- **AND** SHALL store one drafted message per supplier containing the supplier display name, the product names with quantities, and requests for availability, price, and lead time

#### Scenario: Drafting credits exhausted escalates the run

- **WHEN** the organization's AI credits are exhausted during drafting
- **THEN** the system SHALL mark the run `escalated`
- **AND** SHALL NOT invoke the LLM (no unmetered call)
- **AND** SHALL audit the escalation

#### Scenario: Drafting is metered and recorded

- **WHEN** a drafting LLM call completes
- **THEN** the tokens SHALL be recorded in the ai-usage ledger for the organization via the metered client

### Requirement: Inquiry run fan-out via durable outbox

The system SHALL enqueue one durable outbox event per supplier send when a run is sent, and SHALL send each drafted message through the circuit-breakered WhatsApp client (`pkg/whatsapp/client.go`) via the existing outbound send path, transitioning the recipient `pending → sent` with the provider message id. The dispatcher SHALL retry failed sends with exponential backoff and SHALL dead-letter after the maximum attempts; a run whose sends all fail SHALL transition to `failed`. Fan-out SHALL pace sends at most 10 messages per 10 seconds per organization and SHALL record the `outside_24h_window` warning per `whatsapp-outbound-send` when applicable. Send handlers SHALL be idempotent: a re-dispatched event SHALL NOT send twice. Send handlers SHALL re-validate run and recipient state transaction-isolated inside the dispatch claim (run not `cancelled`/`escalated`, recipient still `pending`, kill switch off) before invoking the client; a blocked dispatch SHALL be audited with the reason and SHALL NOT invoke the client.

#### Scenario: Run send enqueues one outbox event per recipient

- **WHEN** a drafted run with 3 recipients is sent
- **THEN** the system SHALL enqueue 3 outbox events (one per supplier)
- **AND** each successful send SHALL mark its recipient `sent` with the provider message id
- **AND** the run SHALL transition to `awaiting_responses` once all sends succeed

#### Scenario: Send failure retries then dead-letters

- **WHEN** a recipient send fails repeatedly through the dispatcher
- **THEN** the outbox event SHALL be retried with exponential backoff
- **AND** after the maximum attempts SHALL be dead-lettered and the recipient marked `failed`
- **AND** the run SHALL transition to `failed` when no recipient remains sendable

#### Scenario: Fan-out respects the rate limit

- **WHEN** a run drafts more than 10 messages for a burst window
- **THEN** the fan-out SHALL pace sends to at most 10 messages per 10 seconds per organization
- **AND** SHALL NOT exceed the burst at any point

#### Scenario: Re-dispatched send does not double-send

- **WHEN** an outbox event is dispatched again after a previous success (e.g., dispatcher completion failure)
- **THEN** the handler SHALL detect the recipient already `sent` (transaction-isolated state check) and SHALL NOT send again

#### Scenario: Kill switch after enqueue blocks the dispatch

- **WHEN** the organization's kill switch is enabled after an `inquiry_send` event is enqueued but before it is dispatched
- **THEN** the dispatch handler SHALL re-validate the run/recipient state transaction-isolated inside the claim, SHALL NOT invoke the WhatsApp client, and SHALL audit the block with reason `kill_switch`

### Requirement: Structured reply extraction from supplier messages

The system SHALL subscribe to `whatsapp.message.received` events as an independent subscriber and, for a message whose sender is an active inquiry-run recipient (run in `sending` or `awaiting_responses` status), SHALL run exactly one metered LLM extraction call — PII masked per `whatsapp-compliance` before the call — returning JSON `{"items":[{"product_name", "sku"?, "disponible": bool, "precio_unitario": number?, "moneda": "COP"?, "cantidad_disponible": number?, "tiempo_entrega": text?, "requiere_seguimiento": bool}], "resumen": text, "requiere_humano": bool}`. The system SHALL persist the extraction in `procurement.inquiry_responses`, idempotent on `(recipient_id, raw_message_id)`, transition the recipient to `answered`, and SHALL flag both the recipient and the run `escalated` when `requiere_humano` is true (low confidence, ambiguity, negotiation mentions, price ranges, "depende") — escalated responses SHALL NOT be auto-quoted anywhere. Exhausted AI credits SHALL escalate without an unmetered call. Messages from non-recipients SHALL NOT trigger extraction.

#### Scenario: Reply from an active recipient is extracted

- **WHEN** a `whatsapp.message.received` event arrives from a contact that is a recipient of a run in `sending` or `awaiting_responses` status
- **THEN** the system SHALL run exactly one metered LLM extraction call with PII masked
- **AND** SHALL persist the extracted response and transition the recipient to `answered`

#### Scenario: Requires-human reply escalates

- **WHEN** extraction sets `requiere_humano = true` (e.g., price ranges, negotiation mentions, or "depende")
- **THEN** the recipient and the run SHALL be flagged `escalated`
- **AND** SHALL NOT auto-quote the extracted prices anywhere

#### Scenario: Credits exhausted escalates without unmetered call

- **WHEN** the organization's AI credits are exhausted before extraction
- **THEN** the system SHALL escalate the recipient and the run
- **AND** SHALL NOT invoke the LLM

#### Scenario: Re-delivered message is not double-extracted

- **WHEN** a webhook redelivery carries a message already persisted for the recipient
- **THEN** the system SHALL NOT run a second extraction
- **AND** SHALL NOT create a second response row

#### Scenario: Message from a non-recipient skips extraction

- **WHEN** a message arrives from a contact that is not an active run recipient
- **THEN** the procurement subscriber SHALL return without invoking the LLM

### Requirement: Aggregation board with deterministic ranking

The system SHALL expose `GET /api/procurement/runs/:id` returning per-supplier recipient status and extracted items, sorted deterministically by availability descending, then unit price ascending, then lead time ascending. The board SHALL optionally include a summary paragraph generated by one metered LLM call; when AI credits are exhausted, the board SHALL return without the summary and SHALL NOT call the LLM. Reads SHALL require `org:view`.

#### Scenario: Board returns deterministically ranked comparison

- **WHEN** an `org:view` member requests a run whose responses contain available and unavailable items with different prices
- **THEN** the response SHALL list suppliers ordered by availability (available first), then unit price ascending, then lead time ascending

#### Scenario: Board summary is metered and optional

- **WHEN** a board summary LLM call succeeds
- **THEN** the tokens SHALL be recorded in the ai-usage ledger and the summary SHALL be included in the response

#### Scenario: Credits exhausted returns board without summary

- **WHEN** AI credits are exhausted for an organization requesting a board summary
- **THEN** the board SHALL be returned without the summary
- **AND** SHALL NOT invoke the LLM

#### Scenario: Board read denied without org:view

- **WHEN** a member without `org:view` requests a run board
- **THEN** the system SHALL return HTTP 403 with a Spanish error message

### Requirement: Order placement with human approval

The system SHALL expose `POST /api/procurement/runs/:id/orders` with body `{supplier_id, items: [{product_id, quantity}], notes}` requiring `org:manage`. Order placement SHALL require the supplier's inquiry response in `answered` status with `requiere_humano = false`, or an explicit `override` by an `org:manage` member; otherwise the system SHALL return HTTP 400 with a Spanish error. On placement the system SHALL, in one transaction, insert a `procurement.orders` row (UNIQUE `(run_id, supplier_id)`), enqueue the pre-composed Spanish order-confirmation text as a durable outbox event (`procurement.order_confirm_send`) through the existing outbound send path (plain text; template infrastructure is a separate change), create a CRM negocio (deal) in the organization's default pipeline (`es_predeterminado`, `estado = 'abierto'`, `moneda = 'COP'`) plus an activity on the supplier contact's timeline, and audit `order_placed`. The order send handler SHALL re-validate at dispatch time, transaction-isolated inside the claim, that the kill switch is off and the contact's consent is not `withdrawn` before invoking the client; a blocked send SHALL be audited with the reason while the order/negocio/actividad remain recorded. A retried placement POST SHALL be idempotent: the UNIQUE `(run_id, supplier_id)` marker SHALL cause the retry to return the existing order without a second send or a second deal. Extracted prices SHALL NOT be quoted to customers by this change.

#### Scenario: Order placed for an answered supplier

- **WHEN** an `org:manage` member places an order for a supplier whose response is `answered` with `requiere_humano = false`
- **THEN** the system SHALL insert a `procurement.orders` row and enqueue the Spanish order-confirmation text via the durable outbox in the same transaction
- **AND** SHALL create a negocio in the default pipeline with `estado = 'abierto'` and `moneda = 'COP'`
- **AND** SHALL create an activity on the supplier contact's timeline
- **AND** SHALL audit `order_placed`

#### Scenario: Retried order placement is idempotent

- **WHEN** an `org:manage` member re-POSTs an order already placed for the same `(run_id, supplier_id)` (e.g., double-click or client retry)
- **THEN** the system SHALL return the existing order
- **AND** SHALL NOT send a second confirmation or create a second deal

#### Scenario: Dispatch-time kill switch blocks an enqueued confirmation

- **WHEN** the kill switch is enabled after the confirmation event is enqueued but before dispatch
- **THEN** the dispatch handler SHALL NOT send
- **AND** SHALL audit the block with reason `kill_switch`
- **AND** the order, negocio, and actividad SHALL remain recorded

#### Scenario: Order without answered response is rejected

- **WHEN** an order is placed for a supplier whose response is not `answered`, or is flagged `requiere_humano`, without an explicit override
- **THEN** the system SHALL return HTTP 400 with a Spanish error message
- **AND** SHALL NOT send a confirmation or create a deal

#### Scenario: Override allows escalation-flagged order

- **WHEN** an `org:manage` member places an order with explicit `override` for a supplier response flagged `requiere_humano`
- **THEN** the system SHALL proceed with the send and the deal creation
- **AND** SHALL audit the override with the acting Stytch `member_id`

#### Scenario: Consent withdrawn blocks the order send

- **WHEN** an order confirmation is evaluated (at placement or at dispatch time) for a supplier contact with consent `withdrawn`
- **THEN** the system SHALL NOT send the confirmation message
- **AND** SHALL record the deal and activity, and audit the blocked send with reason `consent_withdrawn`

#### Scenario: Kill switch blocks order sends

- **WHEN** the organization's kill switch is enabled and an order placement is attempted
- **THEN** the system SHALL NOT send the confirmation
- **AND** SHALL audit the block with reason `kill_switch`
- **AND** SHALL return a Spanish error

### Requirement: Inquiry run lifecycle with governance

The system SHALL track run status (`draft`, `sending`, `awaiting_responses`, `completed`, `partially_answered`, `failed`, `escalated`, `cancelled`) in `procurement.inquiry_runs`, the flow row for the run. The run SHALL transition `draft → sending → awaiting_responses`, then to `completed` when every recipient answered, `partially_answered` when some recipients `timed_out`/`failed` and others answered, `failed` when no recipient is sendable, and `escalated` or `cancelled` from any in-progress state. Recipients SHALL transition `sent → timed_out` lazily on read: any board or run-status read of a run in `awaiting_responses` SHALL reconcile recipients `sent` with `sent_at` older than the response window (fixed 24h default) to `timed_out` and SHALL re-evaluate the run's terminal transition, transaction-isolated and idempotent. Escalation SHALL always be allowed (mirroring `agent-governance`), and enabling the organization's kill switch SHALL cancel in-progress runs and SHALL NOT dispatch further sends.

#### Scenario: Run completes when all recipients answered

- **WHEN** every recipient of a run in `awaiting_responses` has status `answered`
- **THEN** the run SHALL transition to `completed`

#### Scenario: Partial answers mark the run partially answered

- **WHEN** some recipients answered and others `timed_out` or `failed`
- **THEN** the run SHALL transition to `partially_answered`

#### Scenario: Kill switch cancels in-progress runs

- **WHEN** the organization's kill switch is enabled while a run is `sending` or `awaiting_responses`
- **THEN** the run SHALL transition to `cancelled`
- **AND** pending sends SHALL NOT be dispatched (dispatch handlers re-validate run state inside the claim)
- **AND** the action SHALL be audited as `skip` with reason `kill_switch`

#### Scenario: Unanswered recipient times out lazily on read

- **WHEN** a run in `awaiting_responses` is read and a recipient `sent` more than 24h ago has not answered
- **THEN** the read SHALL transition that recipient to `timed_out`
- **AND** SHALL transition the run to `completed` (all answered) or `partially_answered` (some answered, rest `timed_out`/`failed`) accordingly
- **AND** a repeated read SHALL NOT re-transition or duplicate the reconciliation

#### Scenario: Escalation always allowed

- **WHEN** a run is escalated under any organization settings, including kill switch on
- **THEN** the run SHALL transition to `escalated`

### Requirement: Procurement routes are RBAC- and compliance-protected

The system SHALL require the existing `org:manage` permission for all procurement writes and approvals (suppliers, products, run creation/send, order placement) and `org:view` for reads (lists, boards), with all routes behind the existing `auth` + `org_context` + `subscription` middleware like other CRM routes. User-facing errors SHALL be in Spanish. Supplier contact export/forget SHALL reuse the existing agent compliance endpoints (`GET /api/agent/compliance/export/:contactId`, `POST /api/agent/compliance/forget/:contactId`) — no new compliance surface is introduced.

#### Scenario: Unauthenticated request is rejected

- **WHEN** an unauthenticated request hits any `/api/procurement/...` route
- **THEN** the system SHALL return HTTP 401

#### Scenario: Write without org:manage is denied

- **WHEN** a member without `org:manage` attempts to send a run or place an order
- **THEN** the system SHALL return HTTP 403 with a Spanish error message

#### Scenario: Supplier contact forget reuses agent compliance

- **WHEN** an `org:manage` member calls `POST /api/agent/compliance/forget/:contactId` for a supplier contact
- **THEN** the existing agent endpoint SHALL anonymize the contact and set consent `withdrawn`
- **AND** procurement rows SHALL keep referencing the anonymized contact
