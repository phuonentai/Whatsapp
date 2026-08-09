package playbooks

import (
	"encoding/json"

	"github.com/moasq/go-b2b-starter/internal/modules/playbooks/domain"
)

// catalog is the single source of truth for playbook payloads. The migration
// seed (000020_create_playbooks.up.sql) mirrors this data; CatalogValidated
// checks keep the two in sync at startup and in tests.
var catalog = []domain.Playbook{
	{
		Key:             "comercio",
		Name:            "Comercio & E-commerce",
		Vertical:        "retail",
		Description:     "Venta por WhatsApp: pedidos, confirmación de pago, entrega/domicilio, reactivación de clientes y devoluciones.",
		RequiresModules: []string{},
		Payload: payloadJSON(map[string]any{
			"pipeline": map[string]any{
				"nombre": "Ventas por WhatsApp",
				"etapas": []map[string]any{
					{"nombre": "Nuevo Pedido", "orden": 1, "color": "#6B7280", "probabilidad": 10},
					{"nombre": "Confirmado", "orden": 2, "color": "#3B82F6", "probabilidad": 30},
					{"nombre": "Pagado", "orden": 3, "color": "#8B5CF6", "probabilidad": 60},
					{"nombre": "En Entrega", "orden": 4, "color": "#F59E0B", "probabilidad": 85},
					{"nombre": "Entregado", "orden": 5, "color": "#10B981", "probabilidad": 100},
					{"nombre": "Cancelado", "orden": 6, "color": "#EF4444", "probabilidad": 0},
				},
			},
			"tags": []string{"nuevo", "frecuente", "mayorista", "pendiente-pago", "devolucion"},
			"module_configs": map[string]any{
				"tickets": map[string]any{
					"sla_hours":  map[string]any{"low": 48, "normal": 24, "high": 8},
					"priorities": []string{"low", "normal", "high"},
					"tags":       []string{"devolucion", "garantia", "reclamo"},
				},
			},
			"guiones": []map[string]any{
				{"id": "saludo", "titulo": "Saludo", "mensaje": "¡Hola! Gracias por escribirnos. ¿En qué podemos ayudarte hoy?"},
				{"id": "confirmar-pedido", "titulo": "Confirmar pedido", "mensaje": "¡Claro! Confirmamos tu pedido. ¿Quieres que te enviemos el link de pago?"},
				{"id": "link-pago", "titulo": "Link de pago", "mensaje": "Aquí está tu link de pago: puedes pagar con PSE, Nequi o tarjeta. Cuando esté confirmado, lo despachamos."},
				{"id": "entrega", "titulo": "En entrega", "mensaje": "¡Tu pedido va en camino! Te avisamos cuando esté listo para entregar."},
				{"id": "seguimiento", "titulo": "Seguimiento post-venta", "mensaje": "¡Hola! ¿Cómo te fue con tu compra? Si tienes alguna duda, aquí estamos."},
				{"id": "devolucion", "titulo": "Devolución", "mensaje": "Lamentamos el inconveniente. Cuéntanos qué pasó con el pedido para ayudarte con el cambio o la devolución."},
			},
		}),
	},
	{
		Key:             "restaurantes",
		Name:            "Restaurantes & Alimentos",
		Vertical:        "alimentos",
		Description:     "Pedidos y domicilios por WhatsApp: menú, estados del pedido, reservas, promociones y atención de reclamos.",
		RequiresModules: []string{},
		Payload: payloadJSON(map[string]any{
			"pipeline": map[string]any{
				"nombre": "Pedidos del Día",
				"etapas": []map[string]any{
					{"nombre": "Nuevo", "orden": 1, "color": "#6B7280", "probabilidad": 10},
					{"nombre": "En Preparación", "orden": 2, "color": "#3B82F6", "probabilidad": 40},
					{"nombre": "En Camino", "orden": 3, "color": "#F59E0B", "probabilidad": 80},
					{"nombre": "Entregado", "orden": 4, "color": "#10B981", "probabilidad": 100},
					{"nombre": "Cancelado", "orden": 5, "color": "#EF4444", "probabilidad": 0},
				},
			},
			"tags": []string{"domicilio", "mesa", "frecuente", "queja", "reserva"},
			"module_configs": map[string]any{
				"tickets": map[string]any{
					"sla_hours":  map[string]any{"low": 24, "normal": 8, "high": 2},
					"priorities": []string{"low", "normal", "high"},
					"tags":       []string{"queja", "reclamo"},
				},
			},
			"guiones": []map[string]any{
				{"id": "bienvenida", "titulo": "Bienvenida", "mensaje": "¡Bienvenido! ¿Quieres ver el menú del día o hacer una reserva?"},
				{"id": "menu", "titulo": "Enviar menú", "mensaje": "Claro, te compartimos el menú. ¿Qué te gustaría pedir?"},
				{"id": "confirmar-pedido", "titulo": "Confirmar pedido", "mensaje": "¡Perfecto! Tu pedido está confirmado. Te avisamos cuando esté en camino."},
				{"id": "domicilio", "titulo": "Domicilio en camino", "mensaje": "Tu pedido va en camino. ¡Que lo disfrutes!"},
				{"id": "reserva", "titulo": "Confirmar reserva", "mensaje": "¡Reserva confirmada! Te esperamos. Si necesitas cambiar la hora, escríbenos."},
				{"id": "queja", "titulo": "Atención de queja", "mensaje": "Lamentamos mucho lo sucedido. Vamos a revisarlo de inmediato y te damos una solución."},
			},
		}),
	},
	{
		Key:             "citas",
		Name:            "Citas: Salud, Estética & Bienestar",
		Vertical:        "servicios-cita",
		Description:     "Agendamiento por WhatsApp: solicitud y confirmación de citas, recordatorios anti-no-show, venta de bonos y seguimiento post-servicio.",
		RequiresModules: []string{},
		Payload: payloadJSON(map[string]any{
			"pipeline": map[string]any{
				"nombre": "Agenda de Citas",
				"etapas": []map[string]any{
					{"nombre": "Solicitada", "orden": 1, "color": "#6B7280", "probabilidad": 10},
					{"nombre": "Confirmada", "orden": 2, "color": "#3B82F6", "probabilidad": 50},
					{"nombre": "Realizada", "orden": 3, "color": "#10B981", "probabilidad": 100},
					{"nombre": "No Asistió", "orden": 4, "color": "#F59E0B", "probabilidad": 0},
					{"nombre": "Cancelada", "orden": 5, "color": "#EF4444", "probabilidad": 0},
				},
			},
			"tags": []string{"nueva", "recurrente", "bono", "no-show", "paquete"},
			"module_configs": map[string]any{
				"tickets": map[string]any{
					"sla_hours":  map[string]any{"low": 72, "normal": 48, "high": 24},
					"priorities": []string{"low", "normal", "high"},
					"tags":       []string{"reclamo", "reagendamiento"},
				},
			},
			"guiones": []map[string]any{
				{"id": "horarios", "titulo": "Horarios disponibles", "mensaje": "¡Hola! Claro, te contamos los horarios disponibles. ¿Cuál te queda mejor?"},
				{"id": "confirmar-cita", "titulo": "Confirmar cita", "mensaje": "¡Tu cita está confirmada! Te enviamos un recordatorio antes de la cita."},
				{"id": "recordatorio", "titulo": "Recordatorio", "mensaje": "¡Hola! Te recordamos tu cita de mañana. Si necesitas reprogramar, escríbenos."},
				{"id": "post-servicio", "titulo": "Seguimiento post-servicio", "mensaje": "¡Esperamos que hayas tenido una excelente experiencia! ¿Te gustaría agendar tu próxima visita?"},
				{"id": "bono", "titulo": "Bonos y paquetes", "mensaje": "Tenemos bonos y paquetes con descuento. ¿Quieres más información?"},
			},
		}),
	},
	{
		Key:             "servicios-profesionales",
		Name:            "Servicios Profesionales",
		Vertical:        "servicios-b2b",
		Description:     "Gestión de clientes B2B: captura de leads, cotización y propuestas, seguimiento, entrega por hitos y soporte por contrato.",
		RequiresModules: []string{},
		Payload: payloadJSON(map[string]any{
			"pipeline": map[string]any{
				"nombre": "Gestión de Clientes",
				"etapas": []map[string]any{
					{"nombre": "Prospecto", "orden": 1, "color": "#6B7280", "probabilidad": 10},
					{"nombre": "Cotización", "orden": 2, "color": "#3B82F6", "probabilidad": 30},
					{"nombre": "Propuesta Enviada", "orden": 3, "color": "#8B5CF6", "probabilidad": 50},
					{"nombre": "Negociación", "orden": 4, "color": "#F59E0B", "probabilidad": 75},
					{"nombre": "Cliente Activo", "orden": 5, "color": "#10B981", "probabilidad": 100},
					{"nombre": "Cerrado Perdido", "orden": 6, "color": "#EF4444", "probabilidad": 0},
				},
			},
			"tags": []string{"lead", "cliente", "referido", "cobranza", "soporte"},
			"module_configs": map[string]any{
				"tickets": map[string]any{
					"sla_hours":  map[string]any{"low": 72, "normal": 48, "high": 24},
					"priorities": []string{"low", "normal", "high"},
					"tags":       []string{"soporte", "urgencia"},
				},
			},
			"guiones": []map[string]any{
				{"id": "captura", "titulo": "Captura de lead", "mensaje": "¡Gracias por escribirnos! Cuéntanos brevemente qué necesitas y un asesor te atiende."},
				{"id": "cotizacion", "titulo": "Cotización", "mensaje": "Claro, te preparamos la cotización. Te la enviamos hoy. ¿Quieres incluir algo más?"},
				{"id": "propuesta", "titulo": "Propuesta", "mensaje": "Aquí está la propuesta completa. ¿Tienes alguna duda o ajuste que quieras hacer?"},
				{"id": "seguimiento", "titulo": "Seguimiento", "mensaje": "¡Hola! Te escribimos para saber si tuviste oportunidad de revisar la propuesta."},
				{"id": "cliente-activo", "titulo": "Bienvenida de cliente", "mensaje": "¡Bienvenido! A partir de ahora este será nuestro canal de comunicación."},
			},
		}),
	},
	{
		Key:             "talleres",
		Name:            "Talleres & Reparación",
		Vertical:        "talleres",
		Description:     "Órdenes de trabajo por WhatsApp: recepción, cotización con aprobación, estados de reparación, aviso de entrega y garantía.",
		RequiresModules: []string{},
		Payload: payloadJSON(map[string]any{
			"pipeline": map[string]any{
				"nombre": "Órdenes de Trabajo",
				"etapas": []map[string]any{
					{"nombre": "Recibido", "orden": 1, "color": "#6B7280", "probabilidad": 10},
					{"nombre": "Cotizado", "orden": 2, "color": "#3B82F6", "probabilidad": 30},
					{"nombre": "En Reparación", "orden": 3, "color": "#8B5CF6", "probabilidad": 60},
					{"nombre": "Listo", "orden": 4, "color": "#F59E0B", "probabilidad": 90},
					{"nombre": "Entregado", "orden": 5, "color": "#10B981", "probabilidad": 100},
					{"nombre": "Cancelado", "orden": 6, "color": "#EF4444", "probabilidad": 0},
				},
			},
			"tags": []string{"diagnostico", "cotizacion", "garantia", "urgencia", "seguimiento"},
			"module_configs": map[string]any{
				"tickets": map[string]any{
					"sla_hours":  map[string]any{"low": 72, "normal": 48, "high": 8},
					"priorities": []string{"low", "normal", "high"},
					"tags":       []string{"garantia", "reclamo"},
				},
			},
			"guiones": []map[string]any{
				{"id": "recepcion", "titulo": "Recepción", "mensaje": "¡Hola! Recibimos tu vehículo o equipo. Te avisamos con el diagnóstico."},
				{"id": "cotizacion", "titulo": "Cotización", "mensaje": "Te enviamos la cotización. Confírmala para iniciar el trabajo."},
				{"id": "aprobado", "titulo": "Trabajo aprobado", "mensaje": "¡Gracias por confirmar! Estamos trabajando en ello."},
				{"id": "listo", "titulo": "Aviso de entrega", "mensaje": "¡Tu vehículo o equipo está listo! ¿Cuándo quieres recogerlo?"},
				{"id": "garantia", "titulo": "Garantía", "mensaje": "Tranquilo, el trabajo tiene garantía. Agendamos una revisión."},
			},
		}),
	},
}

// Catalog returns the five vertical playbooks as Go data (migration seed mirror).
func Catalog() []domain.Playbook {
	return catalog
}

func payloadJSON(v map[string]any) []byte {
	b, _ := json.Marshal(v)
	return b
}
