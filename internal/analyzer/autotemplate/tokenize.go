package autotemplate

import (
	"strings"
	"sync"
	"unicode"
)

// tokenize splits a log message into tokens using whitespace and configured delimiters
func tokenize(message string, extraDelimiters []rune) []string {
	if len(extraDelimiters) == 0 {
		// Fast path: whitespace only
		return strings.Fields(message)
	}
	
	// Build delimiter map
	delims := make(map[rune]bool, len(extraDelimiters))
	for _, r := range extraDelimiters {
		delims[r] = true
	}
	
	var tokens []string
	var current strings.Builder
	
	for _, r := range message {
		if unicode.IsSpace(r) || delims[r] {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		} else {
			current.WriteRune(r)
		}
	}
	
	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}
	
	return tokens
}

// tokenizerPool reduces allocations for tokenization
var tokenizerPool = sync.Pool{
	New: func() interface{} {
		return &strings.Builder{}
	},
}
