-- Add metric_data column to store type-specific metric information
-- This stores JSON with histogram bounds, exponential histogram scales, etc.

ALTER TABLE metrics ADD COLUMN metric_data TEXT;

-- Record this migration
INSERT OR IGNORE INTO schema_migrations (version) VALUES (6);
