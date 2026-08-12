# knowledge-doc-permissions Delta Spec

## ADDED Requirements

### Requirement: Visibilidad por documento (workspace | admin_only)

El sistema SHALL almacenar una visibilidad por documento (`visibility`: `workspace` por defecto | `admin_only`). Un documento `workspace` SHALL ser recuperable y visible por todo miembro del org; un documento `admin_only` SHALL ser visible y recuperable únicamente por miembros con `org:manage`. La visibilidad SHALL poder actualizarse solo por `org:manage`. La tenancy SHALL permanecer a nivel de org (sin ACL por miembro individual en v1).

#### Scenario: Documento por defecto workspace

- **WHEN** un miembro sube un documento sin especificar visibilidad
- **THEN** el documento SHALL crearse con `visibility = workspace`

#### Scenario: Admin marca un documento admin_only

- **WHEN** un admin actualiza la visibilidad de un documento a `admin_only`
- **THEN** el cambio SHALL persistirse
- **AND** solo usuarios con `org:manage` SHALL ver/recuperar ese documento

#### Scenario: Miembro sin permiso no ve el documento

- **WHEN** un miembro sin `org:manage` lista o abre un documento `admin_only`
- **THEN** el documento SHALL NOT aparecer en la lista ni en el detalle (sin fuga de título)

### Requirement: ACL en la recuperación RAG

La recuperación de chunks para el chat (`SearchSimilarDocuments` o el path equivalente del RAG) SHALL filtrar por la visibilidad del documento del miembro solicitante: un documento `admin_only` SHALL NOT devolver chunks a un miembro sin `org:manage`. El filtro SHALL vivir en el path de retrieval (server-side), nunca solo en la UI. El ocultamiento en UI sin filtro de recuperación SHALL ser tratado como bug de compliance.

#### Scenario: Chat no recupera docs restringidos

- **WHEN** un miembro sin `org:manage` hace una pregunta al knowledge base
- **THEN** el RAG SHALL NO devolver chunks de documentos `admin_only` del org
- **AND** las citas resultantes SHALL NOT referenciar documentos que el miembro no puede ver

#### Scenario: Admin recupera todo

- **WHEN** un miembro con `org:manage` hace una pregunta al knowledge base
- **THEN** el RAG SHALL poder devolver chunks de documentos `workspace` y `admin_only`

### Requirement: Administración de documentos admin-only

En v1, subir, reprocesar, renombrar, eliminar y cambiar visibilidad de documentos SHALL requerir `org:manage`. Los agentes podrán obtener `knowledge:manage` en un change futuro vía rol Stytch separado sin tocar esta página.

#### Scenario: Subida admin-only

- **WHEN** un miembro sin `org:manage` intenta subir un documento
- **THEN** la UI SHALL ocultar/denegar la subida (sin estado fantasma)

### Requirement: Traceability de documentos en export de compliance

El export de compliance (Ley 1581) SHALL listar los documentos indexados del org con su título, estado y visibilidad, para permitir trazar qué datos de contactos pudieron indexarse.

#### Scenario: Export incluye visibilidad

- **WHEN** un admin ejecuta el export de compliance
- **THEN** el resultado SHALL incluir la lista de documentos con su visibilidad
