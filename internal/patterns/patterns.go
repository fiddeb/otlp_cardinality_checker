package patterns

import (
	"fmt"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
)

// Pattern represents a single log template pattern
type Pattern struct {
	Name               string `yaml:"name"`
	Regex              string `yaml:"regex"`
	Placeholder        string `yaml:"placeholder"`
	Description        string `yaml:"description"`
	RequiredSubstring  string `yaml:"required_substring"`
}

// PatternsConfig represents the patterns configuration file
type PatternsConfig struct {
	Patterns []Pattern `yaml:"patterns"`
}

// CompiledPattern is a pattern with compiled regex
type CompiledPattern struct {
	Name               string
	Regex              *regexp.Regexp
	Placeholder        string
	Description        string
	RequiredSubstring  string // if non-empty, skip this pattern when the body does not contain this substring
}

// LoadPatterns loads patterns from a YAML file
func LoadPatterns(filepath string) ([]CompiledPattern, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("reading patterns file: %w", err)
	}

	var config PatternsConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parsing patterns YAML: %w", err)
	}

	compiled := make([]CompiledPattern, 0, len(config.Patterns))
	for _, p := range config.Patterns {
		regex, err := regexp.Compile(p.Regex)
		if err != nil {
			return nil, fmt.Errorf("compiling pattern %s: %w", p.Name, err)
		}

		compiled = append(compiled, CompiledPattern{
			Name:              p.Name,
			Regex:             regex,
			Placeholder:       p.Placeholder,
			Description:       p.Description,
			RequiredSubstring: p.RequiredSubstring,
		})
	}

	return compiled, nil
}

// DrainPreMaskPatterns returns patterns that should be applied to log messages
// before Drain tokenization. The goal is to normalize high-cardinality tokens
// into stable, readable placeholders so that structurally identical messages
// land in the same Drain cluster.
//
// Pattern order matters: more specific patterns must run before generic ones.
// In particular, quoted access log patterns must run before UUID/hex patterns.
func DrainPreMaskPatterns() []CompiledPattern {
	return []CompiledPattern{
		// --- Access log / HTTP proxy log patterns ---

		// Quoted referrer/redirect URL — MUST run before the request-line pattern
		// so that "https://..." inside quotes doesn't confuse the request line regex.
		// Example: "https://app.www.svenskaspel.se/spela" → <URL>
		{
			Name:              "quoted_url",
			Regex:             regexp.MustCompile(`"https?://[^"]*"`),
			Placeholder:       "<URL>",
			RequiredSubstring: "\"",
			Description:       "Quoted HTTP/HTTPS URL (referrer in access logs)",
		},
		// Nginx/Apache/Envoy access log request line + status code combined.
		// Matches the quoted request AND the status code that immediately follows,
		// so byte-count fields further right are NOT misidentified as status codes.
		// `"GET /_numbergames/mcl/eurojackpot/play HTTP/1.1" 200` → `GET <URI> <STATUSCODE>`
		{
			Name:              "http_access_log_request",
			Regex:             regexp.MustCompile(`"(GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS)\s+\S+\s+HTTP/[0-9.]+"\s+[1-5][0-9]{2}\b`),
			Placeholder:       "$1 <URI> <STATUSCODE>",
			RequiredSubstring: "\"",
			Description:       "Quoted HTTP request line + status code in access/proxy logs",
		},
		// IPv4 addresses — produce the readable <IP> label rather than Drain's <*>.
		{
			Name:              "ipv4",
			Regex:             regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`),
			Placeholder:       "<IP>",
			RequiredSubstring: ".",
			Description:       "IPv4 address",
		},

		// --- Identifier patterns ---

		// UUIDs — must come before generic hex to avoid partial matches.
		{
			Name:              "uuid",
			Regex:             regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`),
			Placeholder:       "<ID>",
			Description:       "UUID — replaced before Drain tokenisation",
			RequiredSubstring: "-",
		},
		// HTTP method + path: preserve method and up to 2 literal (non-ID) path segments, mask the rest.
		// "GET /api/v1/users/123/orders" → "GET /api/v1/<PATH>"
		{
			Name:              "http_path",
			Regex:             regexp.MustCompile(`\b(GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS)\s+((?:/[a-zA-Z][a-zA-Z0-9._~-]*){1,2})/\S*`),
			Placeholder:       "$1 $2/<PATH>",
			Description:       "HTTP method + path - keep verb + first two literal segments",
			RequiredSubstring: "/",
		},
		// Bare absolute path with a variable segment (digit or UUID-like) anywhere.
		// "/users/123", "/orders/abc-123/items" → "<PATH>"
		{
			Name:              "path_with_id",
			Regex:             regexp.MustCompile(`(?:^|\s)(/(?:[a-zA-Z0-9._~-]+/)*[0-9][a-zA-Z0-9._~-]*(?:/[a-zA-Z0-9._~-]*)*)(?:\s|$)`),
			Placeholder:       " <PATH> ",
			Description:       "Absolute path containing a numeric segment",
			RequiredSubstring: "/",
		},
		// Query strings — remove entirely so they don't create unique clusters.
		{
			Name:              "query_string",
			Regex:             regexp.MustCompile(`\?[^\s]*`),
			Placeholder:       "",
			Description:       "URL query string — stripped before Drain",
			RequiredSubstring: "?",
		},
		// Hex IDs >= 8 chars that are not already inside a UUID.
		{
			Name:              "hex_id",
			Regex:             regexp.MustCompile(`\b[0-9a-f]{8,}\b`),
			Placeholder:       "<ID>",
			Description:       "Hex identifier",
			RequiredSubstring: "",
		},
	}
}

