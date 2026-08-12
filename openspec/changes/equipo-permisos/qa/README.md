# QA — Equipo y permisos (`view=access`)

Capturas Playwright (chromium) del 2026-08-12 contra el stack e2e mock-auth
(`test-org-rbac` / `admin-rbac@test.com`, backend :8080 con `AUTH_MOCK_ENABLED=true`).

## Capturas (3 tabs × 3 viewports)

- `access-members-390x844.png` / `768x1024` / `1440x900` — tab Miembros
- `access-matrix-390x844.png` / `768x1024` / `1440x900` — tab Matriz de permisos
- `access-modules-390x844.png` / `768x1024` / `1440x900` — tab Módulos

## Verificaciones a11y

- Matriz: tabla real con `th scope="col"` (4 columnas: Recurso + 3 roles) ✓
- Tooltips de celda accesibles por teclado: Tab hasta una celda → tooltip con
  `rol + resource:action` (probado: "Member · activity" → "Member: Sin permiso") ✓
- Preview de impacto: disclosure con `aria-expanded` (false → true al abrir) ✓
- Celdas ✓ / parcial / — con texto (nunca color-only) ✓
- Contraste: paleta Tailwind estándar (emerald/amber/gray 50/700); no se ejecutó
  escaneo automatizado de contraste (se usa la convención de diseño existente).

## Errores de consola

Ninguno originado en los componentes nuevos. Errores preexistentes del entorno
mock: bootstrap del SDK Stytch bloqueado por CSP (`test.stytch.com`) y 404 del
query de configuración de WhatsApp (org de prueba sin conexión; la UI muestra
"No connected").
