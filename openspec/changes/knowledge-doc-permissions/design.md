# Design: Knowledge Base v2 — permisos por documento (ACL) + rail de dos modos

## Context

- Brief: dos trabajos (preguntar vs administrar) en un rail con modo segmentado; permisos por documento como pieza central; ACL en retrieval (server-side); vida del documento (processing/retry); citas con salto a fuente; créditos ambientales; estados honestos.
- Estado actual: `app/dashboard/knowledge/*` — sidebar (sesiones + docs + upload) / chat streaming (SSE, markdown, citas con título, copy, 402 guard), permisos coarse `resource:*`.
- Backend: `documents.documents` sin `visibility`; `SearchSimilarDocuments` (cognitive.sql) sin filtro de visibilidad; `rag_service.go` consume embeddings; export compliance existe (Ley 1581).
- Frontend: `useAiUsageQuery` expone créditos; `useDocumentsQuery`/`useUploadDocument`/`useDeleteDocument`; entitlements `ai_assistant`.

## Goals / Non-Goals

**Goals:**
- Visibilidad por documento (2 niveles) con ACL en el path de retrieval.
- Rail de dos modos + detalle de documento + contexto de fuentes.
- Citas inline saltables, estados de vida del documento, chip de créditos, empty state guiado.

**Non-Goals:**
- NO ACL por miembro individual (el org es la frontera; 2 niveles).
- NO agente con `knowledge:manage` en v1.
- NO cambiar el modelo de streaming/tenancy vigente.

## Decisions

1. **Modelo de visibilidad mínimo (2 niveles)** — `visibility: workspace | admin_only` con default `workspace`. Un ACL completo por miembro sobrepasaría la frontera de tenancy y el caso de uso real (docs de contacto con datos personales → admin_only como respuesta documentada a consent). Alternativa (ACL completo) se descarta: complejidad sin beneficio para orgs single-tenant.
2. **ACL en SQL del retrieval** — `SearchSimilarDocuments` se modifica para filtrar por visibilidad en el JOIN con `documents`: `(visibility = 'workspace' OR (visibility = 'admin_only' AND $rol = admin))`. El filtro en la query garantiza que ningún path de recuperación lo salte; alternativa (filtrar en Go tras la búsqueda) se descarta: podría filtrar menos y filtrar tarde.
3. **Listas y detalle con el mismo filtro** — `ListDocumentsByOrganization`/`GetDocumentByID` reciben la visibilidad del miembro; un doc `admin_only` para un miembro sin permiso se comporta como inexistente (sin fuga de título ni del 404).
4. **Admin-only para administración** — upload/delete/rename/visibility son `org:manage` en v1 (índice = superficie de compliance). Agente podrá gestionar luego vía rol Stytch separado.
5. **Rail de dos modos** — `?mode=chat|docs` en URL, pane principal alterna, contexto xl+ (drawer <1280px, oculto móvil). Mismo frame que la bandeja.
6. **Citas inline + salto a fuente** — el streaming ya da `similar_documents`; los marcadores `[n]` se renderizan y mapean a tarjetas con score legible ("92% coincidencia"); clic → `?mode=docs&doc=<id>` con scroll al chunk. Fallback "Documento no disponible" para docs eliminados.
7. **Anti-hallucination honesto** — sin chunks relevantes → "No encontré esto en tus documentos — ¿puedes agregarlo?" (decisión de confianza, no detalle de código).
8. **Créditos ambientales** — chip en header (amber ≥80%); composer disabled + banner cuando `credits_remaining = 0` (402 es pre-stream); tokens por mensaje en tooltip. El gate de plan entero (`ai_assistant`) → upgrade gate completo.
9. **Traceability compliance** — el export Ley 1581 añade la lista de docs indexados con visibilidad; no rompe su contrato actual.

## Risks / Trade-offs

