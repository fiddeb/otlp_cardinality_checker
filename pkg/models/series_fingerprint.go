package models

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
)

// CreateSeriesFingerprint creates a unique fingerprint for a set of label key-value pairs.
// Labels are sorted alphabetically by key to ensure consistent hashing regardless of order.
//
// Example:
//   labels := map[string]string{
//     "method": "GET",
//     "status": "200",
//     "service": "api",
//   }
//   fingerprint := CreateSeriesFingerprint(labels)
//   // Returns hash of: "method=GET,service=api,status=200"
func CreateSeriesFingerprint(labels map[string]string) string {
	if len(labels) == 0 {
		return "constant" // Constant metric with no labels
	}

	// Sort keys alphabetically for consistent ordering
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build fingerprint string: key1=value1,key2=value2,...
	var builder strings.Builder
	for i, key := range keys {
		if i > 0 {
			builder.WriteString(",")
		}
		builder.WriteString(key)
		builder.WriteString("=")
		builder.WriteString(labels[key])
	}

	// Hash the fingerprint for fixed-size output
	hash := sha256.Sum256([]byte(builder.String()))
	return hex.EncodeToString(hash[:])
}

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
