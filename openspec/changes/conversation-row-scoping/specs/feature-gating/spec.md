# feature-gating Delta Spec

## ADDED Requirements

### Requirement: Flag de scoping de conversaciones solo en planes pagos (mecanismo de entitlement real)

El sistema SHALL definir el flag `conversation_row_scoping` en el mapeo plan→features del entitlement provider (reconciliado con la implementación real: `FeatureProvider.GetEntitlement` → `Entitlement.Features`; NO existe `FeatureService.IsEnabled` ni `internal/platform/features/plans.go` en el código). El flag SHALL ser `true` solo para organizaciones con suscripción activa/trialing/past_due en un plan pago, y `false` en free tier (sin suscripción) o suscripción inactiva. El canal de grant (metadata de suscripción `crm_features`, módulo vendible con `granted_features`, o grant base tipo `defaultGrantedModules`) SHALL definirse en tasks; los nombres de plan "Starter/Pro/Enterprise" son un supuesto a validar contra el catálogo Polar.

#### Scenario: Plan pago activa el scoping

- **WHEN** una organización tiene una suscripción activa/trialing/past_due en plan pago con el flag concedido
- **THEN** `Entitlement.Features["conversation_row_scoping"]` SHALL ser true
- **AND** el predicado de scope SHALL aplicarse en query layer y RLS

#### Scenario: Free tier conserva org-scope

- **WHEN** una organización no tiene suscripción (o plan free)
- **THEN** el flag SHALL ser false
- **AND** la bandeja SHALL comportarse como antes del change (todas las conversaciones visibles para quienes tengan el permiso de bandeja)

#### Scenario: Suscripción inactiva no activa el scoping

- **WHEN** la suscripción de una organización está inactiva (no activa, no trialing, no grace)
- **THEN** el flag SHALL ser false
- **AND** la bandeja SHALL comportarse org-scope

#### Scenario: RLS respeta el flag vía session var

- **WHEN** el plan no incluye la feature
- **THEN** el middleware SHALL setear la session var de scope de modo que la política RLS permita org-wide (p. ej. `app.is_view_all = true` o flag de scope desactivado)
- **AND** la política RLS SHALL ser una sola, sin lógica temporal de plan embebida

#### Scenario: El flag se resuelve una vez por request

- **WHEN** un request cruza múltiples chequeos de feature en el mismo ciclo
- **THEN** el entitlement SHALL leerse una sola vez por request en el middleware (sin lecturas repetidas de suscripción por query)

### Requirement: UI respeta el flag

La interfaz de bandeja SHALL ocultar los controles de scope (tabs de cola/asignación, picker de re-asignación) cuando el flag `conversation_row_scoping` es false.

#### Scenario: Free tier sin controles de scope

- **WHEN** una organización en free tier abre la bandeja
- **THEN** SHALL NO mostrar tabs de cola ni pickers de asignación
- **AND** la lista SHALL mostrar todas las conversaciones (org-scope)
