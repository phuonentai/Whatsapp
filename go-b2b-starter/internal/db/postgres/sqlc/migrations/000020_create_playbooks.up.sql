-- 000019_create_playbooks.up.sql
-- Vertical playbooks: out-of-the-box business procedures (pipeline, tags,
-- module config presets, WhatsApp guiones) per vertical, seeded in one click.

CREATE SCHEMA IF NOT EXISTS modules;

-- ============================================================
-- Playbook registry (catalog of vertical business playbooks)
-- ============================================================

CREATE TABLE modules.playbooks (
    id SERIAL PRIMARY KEY,
    key VARCHAR(100) NOT NULL UNIQUE,
    name VARCHAR(200) NOT NULL,
    vertical VARCHAR(100) NOT NULL,
    description TEXT,
    requires_modules JSONB NOT NULL DEFAULT '[]',
    payload JSONB NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE modules.playbooks IS 'Catálogo de playbooks verticales (procedimientos listos para usar)';
COMMENT ON COLUMN modules.playbooks.payload IS 'Payload semilla: {pipeline:{nombre,etapas[]}, tags[], module_configs{}, guiones[{id,titulo,mensaje}]}';
COMMENT ON COLUMN modules.playbooks.requires_modules IS 'Keys de módulos que deben estar habilitados (vacío = sin requisitos)';

-- ============================================================
-- Per-org playbook state (what was applied and when)
-- ============================================================

CREATE TABLE modules.organization_playbooks (
    id SERIAL PRIMARY KEY,
    organization_id INTEGER NOT NULL REFERENCES organizations.organizations(id) ON DELETE CASCADE,
    playbook_key VARCHAR(100) NOT NULL REFERENCES modules.playbooks(key),
    seeded_pipeline_id INTEGER REFERENCES crm.pipelines(id) ON DELETE SET NULL,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(organization_id, playbook_key)
);

CREATE INDEX idx_org_playbooks_org ON modules.organization_playbooks(organization_id);

COMMENT ON TABLE modules.organization_playbooks IS 'Playbooks aplicados por organización';

-- ============================================================
-- Seed: the five Colombian vertical playbooks
-- ============================================================

INSERT INTO modules.playbooks (key, name, vertical, description, requires_modules, payload)
VALUES (
    'comercio',
    'Comercio & E-commerce',
    'retail',
    'Venta por WhatsApp: pedidos, confirmación de pago, entrega/domicilio, reactivación de clientes y devoluciones.',
    '[]',
    '{
        "pipeline": {
            "nombre": "Ventas por WhatsApp",
            "etapas": [
                {"nombre": "Nuevo Pedido", "orden": 1, "color": "#6B7280", "probabilidad": 10},
                {"nombre": "Confirmado", "orden": 2, "color": "#3B82F6", "probabilidad": 30},
                {"nombre": "Pagado", "orden": 3, "color": "#8B5CF6", "probabilidad": 60},
                {"nombre": "En Entrega", "orden": 4, "color": "#F59E0B", "probabilidad": 85},
                {"nombre": "Entregado", "orden": 5, "color": "#10B981", "probabilidad": 100},
                {"nombre": "Cancelado", "orden": 6, "color": "#EF4444", "probabilidad": 0}
            ]
        },
        "tags": ["nuevo", "frecuente", "mayorista", "pendiente-pago", "devolucion"],
        "module_configs": {
            "tickets": {
                "sla_hours": {"low": 48, "normal": 24, "high": 8},
                "priorities": ["low", "normal", "high"],
                "tags": ["devolucion", "garantia", "reclamo"]
            }
        },
        "guiones": [
            {"id": "saludo", "titulo": "Saludo", "mensaje": "¡Hola! Gracias por escribirnos. ¿En qué podemos ayudarte hoy?"},
            {"id": "confirmar-pedido", "titulo": "Confirmar pedido", "mensaje": "¡Claro! Confirmamos tu pedido. ¿Quieres que te enviemos el link de pago?"},
            {"id": "link-pago", "titulo": "Link de pago", "mensaje": "Aquí está tu link de pago: puedes pagar con PSE, Nequi o tarjeta. Cuando esté confirmado, lo despachamos."},
            {"id": "entrega", "titulo": "En entrega", "mensaje": "¡Tu pedido va en camino! Te avisamos cuando esté listo para entregar."},
            {"id": "seguimiento", "titulo": "Seguimiento post-venta", "mensaje": "¡Hola! ¿Cómo te fue con tu compra? Si tienes alguna duda, aquí estamos."},
            {"id": "devolucion", "titulo": "Devolución", "mensaje": "Lamentamos el inconveniente. Cuéntanos qué pasó con el pedido para ayudarte con el cambio o la devolución."}
        ]
    }'
);