// DefaultPatterns returns the default compiled patterns (fallback if config file not found)
func DefaultPatterns() []CompiledPattern {
	return []CompiledPattern{
		{
			Name:        "timestamp",
			Regex:       regexp.MustCompile(`\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}`),
			Placeholder: "<TIMESTAMP>",
			Description: "ISO-like timestamps",
		},
		{
			Name:        "uuid",
			Regex:       regexp.MustCompile(`\b[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\b`),
			Placeholder: "<UUID>",
			Description: "Standard UUID format",
		},
		{
			Name:        "email",
			Regex:       regexp.MustCompile(`\b[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}\b`),
			Placeholder: "<EMAIL>",
			Description: "Email addresses",
		},
		// SQL patterns - normalize queries while preserving verb + table
		{
			Name:        "sql_select",
			Regex:       regexp.MustCompile(`(db/query:\s*SELECT\s+(?:.*?\s+)?FROM\s+\w+)(?:\s+.+)?$`),
			Placeholder: "$1 <WHERE>",
			Description: "SQL SELECT queries - keep table, mask WHERE",
		},
		{
			Name:        "sql_delete",
			Regex:       regexp.MustCompile(`(db/query:\s*DELETE\s+FROM\s+\w+)(?:\s+.+)?$`),
			Placeholder: "$1 <WHERE>",
			Description: "SQL DELETE queries - keep table, mask WHERE",
		},
		{
			Name:        "sql_update",
			Regex:       regexp.MustCompile(`(db/query:\s*UPDATE\s+\w+)\s+SET\s+.+$`),
			Placeholder: "$1 <SET>",
			Description: "SQL UPDATE queries - keep table, mask SET/WHERE",
		},
		{
			Name:        "sql_insert",
			Regex:       regexp.MustCompile(`(db/query:\s*INSERT\s+INTO\s+\w+)(?:\s+.+)?$`),
			Placeholder: "$1 <VALUES>",
			Description: "SQL INSERT queries - keep table, mask VALUES",
		},
		{
			Name:        "service_method",
			Regex:       regexp.MustCompile(`^([a-zA-Z][a-zA-Z0-9_-]*)/([a-zA-Z][a-zA-Z0-9]+)$`),
			Placeholder: "$1/<METHOD>",
			Description: "gRPC or internal service/method style spans",
		},
		// HTTP span names: preserve verb + first path segment (resource type), mask the rest.
		// "GET /api/v1/users/123" → "GET /api/v1/<PATH>"
		// "POST /orders/456/items" → "POST /orders/<PATH>"
		// Two-segment keep only when neither segment starts with a digit or looks like an ID.
		{
			Name:        "http_span_path",
			Regex:       regexp.MustCompile(`\b(GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS)\s+((?:/[a-zA-Z][a-zA-Z0-9._~-]*){1,2})/\S*`),
			Placeholder: "$1 $2/<PATH>",
			Description: "HTTP span: keep verb + first two literal path segments, mask the rest",
		},
		{
			Name:        "url",
			Regex:       regexp.MustCompile(`https?://[^\s]+`),
			Placeholder: "<URL>",
			Description: "Absolute HTTP/HTTPS URLs",
		},
		{
			Name:        "duration",
			Regex:       regexp.MustCompile(`\d+(?:\.\d+)?(?:µs|ms|s|m|h)\b`),
			Placeholder: "<DURATION>",
			Description: "Time durations with units",
		},
		{
			Name:        "size",
			Regex:       regexp.MustCompile(`\d+(?:\.\d+)?(?:B|KB|MB|GB)\b`),
			Placeholder: "<SIZE>",
			Description: "File/memory sizes with units",
		},
		{
			Name:        "ip",
			Regex:       regexp.MustCompile(`\[::1\]|\b(?:\d{1,3}\.){3}\d{1,3}\b`),
			Placeholder: "<IP>",
			Description: "IPv4 addresses and localhost IPv6",
		},
		{
			Name:        "hex",
			Regex:       regexp.MustCompile(`\b[0-9a-f]{8,}\b`),
			Placeholder: "<HEX>",
			Description: "Long hexadecimal strings",
		},
		{
			Name:        "number",
			Regex:       regexp.MustCompile(`\b\d+\b`),
			Placeholder: "<NUM>",
			Description: "Any numeric value",
		},
	}
}
