## 1. Verify + status mapping [BE-INFRA]

- [x] 1.1 `VerifyMPPayment` fetches the subscription from `s.mpProvider.GetSubscription` instead of the router (`mp_checkout_service.go:139`); keep `SetBillingProvider("mercadopago")` as the last step
- [x] 1.2 `mp_adapter.GetSubscription` maps the preapproval status via `MapMPStatus` before storing (raw `authorized`/`paused`/`cancelled` must never reach the DB)

## 2. Quota metadata [BE-INFRA]

- [x] 2.1 Add `MERCADOPAGO_CHECKOUT_INVOICE_COUNT` / `MERCADOPAGO_BUSINESS_INVOICE_COUNT` (int32, default `0`) to `platform/mercadopago/config.go` and document both in `go-b2b-starter/example.env`
- [x] 2.2 `mp_adapter.CreateCheckoutSession` attaches `metadata.invoice_count_max` to the preapproval body when the plan id maps to a configured quota
- [x] 2.3 `mp_adapter.GetSubscription` parses `metadata.invoice_count_max` from the preapproval search result into `Metadata["invoice_count_max"]` (tolerate numeric and string values)
- [x] 2.4 `mp_webhook_parser.ParseSubscriptionEventData` extracts `data.metadata.invoice_count_max` into `ProductMetadata["invoice_count"]` so `handleSubscriptionUpsert` seeds a nonzero quota
- [x] 2.5 Add adapter/parser unit tests: metadata round-trip, string/number coercion, unknown plan → no metadata

## 3. Webhook payment dispatch [BE-DOMAIN]

- [x] 3.1 `ProcessMPWebhookEvent` payment events dispatch `data.id` (payment id via `ParsePaymentEventData`) instead of the notification id (`process_mp_webhook_event_service.go:44`)
- [x] 3.2 `ParsePaymentEventData` tolerates string payment ids (fallback parse when int64 unmarshal yields 0)
- [x] 3.3 Update `process_mp_webhook_event_service_test.go` fixtures to carry `data.id` and assert the payment id is dispatched

## 4. Optional MP boot [BE-INFRA]

- [x] 4.1 `platform/mercadopago/cmd/init.go` skips config/client registration when `MERCADOPAGO_ACCESS_TOKEN` is empty (no boot panic)
- [x] 4.2 `billing/app/services/module.go`: make the named `mercadopago` binding `optional:"true"` in `billingServiceParams`; guard `CreateMPCheckout`/`VerifyMPPayment`/`CancelMPSubscription` with a clear `mercadopago not configured` error when the adapter is nil
- [x] 4.3 Add a router test: unconfigured MP → all orgs delegate to Polar

## 5. Org context + cancellation persistence [BE-INFRA]

- [x] 5.1 `CreateMPCheckout`/`CancelMPSubscription` read the org id from the Gin context (`c.Get("stytch_org_id")` populated by `RequireOrganization`) and pass it explicitly to the service methods; drop the `ctx.Value("stytch_org_id")` request-context reads (`mp_checkout_service.go:13-16,59-62`)
- [x] 5.2 `CancelMPSubscription` builds the `Subscription` with `CurrentPeriodStart/End` derived from the existing row (fallback: preapproval `next_payment_date`/`end_date`); upsert must not hit the NOT NULL violation (`mp_checkout_service.go:83-92`)
- [x] 5.3 Make `handleSubscriptionCanceled` tolerant of absent period dates for `subscription_cancelled` events (`process_webhook_event_service.go`): keep prior period bounds on the local row
- [x] 5.4 Add tests: checkout/cancel resolve org from Gin context; canceled MP org is inactive locally; cancelled webhook without dates is idempotent

## 6. Webhook ingress + signature hardening [BE-INFRA]

- [x] 6.1 Mount the webhook group as `/v1/webhooks` under the `/api` prefix (match the whatsapp pattern) so `/api/v1/webhooks/polar` and `/api/v1/webhooks/mercadopago` resolve (`billing/routes.go`); add a route test asserting the effective paths
- [x] 6.2 `VerifyWebhookSignature` enforces a timestamp freshness window (5 min) and rejects non-`ts=,v1=` header formats (drop the raw-header fallback) (`mp_webhook_parser.go`); add replay + malformed-header tests
- [x] 6.3 Correct stray `POLAR_WEBHOOK_SECRET` references in `docs/README.md` and `docs/billing.md` to the real `WEBHOOK_SECRET` name used by code and `example.env` (also fixed the same stray reference in `internal/modules/billing/README.md`)

## 7. Verification [OPS-GOV]

- [x] 7.1 Run `go build ./...`, `go vet ./internal/modules/...`, `go test ./...` — all pass
- [x] 7.2 Record verification results and archive decision in `tasks.md`

## Gate results

- `go build ./...` → exit 0
- `go vet ./internal/modules/...` → exit 0
- `go test ./...` → exit 0; 46 packages `ok`, 0 failures