INSERT INTO modules.playbooks (key, name, vertical, description, requires_modules, payload)
VALUES (
    'restaurantes',
    'Restaurantes & Alimentos',
    'alimentos',
    'Pedidos y domicilios por WhatsApp: menú, estados del pedido, reservas, promociones y atención de reclamos.',
    '[]',
    '{
        "pipeline": {
            "nombre": "Pedidos del Día",
            "etapas": [
                {"nombre": "Nuevo", "orden": 1, "color": "#6B7280", "probabilidad": 10},
                {"nombre": "En Preparación", "orden": 2, "color": "#3B82F6", "probabilidad": 40},
                {"nombre": "En Camino", "orden": 3, "color": "#F59E0B", "probabilidad": 80},
                {"nombre": "Entregado", "orden": 4, "color": "#10B981", "probabilidad": 100},
                {"nombre": "Cancelado", "orden": 5, "color": "#EF4444", "probabilidad": 0}
            ]
        },
        "tags": ["domicilio", "mesa", "frecuente", "queja", "reserva"],
        "module_configs": {
            "tickets": {
                "sla_hours": {"low": 24, "normal": 8, "high": 2},
                "priorities": ["low", "normal", "high"],
                "tags": ["queja", "reclamo"]
            }
        },
        "guiones": [
            {"id": "bienvenida", "titulo": "Bienvenida", "mensaje": "¡Bienvenido! ¿Quieres ver el menú del día o hacer una reserva?"},
            {"id": "menu", "titulo": "Enviar menú", "mensaje": "Claro, te compartimos el menú. ¿Qué te gustaría pedir?"},
            {"id": "confirmar-pedido", "titulo": "Confirmar pedido", "mensaje": "¡Perfecto! Tu pedido está confirmado. Te avisamos cuando esté en camino."},
            {"id": "domicilio", "titulo": "Domicilio en camino", "mensaje": "Tu pedido va en camino. ¡Que lo disfrutes!"},
            {"id": "reserva", "titulo": "Confirmar reserva", "mensaje": "¡Reserva confirmada! Te esperamos. Si necesitas cambiar la hora, escríbenos."},
            {"id": "queja", "titulo": "Atención de queja", "mensaje": "Lamentamos mucho lo sucedido. Vamos a revisarlo de inmediato y te damos una solución."}
        ]
    }'
);

INSERT INTO modules.playbooks (key, name, vertical, description, requires_modules, payload)
VALUES (
    'citas',
    'Citas: Salud, Estética & Bienestar',
    'servicios-cita',
    'Agendamiento por WhatsApp: solicitud y confirmación de citas, recordatorios anti-no-show, venta de bonos y seguimiento post-servicio.',
    '[]',
    '{
        "pipeline": {
            "nombre": "Agenda de Citas",
            "etapas": [
                {"nombre": "Solicitada", "orden": 1, "color": "#6B7280", "probabilidad": 10},
                {"nombre": "Confirmada", "orden": 2, "color": "#3B82F6", "probabilidad": 50},
                {"nombre": "Realizada", "orden": 3, "color": "#10B981", "probabilidad": 100},
                {"nombre": "No Asistió", "orden": 4, "color": "#F59E0B", "probabilidad": 0},
                {"nombre": "Cancelada", "orden": 5, "color": "#EF4444", "probabilidad": 0}
            ]
        },
        "tags": ["nueva", "recurrente", "bono", "no-show", "paquete"],
        "module_configs": {
            "tickets": {
                "sla_hours": {"low": 72, "normal": 48, "high": 24},
                "priorities": ["low", "normal", "high"],
                "tags": ["reclamo", "reagendamiento"]
            }
        },
        "guiones": [
            {"id": "horarios", "titulo": "Horarios disponibles", "mensaje": "¡Hola! Claro, te contamos los horarios disponibles. ¿Cuál te queda mejor?"},
            {"id": "confirmar-cita", "titulo": "Confirmar cita", "mensaje": "¡Tu cita está confirmada! Te enviamos un recordatorio antes de la cita."},
            {"id": "recordatorio", "titulo": "Recordatorio", "mensaje": "¡Hola! Te recordamos tu cita de mañana. Si necesitas reprogramar, escríbenos."},
            {"id": "post-servicio", "titulo": "Seguimiento post-servicio", "mensaje": "¡Esperamos que hayas tenido una excelente experiencia! ¿Te gustaría agendar tu próxima visita?"},
            {"id": "bono", "titulo": "Bonos y paquetes", "mensaje": "Tenemos bonos y paquetes con descuento. ¿Quieres más información?"}
        ]
    }'
);

