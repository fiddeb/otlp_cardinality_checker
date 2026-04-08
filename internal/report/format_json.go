package report

import "encoding/json"

// FormatJSON formats a report as indented JSON.
func FormatJSON(r *Report) ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}