- [Migración con datos existentes] → Mitigación: backfill a `workspace` en la misma migración; rollback = drop columna.
- [Path de retrieval alternativo sin filtro] → Mitigación: tasks exigen grep de todos los consumers de `SearchSimilarDocuments`/embeddings; el filtro vive en la query SQL compartida.
- [Score de similitud mostrado como número] → Mitigación: etiquetas legibles ("92% coincidencia"), nunca raw JSON (regla de la app).
- [Citas a docs eliminados] → Mitigación: fallback "Documento no disponible" en la tarjeta de cita.
- [Regresión en streaming] → Mitigación: tests de `chat-message`/`document-sources` existentes; el cambio UI no toca el path SSE.

## Market & Unit Economics

Este change **sí toca la economía de cumplimiento y de IA**, y lo declara:

- **Costo de IA: sin delta en la invocación, con cambio en la superficie de datos.** No se añaden llamadas LLM ni se cambia el metering; el RAG filtra chunks por visibilidad (mismo costo por consulta). La columna `visibility` no añade embeddings ni consumo nuevo.
- **Compliance (Ley 1581) como valor de mercado.** El ACL por documento es la respuesta documentada para docs con datos de contacto cuando `consent_required` está activo; el export de compliance gana traceability de docs indexados. Esto es un diferenciador para el mercado colombiano (Habeas Data) y reduce riesgo regulatorio de clientes; no cambia precios/planes.
- **Precios / créditos: sin cambio.** No se tocan `paywall`, `plan-pricing-ux` ni `ai-usage-metering`; el chip de créditos reutiliza `useAiUsageQuery`.
- **Métrica de adopción:** la UX de citas saltables y el estado honesto "no encontré" pueden medirse como adopción del knowledge base (follow-up de analytics, no en este change).

## Market Risk

- **R1 — Índice como superficie de compliance.** Un documento `admin_only` mal marcado (default `workspace`) puede exponer datos personales en RAG. **Owner:** product/compliance. **Trigger:** incidente de exposición o auditoría; revisión de defaults con el equipo legal. **Mitigación:** default `workspace` es explícito y documentado; el export de compliance lista visibilidad (traceability); la administración (upload/delete/visibility) es admin-only.
- **R2 — Expectativa de gestión por miembros.** La decisión v1 de "solo admin gestiona documentos" puede chocar con orgs donde miembros suben docs. **Owner:** product. **Trigger:** feedback de clientes o solicitudes de subida por no-admin. **Mitigación:** documentado como change futuro vía rol Stytch separado (`knowledge:manage`); la UI oculta la subida para no-admin sin fricción.
- **R3 — Deriva de claims del asistente (anti-hallucination).** El estado honesto "No encontré esto en tus documentos" es un diferenciador de confianza, pero puede leerse como limitación frente a asistentes que "responden todo". **Owner:** product/GTM. **Trigger:** feedback cualitativo o comparativas competitivas. **Mitigación:** copy clara + CTA a agregar el documento.

## Migration Plan

1. **DB**: migración `add_documents_visibility` (columna + default + backfill); `make sqlc` regenera queries.
2. **Backend**: actualizar `documents.sql` (list/detail/update con visibility), `cognitive.sql` (SearchSimilarDocuments con ACL), adapters (reciben rol del contexto org), export compliance.
3. **Frontend**: rail de dos modos → detalle → citas/fuentes → créditos → estados.
4. **Gates**: `make test` (Go), `make sqlc` sin diff, vitest knowledge, lint/build/tsc, Playwright visual/a11y.
5. **Rollback**: drop columna + git revert en ambos repos; sin estado Stytch que revertir.

## Open Questions

- ¿El rol efectivo (`org:manage`) está disponible en el contexto del servicio RAG para el filtro? (spike en tasks; si no, se pasa del middleware de org_context).
- ¿El export compliance actual lista documentos? (si no, se extiende sin romper contrato).
- ¿`similar_documents` del streaming incluye el id de documento para el salto a fuente? (verificar DTO en tasks).
