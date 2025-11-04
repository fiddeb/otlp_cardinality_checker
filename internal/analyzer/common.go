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
// Returns "unknown" if service.name is not found or host.name as fallback.
func getServiceName(attrs map[string]string) string {
	// First try service.name
	if name, ok := attrs["service.name"]; ok && name != "" {
		return name
	}
	
	// Fallback to host.name
	if name, ok := attrs["host.name"]; ok && name != "" {
		return name
	}
	
	// Default to unknown
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
