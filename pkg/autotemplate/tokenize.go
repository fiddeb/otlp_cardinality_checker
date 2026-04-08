package autotemplate

import (
	"sync"
	"unicode"
)

// longTokenThreshold is the character length above which a token is considered
// a high-entropy variable (e.g. base64 blobs, proto payloads with embedded
// spaces). Such tokens are collapsed to "<*>" so they don't inflate the token
// count and prevent cluster matching in Drain's length-bucketed tree.
const longTokenThreshold = 30

// isDigit returns true for ASCII '0'-'9'.
func isDigit(c byte) bool { return c >= '0' && c <= '9' }

// isHexDigit returns true for ASCII hex digits.
func isHexDigit(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

// isVariableToken applies cheap heuristic checks to decide whether a single
// token looks like a high-entropy variable that should become "<*>".
//
// Recognised patterns (no regexp, pure byte scanning):
//   - Pure decimal numbers (including negative and decimal point): -123, 3.14
//   - ISO-8601-ish timestamps: 2025-09-01T05, 2025-09-01, 05:39:27.100Z
//   - Pure hex strings >= 8 chars: trace IDs, span IDs, UUIDs
//   - IPv4 addresses: 10.0.0.1
//   - Tokens that are mostly digits (>60% digit characters, len >= 4)
func isVariableToken(t string) bool {
	n := len(t)
	if n == 0 {
		return false
	}

	// --- Pure number (optional leading minus, digits, at most one dot) ---
	// Examples: "123", "-42", "3.14", "0.001"
	start := 0
	if t[0] == '-' {
		start = 1
	}
	if start < n && isDigit(t[start]) {
		allNum := true
		dots := 0
		for i := start; i < n; i++ {
			if t[i] == '.' {
				dots++
				if dots > 1 {
					allNum = false
					break
				}
			} else if !isDigit(t[i]) {
				allNum = false
				break
			}
		}
		if allNum && dots <= 1 && n-start > 0 {
			return true
		}
	}

	// --- ISO-8601-ish date/time fragments ---
	// Matches: 2025-09-01T05, 2025-09-01, 05:39:27.100Z, 10:55, etc.
	// Heuristic: starts with digit, contains '-' or ':', and is mostly digits/separators.
	if n >= 4 && isDigit(t[0]) {
		hasSep := false
		varChars := 0  // digits + date/time separators
		for i := 0; i < n; i++ {
			c := t[i]
			if isDigit(c) {
				varChars++
			} else if c == '-' || c == ':' || c == '.' || c == 'T' || c == 'Z' {
				varChars++
				if c == '-' || c == ':' || c == 'T' {
					hasSep = true
				}
			}
		}
		if hasSep && varChars*100/n >= 80 {
			return true
		}
	}

	// --- Pure hex strings >= 8 chars (trace IDs, UUIDs, span IDs) ---
	if n >= 8 {
		allHex := true
		for i := 0; i < n; i++ {
			c := t[i]
			if !isHexDigit(c) && c != '-' {
				allHex = false
				break
			}
		}
		if allHex {
			return true
		}
	}

	// --- IPv4 addresses: 4 groups of digits separated by dots ---
	if n >= 7 && isDigit(t[0]) {
		groups := 1
		allIPv4 := true
		for i := 0; i < n; i++ {
			c := t[i]
			if c == '.' {
				groups++
			} else if !isDigit(c) {
				allIPv4 = false
				break
			}
		}
		if allIPv4 && groups == 4 {
			return true
		}
	}

	// --- Mostly-digit tokens (>= 60% digits, at least 4 chars) ---
	// Catches leftover fragments like "49436Z", "27100Z", combined number+suffix.
	if n >= 4 {
		digits := 0
		for i := 0; i < n; i++ {
			if isDigit(t[i]) {
				digits++
			}
		}
		if digits*100/n >= 60 {
			return true
		}
	}

	return false
}

// tokenPool holds reusable token slices to reduce allocation pressure.
// Each pooled item is a *[]string that callers append into.
var tokenPool = sync.Pool{
	New: func() any {
		s := make([]string, 0, 32)
		return &s
	},
}

// getTokenBuf retrieves a zeroed token slice from the pool.
func getTokenBuf() *[]string {
	buf := tokenPool.Get().(*[]string)
	*buf = (*buf)[:0]
	return buf
}

// putTokenBuf returns a token slice to the pool.
// Only return slices that haven't grown beyond a reasonable cap to avoid
// retaining oversized buffers.
func putTokenBuf(buf *[]string) {
	if cap(*buf) <= 256 {
		tokenPool.Put(buf)
	}
}

// tokenize splits a log message into tokens using whitespace and configured
// delimiters, then normalizes the result by:
//   - replacing tokens longer than longTokenThreshold with "<*>"
//   - collapsing consecutive "<*>" wildcards into a single one
//
// The returned slice is freshly allocated (safe to store), but internal work
// buffers are pooled to reduce GC pressure.
func tokenize(message string, extraDelimiters []rune) []string {
	buf := getTokenBuf()

	if len(extraDelimiters) == 0 {
		// Fast path: whitespace-only splitting without strings.Fields allocation.
		// Walk the string once, slicing substrings directly from message.
		start := -1
		for i := 0; i < len(message); i++ {
			c := message[i]
			// ASCII fast-path for whitespace (covers space, tab, newline, cr)
			isSpace := c == ' ' || c == '\t' || c == '\n' || c == '\r'
			if !isSpace && c < 0x80 {
				if start < 0 {
					start = i
				}
				continue
			}
			if !isSpace {
				// Non-ASCII: fall back to unicode check
				// Find the full rune
				r := rune(c)
				size := 1
				if c >= 0x80 {
					r, size = decodeRune(message[i:])
				}
				if !unicode.IsSpace(r) {
					if start < 0 {
						start = i
					}
					i += size - 1
					continue
				}
			}
			// whitespace
			if start >= 0 {
				*buf = append(*buf, message[start:i])
				start = -1
			}
		}
		if start >= 0 {
			*buf = append(*buf, message[start:])
		}
	} else {
		// Build delimiter set as a 128-bit ASCII bitmap + fallback map for non-ASCII.
		var asciiDelim [2]uint64 // bitmap for bytes 0-127
		var runeDelim map[rune]struct{}
		for _, r := range extraDelimiters {
			if r < 128 {
				asciiDelim[r/64] |= 1 << (r % 64)
			} else {
				if runeDelim == nil {
					runeDelim = make(map[rune]struct{})
				}
				runeDelim[r] = struct{}{}
			}
		}

		start := -1
		for i := 0; i < len(message); {
			c := message[i]
			var isSep bool
			size := 1

			if c < 0x80 {
				// ASCII fast path
				isSep = c == ' ' || c == '\t' || c == '\n' || c == '\r' ||
					(asciiDelim[c/64]&(1<<(c%64))) != 0
			} else {
				r, s := decodeRune(message[i:])
				size = s
				isSep = unicode.IsSpace(r)
				if !isSep && runeDelim != nil {
					_, isSep = runeDelim[r]
				}
			}

			if isSep {
				if start >= 0 {
					*buf = append(*buf, message[start:i])
					start = -1
				}
			} else if start < 0 {
				start = i
			}
			i += size
		}
		if start >= 0 {
			*buf = append(*buf, message[start:])
		}
	}

	result := normalizeTokens(*buf)
	putTokenBuf(buf)
	return result
}

// decodeRune decodes the first UTF-8 rune from s.
// Returns the rune and its byte width.
func decodeRune(s string) (rune, int) {
	r, size := rune(s[0]), 1
	if r < 0x80 {
		return r, 1
	}
	// Inline a minimal UTF-8 decoder to avoid importing unicode/utf8
	// for a single function (the compiler may inline this).
	b0 := s[0]
	switch {
	case b0 < 0xC0:
		return 0xFFFD, 1 // invalid
	case b0 < 0xE0:
		if len(s) < 2 {
			return 0xFFFD, 1
		}
		r = rune(b0&0x1F)<<6 | rune(s[1]&0x3F)
		size = 2
	case b0 < 0xF0:
		if len(s) < 3 {
			return 0xFFFD, 1
		}
		r = rune(b0&0x0F)<<12 | rune(s[1]&0x3F)<<6 | rune(s[2]&0x3F)
		size = 3
	default:
		if len(s) < 4 {
			return 0xFFFD, 1
		}
		r = rune(b0&0x07)<<18 | rune(s[1]&0x3F)<<12 | rune(s[2]&0x3F)<<6 | rune(s[3]&0x3F)
		size = 4
	}
	return r, size
}

// normalizeTokens replaces long tokens and recognised variable patterns with
// "<*>" and collapses consecutive wildcards. This ensures timestamps, trace IDs,
// numbers, and base64/protobuf payloads don't fragment into unique Drain
// clusters when they are really the same log template.
//
// The returned slice is a newly allocated copy (safe to store in clusters).
// The input slice is NOT modified.
func normalizeTokens(tokens []string) []string {
	result := make([]string, 0, len(tokens))
	prevWild := false
	for _, t := range tokens {
		if len(t) > longTokenThreshold || isVariableToken(t) {
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

