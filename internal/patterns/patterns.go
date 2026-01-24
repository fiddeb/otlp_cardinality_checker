package patterns

import (
	"fmt"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
)

// Pattern represents a single log template pattern
type Pattern struct {
	Name        string `yaml:"name"`
	Regex       string `yaml:"regex"`
	Placeholder string `yaml:"placeholder"`
	Description string `yaml:"description"`
}

// PatternsConfig represents the patterns configuration file
type PatternsConfig struct {
	Patterns []Pattern `yaml:"patterns"`
}

// CompiledPattern is a pattern with compiled regex
type CompiledPattern struct {
	Name        string
	Regex       *regexp.Regexp
	Placeholder string
	Description string
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
			Name:        p.Name,
			Regex:       regex,
			Placeholder: p.Placeholder,
			Description: p.Description,
		})
	}

	return compiled, nil
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
		{
			Name:        "url",
			Regex:       regexp.MustCompile(`https?://[^\s]+|\s(/[a-zA-Z0-9/_.-]+)`),
			Placeholder: " <URL>",
			Description: "HTTP/HTTPS URLs and absolute paths",
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
