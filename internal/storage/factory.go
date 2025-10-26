// Package storage provides storage implementations for telemetry metadata.
package storage

import (
	"fmt"
	"log"

	"github.com/fidde/otlp_cardinality_checker/internal/analyzer/autotemplate"
	"github.com/fidde/otlp_cardinality_checker/internal/storage/memory"
	"github.com/fidde/otlp_cardinality_checker/internal/storage/sqlite"
)

// Config holds storage configuration.
type Config struct {
	// Backend selects the storage backend: "memory" or "sqlite"
	Backend string

	// SQLite-specific config
	SQLiteDBPath string

	// Autotemplate config (shared)
	UseAutoTemplate bool
	AutoTemplateCfg autotemplate.Config
}

// DefaultConfig returns default storage configuration.
func DefaultConfig() Config {
	cfg := autotemplate.DefaultConfig()
	cfg.Shards = 4
	cfg.SimThreshold = 0.7

	return Config{
		Backend:         "memory",
		SQLiteDBPath:    "data/otlp_metadata.db",
		UseAutoTemplate: false,
		AutoTemplateCfg: cfg,
	}
}

// NewStorage creates a storage implementation based on configuration.
func NewStorage(cfg Config) (Storage, error) {
	switch cfg.Backend {
	case "memory":
		log.Printf("Using in-memory storage (autotemplate: %v)", cfg.UseAutoTemplate)
		return memory.NewWithAutoTemplate(cfg.UseAutoTemplate), nil

	case "sqlite":
		log.Printf("Using SQLite storage: %s (autotemplate: %v)", cfg.SQLiteDBPath, cfg.UseAutoTemplate)
		sqliteCfg := sqlite.DefaultConfig(cfg.SQLiteDBPath)
		sqliteCfg.UseAutoTemplate = cfg.UseAutoTemplate
		sqliteCfg.AutoTemplateCfg = cfg.AutoTemplateCfg

		store, err := sqlite.New(sqliteCfg)
		if err != nil {
			return nil, fmt.Errorf("creating SQLite store: %w", err)
		}
		return store, nil

	default:
		return nil, fmt.Errorf("unknown storage backend: %s (supported: memory, sqlite)", cfg.Backend)
	}
}
