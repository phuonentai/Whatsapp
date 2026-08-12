## ADDED Requirements

### Requirement: MercadoPago adapter is optional
The system SHALL start without MercadoPago credentials: when `MERCADOPAGO_ACCESS_TOKEN` is unset, the backend SHALL boot with Polar-only routing and MP service calls SHALL return a clear "not configured" error.

#### Scenario: Boot without MP credentials
- **WHEN** the backend starts without `MERCADOPAGO_ACCESS_TOKEN`
- **THEN** the server SHALL start successfully
- **AND** the provider router SHALL delegate all organizations to Polar
- **AND** `CreateMPCheckout` and `VerifyMPPayment` SHALL return a configuration error
