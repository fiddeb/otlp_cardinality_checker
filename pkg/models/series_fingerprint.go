package models

import (
	"sort"
	"strings"
)

// CreateSeriesFingerprintFast creates a fingerprint without hashing for better performance.
// Suitable for HyperLogLog which internally hashes the input anyway.
func CreateSeriesFingerprintFast(labels map[string]string) string {
	if len(labels) == 0 {
		return "constant"
	}

	// Sort keys alphabetically
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build fingerprint string without additional hashing
	var builder strings.Builder
	for i, key := range keys {
		if i > 0 {
			builder.WriteString(",")
		}
		builder.WriteString(key)
		builder.WriteString("=")
		builder.WriteString(labels[key])
	}

	return builder.String()
}
