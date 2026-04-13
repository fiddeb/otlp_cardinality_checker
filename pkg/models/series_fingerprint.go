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

// CreateSeriesFingerprintWithResource creates a fingerprint from both resource
// and data-point attributes. Keys are prefixed with "R:" or "D:" to prevent
// namespace collisions (the same key may appear at both levels in OTLP).
func CreateSeriesFingerprintWithResource(resourceAttrs, dpAttrs map[string]string) string {
	if len(resourceAttrs) == 0 && len(dpAttrs) == 0 {
		return "constant"
	}

	keys := make([]string, 0, len(resourceAttrs)+len(dpAttrs))
	combined := make(map[string]string, len(resourceAttrs)+len(dpAttrs))

	for k, v := range resourceAttrs {
		prefixed := "R:" + k
		keys = append(keys, prefixed)
		combined[prefixed] = v
	}
	for k, v := range dpAttrs {
		prefixed := "D:" + k
		keys = append(keys, prefixed)
		combined[prefixed] = v
	}

	sort.Strings(keys)

	var builder strings.Builder
	for i, key := range keys {
		if i > 0 {
			builder.WriteString(",")
		}
		builder.WriteString(key)
		builder.WriteString("=")
		builder.WriteString(combined[key])
	}

	return builder.String()
}
