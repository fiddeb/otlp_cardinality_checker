package autotemplate

import (
	"strings"
	"unicode"
)

// longTokenThreshold is the character length above which a token is considered
// a high-entropy variable (e.g. base64 blobs, proto payloads with embedded
// spaces). Such tokens are collapsed to "<*>" so they don't inflate the token
// count and prevent cluster matching in Drain's length-bucketed tree.
const longTokenThreshold = 30

// tokenize splits a log message into tokens using whitespace and configured
// delimiters, then normalizes the result by:
//   - replacing tokens longer than longTokenThreshold with "<*>"
//   - collapsing consecutive "<*>" wildcards into a single one
func tokenize(message string, extraDelimiters []rune) []string {
	var raw []string

	if len(extraDelimiters) == 0 {
		// Fast path: whitespace only
		raw = strings.Fields(message)
	} else {
		// Build delimiter map
		delims := make(map[rune]bool, len(extraDelimiters))
		for _, r := range extraDelimiters {
			delims[r] = true
		}

		var current strings.Builder
		for _, r := range message {
			if unicode.IsSpace(r) || delims[r] {
				if current.Len() > 0 {
					raw = append(raw, current.String())
					current.Reset()
				}
			} else {
				current.WriteRune(r)
			}
		}
		if current.Len() > 0 {
			raw = append(raw, current.String())
		}
	}

	return normalizeTokens(raw)
}

// normalizeTokens replaces long tokens with "<*>" and collapses consecutive
// wildcards. This ensures base64/protobuf payloads with embedded spaces don't
// fragment into many tokens and land in a different Drain bucket than their
// matching template.
func normalizeTokens(tokens []string) []string {
	result := make([]string, 0, len(tokens))
	prevWild := false
	for _, t := range tokens {
		if len(t) > longTokenThreshold {
			if prevWild {
				continue // collapse consecutive wildcards
			}
			result = append(result, "<*>")
			prevWild = true
		} else {
			if t == "<*>" {
				if prevWild {
					continue
				}
				prevWild = true
			} else {
				prevWild = false
			}
			result = append(result, t)
		}
	}
	return result
}