New/updated tests (all pass):
- `internal/modules/billing/infra/mercadopago/mp_adapter_test.go` (new): `TestCreateCheckoutSession_AttachesPlanQuotaMetadata`, `TestCreateCheckoutSession_BusinessPlanQuotaMetadata`, `TestCreateCheckoutSession_UnknownPlanCarriesNoMetadata`, `TestGetSubscription_MapsStatusAndParsesQuotaMetadata`, `TestGetSubscription_StringQuotaMetadata`, `TestGetSubscription_CancelledStatusMapped`
- `internal/modules/billing/infra/mercadopago/mp_webhook_parser_test.go`: `TestParseSubscriptionEventData_ExtractsNumericInvoiceCount`, `TestParseSubscriptionEventData_ExtractsStringInvoiceCount`, `TestParseSubscriptionEventData_NoQuotaMetadata`, `TestParseSubscriptionEventData_KeepsDerivedPeriodBounds`, `TestParsePaymentEventData_NumericPaymentID`, `TestParsePaymentEventData_StringPaymentID`, `TestParsePaymentEventData_MissingID`, `TestVerifyWebhookSignature_RawHeaderFallbackRejected`, `TestVerifyWebhookSignature_ReplayRejected`, `TestVerifyWebhookSignature_FutureTimestampRejected`, `TestVerifyWebhookSignature_MalformedHeaderRejected`
- `internal/modules/billing/app/services/process_mp_webhook_event_service_test.go`: `TestProcessMPWebhookEvent_PaymentEventsDispatchedToClientPayments`, `TestProcessMPWebhookEvent_PaymentEventsDispatchAllTypes`, `TestProcessMPWebhookEvent_PaymentEventsDispatchStringPaymentIDs`, `TestProcessMPWebhookEvent_SubscriptionCancelledWithoutDatesKeepsPriorBounds`, `TestProcessMPWebhookEvent_SubscriptionCancelledWithoutDatesAndWithoutPriorRow`
- `internal/modules/billing/app/services/mp_checkout_service_test.go` (new): `TestCreateMPCheckout_UnconfiguredProviderReturnsClearError`, `TestVerifyMPPayment_UnconfiguredProviderReturnsClearError`, `TestCancelMPSubscription_UnconfiguredProviderReturnsClearError`, `TestCancelMPSubscription_MissingOrgContextErrors`, `TestCreateMPCheckout_ResolvesOrgFromExplicitParam`, `TestCancelMPSubscription_PersistsCanceledRowWithExistingPeriodBounds`, `TestCancelMPSubscription_FallsBackToProviderPeriodBounds`, `TestCancelMPSubscription_NeverZeroPeriodBounds`, `TestVerifyMPPayment_FetchesFromMPProviderAndSetsProviderLast`
- `internal/modules/billing/infra/routing/provider_router_test.go`: `TestProviderRouter_UnconfiguredMPDelegatesAllOrgsToPolar`
- `internal/modules/billing/handler_test.go` (new): `TestCreateMPCheckout_ResolvesOrgFromGinContext`, `TestCreateMPCheckout_MissingGinOrgIsRejected`, `TestCancelMPSubscription_ResolvesOrgFromGinContext`, `TestCancelMPSubscription_MissingGinOrgIsRejected`
- `internal/modules/billing/routes_test.go` (new): `TestRoutes_WebhookPathsResolveUnderSingleAPIPrefix`, `TestRoutes_DuplicatedAPIPrefixNoLongerResolves`

## Changed files

- `internal/platform/mercadopago/config.go` (+2 env fields with defaults)
- `go-b2b-starter/example.env` (document `MERCADOPAGO_CHECKOUT_INVOICE_COUNT` / `MERCADOPAGO_BUSINESS_INVOICE_COUNT`)
- `internal/platform/mercadopago/cmd/init.go` (skip config/client registration when token unset)
- `internal/modules/billing/infra/mercadopago/mp_adapter.go` (status mapping, quota metadata write/read, `NewMPAdapter` takes `*Config`)
- `internal/modules/billing/infra/mercadopago/mp_webhook_parser.go` (subscription quota metadata, string payment ids, signature freshness + format enforcement)
- `internal/modules/billing/app/services/mp_checkout_service.go` (verify via MP adapter, nil guards, explicit org param, cancel period bounds)
- `internal/modules/billing/app/services/process_mp_webhook_event_service.go` (dispatch `data.id`)
- `internal/modules/billing/app/services/process_webhook_event_service.go` (cancel tolerant of absent dates)
- `internal/modules/billing/app/services/subscription_service_dec.go` (interface signatures)
- `internal/modules/billing/app/services/module.go` (optional MP bindings)
- `internal/modules/billing/provider.go` (optional `mpCfg` in handler wiring)
- `internal/modules/billing/handler.go` (resolve org from Gin context)
- `internal/modules/billing/routes.go` (webhook group `/v1/webhooks`)
- `internal/modules/billing/infra/routing/provider_router.go` (Polar-only degradation when MP unconfigured)
- `internal/modules/agent/app/services/agent_service_test.go` (mock signatures follow new interface)
- `docs/README.md`, `docs/billing.md`, `internal/modules/billing/README.md` (`POLAR_WEBHOOK_SECRET` → `WEBHOOK_SECRET`)
- Tests listed above

## Archive decision

**Archive deferred:** centralized verification phase per repo practice.
