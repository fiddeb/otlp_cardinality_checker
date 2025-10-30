-- Add OTLP-standard fields to spans table

-- Add kind as INTEGER (OTLP SpanKind enum: 0=UNSPECIFIED, 1=INTERNAL, 2=SERVER, 3=CLIENT, 4=PRODUCER, 5=CONSUMER)
ALTER TABLE spans ADD COLUMN kind_number INTEGER;

-- Add kind_name for human-readable span kind
ALTER TABLE spans ADD COLUMN kind_name TEXT;

-- Add boolean flags for trace_state and parent_span_id presence
ALTER TABLE spans ADD COLUMN has_trace_state INTEGER DEFAULT 0;
ALTER TABLE spans ADD COLUMN has_parent_span_id INTEGER DEFAULT 0;

-- Add status_codes as JSON array of observed status codes
ALTER TABLE spans ADD COLUMN status_codes TEXT;

-- Add dropped attributes statistics
ALTER TABLE spans ADD COLUMN dropped_attrs_total INTEGER DEFAULT 0;
ALTER TABLE spans ADD COLUMN dropped_attrs_items INTEGER DEFAULT 0;
ALTER TABLE spans ADD COLUMN dropped_attrs_max INTEGER DEFAULT 0;

-- Add dropped events statistics
ALTER TABLE spans ADD COLUMN dropped_events_total INTEGER DEFAULT 0;
ALTER TABLE spans ADD COLUMN dropped_events_items INTEGER DEFAULT 0;
ALTER TABLE spans ADD COLUMN dropped_events_max INTEGER DEFAULT 0;

-- Add dropped links statistics
ALTER TABLE spans ADD COLUMN dropped_links_total INTEGER DEFAULT 0;
ALTER TABLE spans ADD COLUMN dropped_links_items INTEGER DEFAULT 0;
ALTER TABLE spans ADD COLUMN dropped_links_max INTEGER DEFAULT 0;
