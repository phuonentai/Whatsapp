-- Seed the analytics module (sales reporting, read-only) in the module registry.

INSERT INTO modules.modules (key, name, description, granted_features, requires, config_schema, is_internal)
VALUES (
    'analytics',
    'Analytics (Reportes de ventas)',
    'Reportes de ventas: ventas facturadas por periodo, top clientes, embudo de negocios y contactos inactivos.',
    '["analytics_module"]',
    '[]',
    '{
        "type": "object",
        "properties": {},
        "additionalProperties": false
    }',
    false
)
ON CONFLICT (key) DO NOTHING;
