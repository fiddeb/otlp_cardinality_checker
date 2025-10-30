-- Add OTLP-standard fields to logs table

ALTER TABLE logs ADD COLUMN severity_number INTEGER;
ALTER TABLE logs ADD COLUMN has_trace_context INTEGER DEFAULT 0;
ALTER TABLE logs ADD COLUMN has_span_context INTEGER DEFAULT 0;
ALTER TABLE logs ADD COLUMN event_names TEXT; -- JSON array
ALTER TABLE logs ADD COLUMN dropped_attrs_total INTEGER DEFAULT 0;
ALTER TABLE logs ADD COLUMN dropped_attrs_records INTEGER DEFAULT 0;
ALTER TABLE logs ADD COLUMN dropped_attrs_max INTEGER DEFAULT 0;
