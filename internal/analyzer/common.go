package analyzer

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
