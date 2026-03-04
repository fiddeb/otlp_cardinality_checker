// Package main is the entry point for the OTLP Cardinality Checker.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/fidde/otlp_cardinality_checker/internal/api"
	"github.com/fidde/otlp_cardinality_checker/internal/receiver"
	"github.com/fidde/otlp_cardinality_checker/internal/storage"
	"github.com/fidde/otlp_cardinality_checker/internal/version"
)

func main() {
	// Handle --version / -version flag before anything else.
	if len(os.Args) == 2 && (os.Args[1] == "--version" || os.Args[1] == "-version" || os.Args[1] == "version") {
		fmt.Printf("occ %s (commit: %s, built: %s)\n", version.Version, version.Commit, version.BuildDate)
		os.Exit(0)
	}

	// Parse --watch-fields=key1,key2 flag (additive at startup for Kafka replay).
	var watchFieldsRaw string
	for _, arg := range os.Args[1:] {
		if strings.HasPrefix(arg, "--watch-fields=") {
			watchFieldsRaw = strings.TrimPrefix(arg, "--watch-fields=")
		}
	}
	var watchFields []string
	if watchFieldsRaw != "" {
		for _, k := range strings.Split(watchFieldsRaw, ",") {
			k = strings.TrimSpace(k)
			if k != "" {
				watchFields = append(watchFields, k)
			}
		}
	}

	log.Printf("Starting OTLP Cardinality Checker %s (commit: %s, built: %s)", version.Version, version.Commit, version.BuildDate)

	// Configure storage from environment
	useAutoTemplate := getEnvBool("USE_AUTOTEMPLATE", true)

	storageCfg := storage.DefaultConfig()
	storageCfg.UseAutoTemplate = useAutoTemplate

	if useAutoTemplate {
		log.Println("Autotemplate mode enabled (Drain-style extraction)")
	} else {
		log.Println("Using regex-based template extraction")
	}

	// Validate --watch-fields count against configured limit.
	if len(watchFields) > storageCfg.MaxWatchedFields {
		log.Fatalf("--watch-fields specifies %d keys but MaxWatchedFields limit is %d", len(watchFields), storageCfg.MaxWatchedFields)
	}

	// Create storage (always in-memory)
	store := storage.NewStorage(storageCfg)
	defer func() {
		if err := store.Close(); err != nil {
			log.Printf("Error closing storage: %v", err)
		}
	}()

	// Activate deep watch for any startup fields.
	if len(watchFields) > 0 {
		ctx := context.Background()
		for _, key := range watchFields {
			if err := store.WatchAttribute(ctx, key); err != nil {
				log.Fatalf("Failed to watch attribute %q: %v", key, err)
			}
			log.Printf("Deep watch activated for attribute key %q", key)
		}
	}

	// Create OTLP receivers
	otlpHTTPAddr := getEnv("OTLP_HTTP_ADDR", "0.0.0.0:4318")
	otlpGRPCAddr := getEnv("OTLP_GRPC_ADDR", "0.0.0.0:4317")
	httpReceiver := receiver.NewHTTPReceiver(otlpHTTPAddr, store)
	grpcReceiver := receiver.NewGRPCReceiver(otlpGRPCAddr, store)

	// Create REST API server
	apiAddr := getEnv("API_ADDR", "0.0.0.0:8080")
	apiServer := api.NewServer(apiAddr, store)

	// Start pprof server for profiling (separate port)
	pprofAddr := getEnv("PPROF_ADDR", "localhost:6060")
	go func() {
		log.Printf("Starting pprof server on http://%s/debug/pprof", pprofAddr)
		if err := http.ListenAndServe(pprofAddr, nil); err != nil {
			log.Printf("pprof server error: %v", err)
		}
	}()

	// Start servers in goroutines
	errChan := make(chan error, 3)

	go func() {
		log.Printf("Starting OTLP HTTP receiver on %s", otlpHTTPAddr)
		if err := httpReceiver.Start(); err != nil {
			errChan <- fmt.Errorf("OTLP HTTP receiver error: %w", err)
		}
	}()

	go func() {
		log.Printf("Starting OTLP gRPC receiver on %s", otlpGRPCAddr)
		if err := grpcReceiver.Start(); err != nil {
			errChan <- fmt.Errorf("OTLP gRPC receiver error: %w", err)
		}
	}()

	go func() {
		log.Printf("Starting REST API server on %s", apiAddr)
		if err := apiServer.Start(); err != nil {
			errChan <- fmt.Errorf("API server error: %w", err)
		}
	}()

	// Give servers time to start
	time.Sleep(100 * time.Millisecond)
	log.Println("All servers started successfully")
	log.Println("OTLP endpoints:")
	log.Printf("  - HTTP: http://%s/v1/metrics", otlpHTTPAddr)
	log.Printf("  - HTTP: http://%s/v1/traces", otlpHTTPAddr)
	log.Printf("  - HTTP: http://%s/v1/logs", otlpHTTPAddr)
	log.Printf("  - gRPC: %s", otlpGRPCAddr)
	log.Println("API endpoints:")
	log.Printf("  - Metrics: http://%s/api/v1/metrics", apiAddr)
	log.Printf("  - Spans: http://%s/api/v1/spans", apiAddr)
	log.Printf("  - Logs: http://%s/api/v1/logs", apiAddr)
	log.Printf("  - Services: http://%s/api/v1/services", apiAddr)
	log.Printf("  - Health: http://%s/health", apiAddr)
	log.Println("Profiling:")
	log.Printf("  - pprof: http://%s/debug/pprof", pprofAddr)

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errChan:
		log.Fatalf("Server error: %v", err)
	case sig := <-sigChan:
		log.Printf("Received signal: %v, shutting down...", sig)
	}

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Println("Shutting down servers...")
	if err := httpReceiver.Shutdown(shutdownCtx); err != nil {
		log.Printf("Error shutting down OTLP HTTP receiver: %v", err)
	}
	if err := grpcReceiver.Shutdown(shutdownCtx); err != nil {
		log.Printf("Error shutting down OTLP gRPC receiver: %v", err)
	}
	if err := apiServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("Error shutting down API server: %v", err)
	}

	log.Println("Closing storage...")
	if err := store.Close(); err != nil {
		log.Printf("Error closing storage: %v", err)
	}

	log.Println("Shutdown complete")
}

// getEnv gets an environment variable with a default fallback.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvBool gets a boolean environment variable with a default fallback.
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	return defaultValue
}
