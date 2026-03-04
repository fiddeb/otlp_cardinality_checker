// Package storage provides storage implementations for telemetry metadata.
package storage

import (
	"log"

	"github.com/fidde/otlp_cardinality_checker/internal/storage/memory"
)

// Config holds storage configuration.
type Config struct {
	// UseAutoTemplate enables Drain-style automatic log template extraction.
	UseAutoTemplate bool

	// MaxWatchedFields is the maximum number of simultaneously watched attribute keys.
	// Defaults to 10.
	MaxWatchedFields int
}

// DefaultConfig returns default storage configuration.
func DefaultConfig() Config {
	return Config{
		UseAutoTemplate:  true,
		MaxWatchedFields: 10,
	}
}

// NewStorage creates a new in-memory storage implementation.
func NewStorage(cfg Config) Storage {
	log.Printf("Using in-memory storage (autotemplate: %v, max_watched_fields: %d)", cfg.UseAutoTemplate, cfg.MaxWatchedFields)
	return memory.NewWithAutoTemplate(cfg.UseAutoTemplate, cfg.MaxWatchedFields)
}
