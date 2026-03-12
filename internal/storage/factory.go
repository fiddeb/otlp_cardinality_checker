// Package storage provides storage implementations for telemetry metadata.
package storage

import (
	"log"

	"github.com/fidde/otlp_cardinality_checker/internal/storage/memory"
)

// DefaultPodLogServiceLabels is the ordered list of resource attribute keys used
// for service name discovery when POD_LOG_ENRICHMENT is enabled.
var DefaultPodLogServiceLabels = []string{
	"service_name", "service", "app", "application", "name",
	"app_kubernetes_io_name", "k8s.container.name", "k8s.deployment.name",
	"k8s.pod.name", "container", "component", "workload", "job",
}

// Config holds storage configuration.
type Config struct {
	// UseAutoTemplate enables Drain-style automatic log template extraction.
	UseAutoTemplate bool

	// MaxWatchedFields is the maximum number of simultaneously watched attribute keys.
	// Defaults to 10.
	MaxWatchedFields int

	// PodLogEnrichment enables pod log enrichment (service name discovery and
	// severity inference from body). Controlled by POD_LOG_ENRICHMENT env var.
	PodLogEnrichment bool

	// PodLogServiceLabels is the ordered list of resource attribute keys for
	// service name discovery. Defaults to DefaultPodLogServiceLabels.
	PodLogServiceLabels []string
}

// DefaultConfig returns default storage configuration.
func DefaultConfig() Config {
	return Config{
		UseAutoTemplate:     true,
		MaxWatchedFields:    10,
		PodLogEnrichment:    false,
		PodLogServiceLabels: DefaultPodLogServiceLabels,
	}
}

// NewStorage creates a new in-memory storage implementation.
func NewStorage(cfg Config) Storage {
	log.Printf("Using in-memory storage (autotemplate: %v, max_watched_fields: %d, pod_log_enrichment: %v)",
		cfg.UseAutoTemplate, cfg.MaxWatchedFields, cfg.PodLogEnrichment)
	return memory.NewWithConfig(cfg.UseAutoTemplate, cfg.MaxWatchedFields, cfg.PodLogEnrichment, cfg.PodLogServiceLabels)
}
