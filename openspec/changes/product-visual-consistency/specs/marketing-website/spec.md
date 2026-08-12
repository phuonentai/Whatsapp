# marketing-website Delta Spec

## MODIFIED Requirements

### Requirement: Marketing sin clases duras residuales

Las páginas de marketing y sus componentes SHALL quedar libres de clases duras residuales fuera del lenguaje (emerald-500 sueltos en botones/acentos, sombras emerald, superficies `slate-900` decorativas no aprobadas), migrando a tokens/lenguaje. Las rutas, metadata, sitemap, structured data y copy SHALL permanecer sin cambios.

#### Scenario: Barrido final

- **WHEN** se ejecuta el grep de verificación de marketing
- **THEN** no SHALL quedar coincidencias de clases duras residuales fuera del lenguaje aprobado
- **AND** todas las rutas públicas SHALL renderizar sin cambios
