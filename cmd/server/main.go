// Package main is the entry point for the OTLP Cardinality Checker.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/fidde/otlp_cardinality_checker/internal/api"
	"github.com/fidde/otlp_cardinality_checker/internal/receiver"
	"github.com/fidde/otlp_cardinality_checker/internal/report"
	"github.com/fidde/otlp_cardinality_checker/internal/storage"
	"github.com/fidde/otlp_cardinality_checker/internal/storage/sessions"
	"github.com/fidde/otlp_cardinality_checker/internal/version"
	"github.com/fidde/otlp_cardinality_checker/pkg/models"
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

	// Parse CI/CD mode flags.
	minimal := parseBoolFlag("--minimal", "OCC_MINIMAL")
	idleTimeoutStr := parseStringFlag("--idle-timeout", "OCC_IDLE_TIMEOUT")
	reportOutput := parseStringFlag("--report-output", "OCC_REPORT_OUTPUT")
	reportFormat := parseStringFlag("--report-format", "OCC_REPORT_FORMAT")
	exitOnThreshold := parseBoolFlag("--exit-on-threshold", "OCC_EXIT_ON_THRESHOLD")
	sessionExport := parseStringFlag("--session-export", "OCC_SESSION_EXPORT")

	if reportFormat == "" {
		reportFormat = "text"
	}
	if reportFormat != "json" && reportFormat != "text" {
		log.Fatalf("Invalid --report-format %q: must be 'json' or 'text'", reportFormat)
	}

	var idleTimeout time.Duration
	if idleTimeoutStr != "" {
		var err error
		idleTimeout, err = time.ParseDuration(idleTimeoutStr)
		if err != nil {
			log.Fatalf("Invalid --idle-timeout %q: %v", idleTimeoutStr, err)
		}
	}

	if minimal {
		log.Println("Running in minimal mode (UI disabled)")
	} else {
		log.Println("Running in normal mode")
	}

	log.Printf("Starting OTLP Cardinality Checker %s (commit: %s, built: %s)", version.Version, version.Commit, version.BuildDate)

	// Configure storage from environment
	useAutoTemplate := getEnvBool("USE_AUTOTEMPLATE", true)
	podLogEnrichment := getEnvBool("POD_LOG_ENRICHMENT", false)

	storageCfg := storage.DefaultConfig()
	storageCfg.UseAutoTemplate = useAutoTemplate
	storageCfg.PodLogEnrichment = podLogEnrichment

	if rawLabels := os.Getenv("POD_LOG_SERVICE_LABELS"); rawLabels != "" {
		var labels []string
		for _, l := range strings.Split(rawLabels, ",") {
			l = strings.TrimSpace(l)
			if l != "" {
				labels = append(labels, l)
			}
		}
		if len(labels) > 0 {
			storageCfg.PodLogServiceLabels = labels
		}
	}

	if useAutoTemplate {
		log.Println("Autotemplate mode enabled (Drain-style extraction)")
	} else {
		log.Println("Using regex-based template extraction")
	}
	if podLogEnrichment {
		log.Printf("Pod log enrichment enabled (service_labels: %v)", storageCfg.PodLogServiceLabels)
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

	// Wire up activity tracking for idle timeout.
	var lastActivity atomic.Int64
	lastActivity.Store(time.Now().UnixNano())
	notifyActivity := func() { lastActivity.Store(time.Now().UnixNano()) }
	httpReceiver.OnActivity = notifyActivity
	grpcReceiver.OnActivity = notifyActivity

	// Create REST API server
	apiAddr := getEnv("API_ADDR", "0.0.0.0:8090")
	apiServer := api.NewServer(apiAddr, store, api.ServerOptions{DisableUI: minimal})

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

	// Start idle timeout checker if set.
	var idleStop chan struct{}
	if idleTimeout > 0 {
		log.Printf("Idle timeout set to %s (shutdown when no OTLP data received)", idleTimeout)
		idleStop = make(chan struct{})
		go func() {
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					last := time.Unix(0, lastActivity.Load())
					if time.Since(last) >= idleTimeout {
						log.Printf("No OTLP data received for %s, initiating shutdown...", idleTimeout)
						sigChan <- syscall.SIGTERM
						return
					}
				case <-idleStop:
					return
				}
			}
		}()
	}

	select {
	case err := <-errChan:
		log.Fatalf("Server error: %v", err)
	case sig := <-sigChan:
		log.Printf("Received signal: %v, shutting down...", sig)
	}

	if idleStop != nil {
		close(idleStop)
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

	// Generate report on shutdown if requested, or always on idle timeout.
	exitCode := 0
	if reportOutput != "" || idleTimeout > 0 {
		gen := report.NewGenerator(store)
		rpt, err := gen.Generate(shutdownCtx, idleTimeout)
		if err != nil {
			log.Printf("Error generating report: %v", err)
		} else {
			var formatted []byte
			switch reportFormat {
			case "json":
				formatted, err = report.FormatJSON(rpt)
			default:
				formatted, err = report.FormatText(rpt)
			}
			if err != nil {
				log.Printf("Error formatting report: %v", err)
			} else {
				if reportOutput != "" {
					if err := os.WriteFile(reportOutput, formatted, 0644); err != nil {
						log.Printf("Error writing report to %s: %v", reportOutput, err)
					} else {
						log.Printf("Report written to %s", reportOutput)
					}
				}
				// Always print text report to stdout.
				textOut := formatted
				if reportFormat != "text" {
					textOut, _ = report.FormatText(rpt)
				}
				fmt.Println(string(textOut))
			}

			// Calculate exit code from severity if requested.
			if exitOnThreshold {
				exitCode = rpt.MaxExitCode()
			}
		}
	} else if exitOnThreshold {
		// No report requested but exit-on-threshold set: still calculate.
		gen := report.NewGenerator(store)
		rpt, err := gen.Generate(shutdownCtx, 0)
		if err != nil {
			log.Printf("Error generating report for threshold check: %v", err)
		} else {
			exitCode = rpt.MaxExitCode()
		}
	}

	// Session export on shutdown if requested.
	if sessionExport != "" {
		if err := exportSession(shutdownCtx, store, sessionExport); err != nil {
			log.Printf("Error exporting session: %v", err)
		} else {
			log.Printf("Session exported to %s", sessionExport)
		}
	}

	log.Println("Closing storage...")
	if err := store.Close(); err != nil {
		log.Printf("Error closing storage: %v", err)
	}

	log.Println("Shutdown complete")
	if exitCode != 0 {
		os.Exit(exitCode)
	}
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

// parseBoolFlag checks os.Args for a boolean flag and falls back to an env var.
func parseBoolFlag(flag, envKey string) bool {
	for _, arg := range os.Args[1:] {
		if arg == flag {
			return true
		}
	}
	return getEnvBool(envKey, false)
}

// parseStringFlag checks os.Args for --key=value or --key value forms and falls back to an env var.
func parseStringFlag(flag, envKey string) string {
	prefix := flag + "="
	args := os.Args[1:]
	for i, arg := range args {
		if strings.HasPrefix(arg, prefix) {
			return strings.TrimPrefix(arg, prefix)
		}
		if arg == flag && i+1 < len(args) {
			return args[i+1]
		}
	}
	return os.Getenv(envKey)
}

// exportSession serializes the current storage state to a session JSON file.
func exportSession(ctx context.Context, store storage.Storage, path string) error {
	metrics, mErr := store.ListMetrics(ctx, "")
	spans, sErr := store.ListSpans(ctx, "")
	logs, lErr := store.ListLogs(ctx, "")
	attrs, aErr := store.ListAttributes(ctx, nil)
	services, svErr := store.ListServices(ctx)

	for _, err := range []error{mErr, sErr, lErr, aErr, svErr} {
		if err != nil {
			return fmt.Errorf("reading store data: %w", err)
		}
	}

	serializer := sessions.NewSerializer()
	sMetrics, err := serializer.MarshalMetrics(metrics)
	if err != nil {
		return fmt.Errorf("serializing metrics: %w", err)
	}
	sSpans, err := serializer.MarshalSpans(spans)
	if err != nil {
		return fmt.Errorf("serializing spans: %w", err)
	}
	sLogs, err := serializer.MarshalLogs(logs)
	if err != nil {
		return fmt.Errorf("serializing logs: %w", err)
	}
	sAttrs, err := serializer.MarshalAttributes(attrs)
	if err != nil {
		return fmt.Errorf("serializing attributes: %w", err)
	}

	session := &models.Session{
		Version:     1,
		ID:          fmt.Sprintf("export-%d", time.Now().Unix()),
		Description: "Exported on shutdown",
		Created:     time.Now().UTC(),
		Signals:     []string{"metrics", "traces", "logs"},
		Data: models.SessionData{
			Metrics:    sMetrics,
			Spans:      sSpans,
			Logs:       sLogs,
			Attributes: sAttrs,
		},
		Stats: models.SessionStats{
			MetricsCount:    len(metrics),
			SpansCount:      len(spans),
			LogsCount:       len(logs),
			AttributesCount: len(attrs),
			Services:        services,
		},
	}

	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("marshaling session: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}
