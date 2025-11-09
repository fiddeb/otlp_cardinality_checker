// Package storage provides storage implementations for telemetry metadata.
package storage

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/fidde/otlp_cardinality_checker/internal/analyzer/autotemplate"
	"github.com/fidde/otlp_cardinality_checker/internal/storage/clickhouse"
	"github.com/fidde/otlp_cardinality_checker/internal/storage/memory"
)

// Config holds storage configuration.
type Config struct {
	// Backend selects the storage backend: "memory" or "clickhouse"
	Backend string

	// ClickHouse-specific config
	ClickHouseAddr string

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
		Backend:         "clickhouse",
		ClickHouseAddr:  "localhost:9000",
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

	case "clickhouse":
		log.Printf("Using ClickHouse storage: %s (autotemplate: %v)", cfg.ClickHouseAddr, cfg.UseAutoTemplate)
		
		chCfg := clickhouse.DefaultConfig()
		chCfg.Addr = cfg.ClickHouseAddr
		
		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
		
		store, err := clickhouse.NewStore(context.Background(), chCfg, logger)
		if err != nil {
			return nil, fmt.Errorf("creating ClickHouse store: %w", err)
		}
		return store, nil

	default:
		return nil, fmt.Errorf("unknown storage backend: %s (supported: memory, clickhouse)", cfg.Backend)
	}
}
