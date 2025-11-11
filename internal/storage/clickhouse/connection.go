package clickhouse

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

const (
	defaultMaxOpenConns = 10
	defaultMaxIdleConns = 5
	defaultDialTimeout  = 10 * time.Second
	defaultMaxRetries   = 3
	defaultRetryDelay   = 1 * time.Second
)

// ConnectionConfig holds ClickHouse connection parameters
type ConnectionConfig struct {
	Addr          string
	Database      string
	Username      string
	Password      string
	MaxOpenConns  int
	MaxIdleConns  int
	DialTimeout   time.Duration
	MaxRetries    int
	TLS           *tls.Config
	BatchSize     int           // Number of rows to buffer before flushing (default: 5000)
	FlushInterval time.Duration // Max time between flushes (default: 2s)
}

// DefaultConfig returns a connection config with sensible defaults
func DefaultConfig() *ConnectionConfig {
	return &ConnectionConfig{
		Addr:          "localhost:9000",
		Database:      "default",
		Username:      "default",
		Password:      "",
		MaxOpenConns:  defaultMaxOpenConns,
		MaxIdleConns:  defaultMaxIdleConns,
		DialTimeout:   defaultDialTimeout,
		MaxRetries:    defaultMaxRetries,
		TLS:           nil, // No TLS for local development
		BatchSize:     0,   // Use default from buffer.go
		FlushInterval: 0,   // Use default from buffer.go
	}
}

// Connect establishes a connection to ClickHouse with retry logic
func Connect(ctx context.Context, config *ConnectionConfig) (driver.Conn, error) {
	if config == nil {
		config = DefaultConfig()
	}

	opts := &clickhouse.Options{
		Addr: []string{config.Addr},
		Auth: clickhouse.Auth{
			Database: config.Database,
			Username: config.Username,
			Password: config.Password,
		},
		Settings: clickhouse.Settings{
			"max_execution_time": 60,
		},
		DialTimeout:      config.DialTimeout,
		MaxOpenConns:     config.MaxOpenConns,
		MaxIdleConns:     config.MaxIdleConns,
		ConnMaxLifetime:  time.Hour,
		ConnOpenStrategy: clickhouse.ConnOpenInOrder,
		TLS:              config.TLS,
	}

	// Retry logic with exponential backoff
	var conn driver.Conn
	var err error
	retryDelay := defaultRetryDelay

	for attempt := 1; attempt <= config.MaxRetries; attempt++ {
		conn, err = clickhouse.Open(opts)
		if err == nil {
			// Test connection
			if err = conn.Ping(ctx); err == nil {
				return conn, nil
			}
		}

		if attempt < config.MaxRetries {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(retryDelay):
				// Exponential backoff
				retryDelay *= 2
			}
		}
	}

	return nil, fmt.Errorf("failed to connect to ClickHouse after %d attempts: %w", config.MaxRetries, err)
}
