package analyzer

import (
	"context"
)

// AttributeCatalog defines the interface for storing attribute values.
// This allows analyzers to feed the global attribute catalog without
// tight coupling to the storage implementation.
type AttributeCatalog interface {
	StoreAttributeValue(ctx context.Context, key, value, signalType, scope string) error
}

// getServiceName extracts service.name from resource attributes.
// When labels are provided (pod log enrichment mode), the function also
// checks the ordered label list and falls back to "unknown_service".
// With no labels the behaviour is unchanged: falls back to "unknown".
func getServiceName(attrs map[string]string, labels ...string) string {
	// Always try service.name first.
	if name, ok := attrs["service.name"]; ok && name != "" {
		return name
	}

	// Enrichment mode: iterate the priority label list.
	for _, label := range labels {
		if v, ok := attrs[label]; ok && v != "" {
			return v
		}
	}

	if len(labels) > 0 {
		return "unknown_service"
	}
	return "unknown"
}

// extractAttributesToCatalog extracts all attributes and stores them in the catalog.
// This feeds the global attribute catalog with key-value pairs from telemetry data.
func extractAttributesToCatalog(ctx context.Context, catalog AttributeCatalog, attrs map[string]string, signalType, scope string) {
	if catalog == nil {
		return
	}

	for key, value := range attrs {
		// Best-effort: ignore errors to not block telemetry processing
		_ = catalog.StoreAttributeValue(ctx, key, value, signalType, scope)
	}
}

// batchCatalog deduplicates catalog writes within a single request batch.
// Each unique (signalType, scope, key, value) tuple is forwarded to the real
// catalog exactly once, regardless of how many log/metric/span records in the
// batch share the same attribute. This collapses the O(records × attrs) write
// lock acquisitions on storage.attributesmu down to O(unique tuples), which
// is the dominant contention source under load (50 VUs × ~9k records/req).
type batchCatalog struct {
	seen    map[string]struct{} // compound key: signalType\0scope\0attrKey\0value
	catalog AttributeCatalog
}

func newBatchCatalog(catalog AttributeCatalog) *batchCatalog {
	return &batchCatalog{
		seen:    make(map[string]struct{}),
		catalog: catalog,
	}
}

func (b *batchCatalog) StoreAttributeValue(ctx context.Context, key, value, signalType, scope string) error {
	if b.catalog == nil {
		return nil
	}
	// Use a separator that cannot appear in normal attribute keys/values.
	k := signalType + "\x00" + scope + "\x00" + key + "\x00" + value
	if _, ok := b.seen[k]; ok {
		return nil
	}
	b.seen[k] = struct{}{}
	return b.catalog.StoreAttributeValue(ctx, key, value, signalType, scope)
}
