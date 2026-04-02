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

// normalizeTokens replaces long tokens with "<*>" and collapses consecutive
// wildcards. This ensures base64/protobuf payloads with embedded spaces don't
// fragment into many tokens and land in a different Drain bucket than their
// matching template.
//
// The returned slice is a newly allocated copy (safe to store in clusters).
// The input slice is NOT modified.
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

