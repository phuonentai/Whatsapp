# Proposal: Knowledge Base v2 — permisos por documento (ACL) + rail de dos modos

## Why

El brief de diseño de Knowledge Base define dos trabajos distintos (preguntar vs administrar) que hoy comparten un rail sin modo, y plantea el cambio de contrato central: **permisos por documento** (`visibility: workspace | admin_only`) con recuperación RAG con ACL server-side — ocultar en UI sin filtrar en recuperación es un bug de compliance. Un índice IA es una superficie de cumplimiento (Ley 1581): un índice envenenado o desactualizado es peor que ninguno. El brief también especifica la UX de vida del documento (upload → processing → processed/failed + retry), citas inline con salto a la fuente, contexto de fuentes, chip de créditos y estados honestos.

## What Changes

- **Backend (Go + migración + SQLC)**:
  - Nueva columna `visibility` en `documents.documents` (`workspace` default | `admin_only`); migración con backfill a `workspace`.
  - `UpdateDocument` acepta `visibility`; `ListDocumentsByOrganization` devuelve solo docs visibles para el miembro (admin ve todos); `GetDocumentByID` aplica el mismo filtro (no filtrar por doc restringido es fuga de título).
  - **ACL en recuperación RAG**: `SearchSimilarDocuments` (cognitive.sql) filtra embeddings por visibilidad del documento del miembro solicitante (`workspace` para todos, `admin_only` solo si `org:manage`) — el filtro vive en el path de retrieval, nunca solo en UI.
  - Compliance: el export Ley 1581 SHALL listar los documentos indexados con su visibilidad (traceability).
  - Subida/borrado/reproceso/renombrado/visibilidad: solo `org:manage` en v1 (un índice es superficie de compliance).
- **Frontend (`app/dashboard/knowledge/*`)**:
  - Rail con **modo segmentado Chat / Documentos** (`?mode=chat|docs` en URL, mismo frame físico que el rail de la bandeja); pane principal cambia thread ⇄ detalle de documento; pane de contexto (xl+: fuentes con score de similitud y salto al doc; en docs mode: metadata del documento).
  - Chat: citas inline `[1][2]` + tarjetas de fuente (título, icono por tipo, score) + botón copiar + "Fuentes"; estado honesto "No encontré esto en tus documentos" cuando RAG no encuentra chunks; streaming sin cambios (SSE existente).
  - Docs: lista con chips de estado (procesando amber / listo emerald / error red + Retry), detalle con metadata (tamaño, fecha, visibilidad), Reprocesar/Renombrar/Eliminar con ConfirmDialog; badge de lock en docs `admin_only`; miembro que abre link a doc restringido ve 403 "Documento restringido" (sin título filtrado).
  - Header: chip de créditos (340/1000) amber ≥80%, botón "Agregar documento", composer deshabilitado con banner cuando créditos = 0 (402 pre-stream); tokens usados por mensaje en tooltip.
  - Empty state guiado: "Agrega tu primer documento" + botón de PDF de muestra + explicación.
  - 402 guard: composer disables antes de enviar (input disabled + banner "Créditos agotados — ver plan").
  - a11y: streaming en live region, citas como links reales, Enter send / Shift+Enter newline, estados nunca color-only.

## Capabilities

### New Capabilities

- `knowledge-doc-permissions`: modelo de visibilidad por documento (`workspace`|`admin_only`) con ACL server-side en recuperación RAG, administración admin-only (upload/delete/visibility), y traceability en el export de compliance. (Corresponde al item "Knowledge document permissions (visibility ACL)" del inventario.)

### Modified Capabilities

- `knowledge-base-ui`: la UI pasa a rail de dos modos (Chat/Documentos), detalle de documento con visibilidad, citas inline con salto a fuente, chip de créditos y estados de vida del documento; el gate del módulo sigue por entitlement `ai_assistant`.
- `cognitive-streaming`: la recuperación RAG SHALL aplicar el ACL de visibilidad de documentos; el chat nunca recupera chunks de docs que el miembro no puede ver.

## Impact

- **Backend**: `go-b2b-starter/` — migración (columna `visibility` + backfill), SQLC (`documents.sql`, `cognitive.sql`), adapters (`cognitive_store.go`, `document_listener.go`/`rag_service.go`), endpoints de documentos (list/detail/update visibility), export compliance. `make sqlc`, `make test`.
- **Frontend**: `next_b2b_starter/app/dashboard/knowledge/*` (rail, detalle, citas, créditos) + `lib/copy/ui.ts` + `lib/api/api/repositories/document-repository.ts` (param `visibility`).
- **Auth**: sin cambios de contrato Stytch (se usa el rol efectivo `org:manage` del miembro para el ACL); sin nuevos permisos Stytch en v1 (upload/delete/visibility admin-only vía permiso existente).
- **Dependencias**: ninguna nueva.
- **Ops**: `make test` (Go), `pnpm build`/`lint`/`tsc`, vitest de knowledge (document-upload, document-sources, chat-message) — actualizar si dependen de estructura; Playwright visual/a11y → `qa/`.
- **Rollback**: migración reversible (drop columna con default); git revert de ambos repos.
- **Non-Goals**: sin editor de roles (change aparte); sin ACL por miembro individual (el org es la frontera de tenancy — solo 2 niveles); sin "agente puede gestionar docs" en v1 (change futuro vía rol Stytch separado); sin almacenar credenciales localmente (todo auth sigue en Stytch B2B).

## Assumptions

- El modelo de documentos actual (`DocumentsDocument`) no tiene `visibility` — verificado; la columna es nueva.
- `SearchSimilarDocuments` es el path de recuperación único usado por el RAG (`rag_service.go`) — verificar en tasks; si hay otros paths, todos aplican el filtro.
- El export de compliance (Ley 1581) puede extenderse para listar docs indexados con visibilidad sin romper su contrato actual.
- Los créditos ya se exponen (`useAiUsageQuery`) para el chip del header.
