-- Rollback initial schema

DROP INDEX IF EXISTS idx_log_templates_count;
DROP INDEX IF EXISTS idx_log_templates_severity_service;
DROP TABLE IF EXISTS log_body_templates;
DROP INDEX IF EXISTS idx_log_keys_severity;
DROP TABLE IF EXISTS log_keys;
DROP INDEX IF EXISTS idx_log_services_service;
DROP TABLE IF EXISTS log_services;
DROP TABLE IF EXISTS logs;

DROP TABLE IF EXISTS span_events;
DROP INDEX IF EXISTS idx_span_keys_name;
DROP TABLE IF EXISTS span_keys;
DROP INDEX IF EXISTS idx_span_services_service;
DROP TABLE IF EXISTS span_services;
DROP TABLE IF EXISTS spans;

DROP INDEX IF EXISTS idx_metric_keys_name;
DROP TABLE IF EXISTS metric_keys;
DROP INDEX IF EXISTS idx_metric_services_service;
DROP TABLE IF EXISTS metric_services;
DROP TABLE IF EXISTS metrics;
