-- 000025_update_playbook_sequence_seeds.down.sql
-- Restores the five sequence guiones to their original single-shot form.

-- comercio: confirmar-pedido
UPDATE modules.playbooks
SET payload = jsonb_set(payload, '{guiones,1}', '{"id": "confirmar-pedido", "titulo": "Confirmar pedido", "mensaje": "¡Claro! Confirmamos tu pedido. ¿Quieres que te enviemos el link de pago?"}'::jsonb)
WHERE key = 'comercio';

-- restaurantes: domicilio
UPDATE modules.playbooks
SET payload = jsonb_set(payload, '{guiones,3}', '{"id": "domicilio", "titulo": "Domicilio en camino", "mensaje": "Tu pedido va en camino. ¡Que lo disfrutes!"}'::jsonb)
WHERE key = 'restaurantes';

-- citas: confirmar-cita
UPDATE modules.playbooks
SET payload = jsonb_set(payload, '{guiones,1}', '{"id": "confirmar-cita", "titulo": "Confirmar cita", "mensaje": "¡Tu cita está confirmada! Te enviamos un recordatorio antes de la cita."}'::jsonb)
WHERE key = 'citas';

-- servicios-profesionales: cotizacion
UPDATE modules.playbooks
SET payload = jsonb_set(payload, '{guiones,1}', '{"id": "cotizacion", "titulo": "Cotización", "mensaje": "Claro, te preparamos la cotización. Te la enviamos hoy. ¿Quieres incluir algo más?"}'::jsonb)
WHERE key = 'servicios-profesionales';

-- talleres: cotizacion
UPDATE modules.playbooks
SET payload = jsonb_set(payload, '{guiones,1}', '{"id": "cotizacion", "titulo": "Cotización", "mensaje": "Te enviamos la cotización. Confírmala para iniciar el trabajo."}'::jsonb)
WHERE key = 'talleres';
