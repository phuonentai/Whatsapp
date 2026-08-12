-- 000037_create_procurement_schema.down.sql
-- Procurement capability rollback: drops the whole schema (additive migration).

DROP SCHEMA IF EXISTS procurement CASCADE;
