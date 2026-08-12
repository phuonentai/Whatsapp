-- 000038_create_procurement_schedules.down.sql
-- Rollback: drop the four scheduling tables (join tables cascade with their
-- parent; explicit order keeps dependency intent clear). The procurement
-- schema and sibling tables are untouched; already-created scheduled runs
-- remain ordinary inquiry runs (source='scheduled' is inert without the
-- scheduler).

DROP TABLE IF EXISTS procurement.schedule_followups;
DROP TABLE IF EXISTS procurement.schedule_suppliers;
DROP TABLE IF EXISTS procurement.schedule_products;
DROP TABLE IF EXISTS procurement.schedules;
