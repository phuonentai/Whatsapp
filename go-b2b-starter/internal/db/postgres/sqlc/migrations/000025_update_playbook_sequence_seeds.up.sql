-- 000025_update_playbook_sequence_seeds.up.sql
-- Converts one guion per vertical into a scripted sequence (pasos): an ordered
-- list of messages that auto-advance in the inbox composer as the human sends
-- each step. Forward-only: payload JSONB updated in place; no schema change.

-- comercio: confirmar-pedido → detalle del pedido → dirección → link de pago
UPDATE modules.playbooks
SET payload = jsonb_set(payload, '{guiones,1}', '{
  "id": "confirmar-pedido",
  "titulo": "Confirmar pedido",
  "pasos": [
    {"id": "detalle", "titulo": "Detalle del pedido", "mensaje": "¡Perfecto! ¿Qué producto(s) quieres y en qué cantidad?"},
    {"id": "direccion", "titulo": "Dirección", "mensaje": "¿A qué dirección lo enviamos? ¿Algún punto de referencia?"},
    {"id": "link-pago", "titulo": "Link de pago", "mensaje": "Te enviamos el link de pago: puedes pagar con PSE, Nequi o tarjeta. Cuando esté confirmado, lo despachamos."}
  ]
}'::jsonb)
WHERE key = 'comercio';

-- restaurantes: domicilio → confirmación del pedido → en camino → follow-up
UPDATE modules.playbooks
SET payload = jsonb_set(payload, '{guiones,3}', '{
  "id": "domicilio",
  "titulo": "Domicilio",
  "pasos": [
    {"id": "confirmar", "titulo": "Confirmar pedido", "mensaje": "¡Perfecto! Tu pedido está confirmado y en preparación."},
    {"id": "en-camino", "titulo": "En camino", "mensaje": "Tu pedido va en camino. ¡Que lo disfrutes!"},
    {"id": "follow-up", "titulo": "Seguimiento", "mensaje": "¡Esperamos que todo esté delicioso! ¿Quieres pedir algo más o dejamos algo para tu próxima visita?"}
  ]
}'::jsonb)
WHERE key = 'restaurantes';

-- citas: confirmar-cita → fecha → hora → confirmación + recordatorio
UPDATE modules.playbooks
SET payload = jsonb_set(payload, '{guiones,1}', '{
  "id": "confirmar-cita",
  "titulo": "Confirmar cita",
  "pasos": [
    {"id": "fecha", "titulo": "Fecha", "mensaje": "¡Claro! ¿Qué día te queda bien para agendar?"},
    {"id": "hora", "titulo": "Hora", "mensaje": "Perfecto. ¿A qué hora te queda mejor?"},
    {"id": "confirmacion", "titulo": "Confirmación", "mensaje": "¡Tu cita está confirmada! Te enviamos un recordatorio antes de la cita. Si necesitas reprogramar, escríbenos."}
  ]
}'::jsonb)
WHERE key = 'citas';

-- servicios-profesionales: cotizacion → necesidad → alcance → propuesta
UPDATE modules.playbooks
SET payload = jsonb_set(payload, '{guiones,1}', '{
  "id": "cotizacion",
  "titulo": "Cotización",
  "pasos": [
    {"id": "necesidad", "titulo": "Necesidad", "mensaje": "Claro, con gusto te cotizamos. Cuéntanos brevemente qué necesitas."},
    {"id": "alcance", "titulo": "Alcance", "mensaje": "Perfecto. ¿Hay algún plazo o alcance específico que debamos tener en cuenta?"},
    {"id": "propuesta", "titulo": "Propuesta", "mensaje": "Te preparamos la cotización y te la enviamos hoy. ¿Quieres incluir algo más?"}
  ]
}'::jsonb)
WHERE key = 'servicios-profesionales';

-- talleres: cotizacion → síntoma → cotización → aprobación
UPDATE modules.playbooks
SET payload = jsonb_set(payload, '{guiones,1}', '{
  "id": "cotizacion",
  "titulo": "Cotización",
  "pasos": [
    {"id": "sintoma", "titulo": "Síntoma", "mensaje": "¡Hola! Cuéntanos qué falla presenta el vehículo o equipo para hacer el diagnóstico."},
    {"id": "cotizacion", "titulo": "Cotización", "mensaje": "Te enviamos la cotización con el detalle del trabajo."},
    {"id": "aprobacion", "titulo": "Aprobación", "mensaje": "Confírmala para iniciar el trabajo. El trabajo tiene garantía."}
  ]
}'::jsonb)
WHERE key = 'talleres';
