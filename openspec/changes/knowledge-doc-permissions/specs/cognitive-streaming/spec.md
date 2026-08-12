# cognitive-streaming Delta Spec

## MODIFIED Requirements

### Requirement: Recuperación RAG con ACL de visibilidad

El path de recuperación RAG (búsqueda de chunks por similitud sobre `cognitive.document_embeddings`) SHALL aplicar el ACL de visibilidad del documento: solo se recuperarán chunks de documentos `workspace` (todos los miembros) y de documentos `admin_only` cuando el miembro solicitante tenga `org:manage`. La recuperación SHALL ocurrir dentro del contexto del org del miembro (aislamiento por tenancy vigente) y SHALL NOT filtrar por UI.

#### Scenario: Búsqueda de un miembro sin permiso

- **WHEN** un miembro sin `org:manage` dispara una búsqueda RAG
- **THEN** los resultados SHALL excluir chunks de documentos `admin_only`
- **AND** las citas del streaming SHALL NOT referenciar documentos restringidos

#### Scenario: Búsqueda de un admin

- **WHEN** un miembro con `org:manage` dispara una búsqueda RAG
- **THEN** los resultados SHALL incluir chunks de documentos `workspace` y `admin_only`