INSERT INTO modules.playbooks (key, name, vertical, description, requires_modules, payload)
VALUES (
    'servicios-profesionales',
    'Servicios Profesionales',
    'servicios-b2b',
    'Gestión de clientes B2B: captura de leads, cotización y propuestas, seguimiento, entrega por hitos y soporte por contrato.',
    '[]',
    '{
        "pipeline": {
            "nombre": "Gestión de Clientes",
            "etapas": [
                {"nombre": "Prospecto", "orden": 1, "color": "#6B7280", "probabilidad": 10},
                {"nombre": "Cotización", "orden": 2, "color": "#3B82F6", "probabilidad": 30},
                {"nombre": "Propuesta Enviada", "orden": 3, "color": "#8B5CF6", "probabilidad": 50},
                {"nombre": "Negociación", "orden": 4, "color": "#F59E0B", "probabilidad": 75},
                {"nombre": "Cliente Activo", "orden": 5, "color": "#10B981", "probabilidad": 100},
                {"nombre": "Cerrado Perdido", "orden": 6, "color": "#EF4444", "probabilidad": 0}
            ]
        },
        "tags": ["lead", "cliente", "referido", "cobranza", "soporte"],
        "module_configs": {
            "tickets": {
                "sla_hours": {"low": 72, "normal": 48, "high": 24},
                "priorities": ["low", "normal", "high"],
                "tags": ["soporte", "urgencia"]
            }
        },
        "guiones": [
            {"id": "captura", "titulo": "Captura de lead", "mensaje": "¡Gracias por escribirnos! Cuéntanos brevemente qué necesitas y un asesor te atiende."},
            {"id": "cotizacion", "titulo": "Cotización", "mensaje": "Claro, te preparamos la cotización. Te la enviamos hoy. ¿Quieres incluir algo más?"},
            {"id": "propuesta", "titulo": "Propuesta", "mensaje": "Aquí está la propuesta completa. ¿Tienes alguna duda o ajuste que quieras hacer?"},
            {"id": "seguimiento", "titulo": "Seguimiento", "mensaje": "¡Hola! Te escribimos para saber si tuviste oportunidad de revisar la propuesta."},
            {"id": "cliente-activo", "titulo": "Bienvenida de cliente", "mensaje": "¡Bienvenido! A partir de ahora este será nuestro canal de comunicación."}
        ]
    }'
);

INSERT INTO modules.playbooks (key, name, vertical, description, requires_modules, payload)
VALUES (
    'talleres',
    'Talleres & Reparación',
    'talleres',
    'Órdenes de trabajo por WhatsApp: recepción, cotización con aprobación, estados de reparación, aviso de entrega y garantía.',
    '[]',
    '{
        "pipeline": {
            "nombre": "Órdenes de Trabajo",
            "etapas": [
                {"nombre": "Recibido", "orden": 1, "color": "#6B7280", "probabilidad": 10},
                {"nombre": "Cotizado", "orden": 2, "color": "#3B82F6", "probabilidad": 30},
                {"nombre": "En Reparación", "orden": 3, "color": "#8B5CF6", "probabilidad": 60},
                {"nombre": "Listo", "orden": 4, "color": "#F59E0B", "probabilidad": 90},
                {"nombre": "Entregado", "orden": 5, "color": "#10B981", "probabilidad": 100},
                {"nombre": "Cancelado", "orden": 6, "color": "#EF4444", "probabilidad": 0}
            ]
        },
        "tags": ["diagnostico", "cotizacion", "garantia", "urgencia", "seguimiento"],
        "module_configs": {
            "tickets": {
                "sla_hours": {"low": 72, "normal": 48, "high": 8},
                "priorities": ["low", "normal", "high"],
                "tags": ["garantia", "reclamo"]
            }
        },
        "guiones": [
            {"id": "recepcion", "titulo": "Recepción", "mensaje": "¡Hola! Recibimos tu vehículo o equipo. Te avisamos con el diagnóstico."},
            {"id": "cotizacion", "titulo": "Cotización", "mensaje": "Te enviamos la cotización. Confírmala para iniciar el trabajo."},
            {"id": "aprobado", "titulo": "Trabajo aprobado", "mensaje": "¡Gracias por confirmar! Estamos trabajando en ello."},
            {"id": "listo", "titulo": "Aviso de entrega", "mensaje": "¡Tu vehículo o equipo está listo! ¿Cuándo quieres recogerlo?"},
            {"id": "garantia", "titulo": "Garantía", "mensaje": "Tranquilo, el trabajo tiene garantía. Agendamos una revisión."}
        ]
    }'
);
