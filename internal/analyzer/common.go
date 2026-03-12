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
