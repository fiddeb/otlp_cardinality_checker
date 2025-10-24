// Package main is the entry point for the OTLP Cardinality Checker.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fidde/otlp_cardinality_checker/internal/api"
	"github.com/fidde/otlp_cardinality_checker/internal/receiver"
	"github.com/fidde/otlp_cardinality_checker/internal/storage/memory"
)

func main() {
	log.Println("Starting OTLP Cardinality Checker...")

	// Create storage
	store := memory.New()

	// Create OTLP receivers
	otlpHTTPAddr := getEnv("OTLP_HTTP_ADDR", "0.0.0.0:4318")
	otlpGRPCAddr := getEnv("OTLP_GRPC_ADDR", "0.0.0.0:4317")
	httpReceiver := receiver.NewHTTPReceiver(otlpHTTPAddr, store)
	grpcReceiver := receiver.NewGRPCReceiver(otlpGRPCAddr, store)

	// Create REST API server
	apiAddr := getEnv("API_ADDR", "0.0.0.0:8080")
	apiServer := api.NewServer(apiAddr, store)

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

	log.Println("Shutdown complete")
}

// getEnv gets an environment variable with a default fallback.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
