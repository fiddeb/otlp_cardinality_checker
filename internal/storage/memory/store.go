// Package memory provides an in-memory storage implementation for metadata.
package memory

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/fidde/otlp_cardinality_checker/pkg/models"
)

var (
	// ErrNotFound is returned when a requested item is not found
	ErrNotFound = errors.New("not found")
)

// Store is an in-memory storage for telemetry metadata.
type Store struct {
	// Metrics storage: metric name -> metadata
	metrics map[string]*models.MetricMetadata
	metricsmu sync.RWMutex

	// Spans storage: span name -> metadata
	spans map[string]*models.SpanMetadata
	spansmu sync.RWMutex

	// Logs storage: severity text -> metadata
	logs map[string]*models.LogMetadata
	logsmu sync.RWMutex

	// Services tracks all service names seen
	services map[string]struct{}
	servicesmu sync.RWMutex
}

// New creates a new in-memory store.
func New() *Store {
	return &Store{
		metrics:  make(map[string]*models.MetricMetadata),
		spans:    make(map[string]*models.SpanMetadata),
		logs:     make(map[string]*models.LogMetadata),
		services: make(map[string]struct{}),
	}
}

// StoreMetric stores or updates metric metadata.
func (s *Store) StoreMetric(ctx context.Context, metric *models.MetricMetadata) error {
	if metric == nil {
		return errors.New("metric cannot be nil")
	}
	if metric.Name == "" {
		return errors.New("metric name cannot be empty")
	}

	s.metricsmu.Lock()
	defer s.metricsmu.Unlock()

	// Track services
	s.trackServices(metric.Services)

	// If metric exists, merge with existing
	if existing, exists := s.metrics[metric.Name]; exists {
		existing.MergeMetricMetadata(metric)
		return nil
	}

	// Store new metric
	s.metrics[metric.Name] = metric
	return nil
}

// GetMetric retrieves metric metadata by name.
func (s *Store) GetMetric(ctx context.Context, name string) (*models.MetricMetadata, error) {
	s.metricsmu.RLock()
	defer s.metricsmu.RUnlock()

	metric, exists := s.metrics[name]
	if !exists {
		return nil, fmt.Errorf("metric %s: %w", name, ErrNotFound)
	}

	return metric, nil
}

// ListMetrics returns all metrics, optionally filtered by service name.
func (s *Store) ListMetrics(ctx context.Context, serviceName string) ([]*models.MetricMetadata, error) {
	s.metricsmu.RLock()
	defer s.metricsmu.RUnlock()

	metrics := make([]*models.MetricMetadata, 0, len(s.metrics))
	for _, metric := range s.metrics {
		// Filter by service if specified
		if serviceName != "" {
			if _, hasService := metric.Services[serviceName]; !hasService {
				continue
			}
		}
		metrics = append(metrics, metric)
	}

	// Sort by name for consistency
	sort.Slice(metrics, func(i, j int) bool {
		return metrics[i].Name < metrics[j].Name
	})

	return metrics, nil
}

// StoreSpan stores or updates span metadata.
func (s *Store) StoreSpan(ctx context.Context, span *models.SpanMetadata) error {
	if span == nil {
		return errors.New("span cannot be nil")
	}
	if span.Name == "" {
		return errors.New("span name cannot be empty")
	}

	s.spansmu.Lock()
	defer s.spansmu.Unlock()

	// Track services
	s.trackServices(span.Services)

	// If span exists, merge with existing
	if existing, exists := s.spans[span.Name]; exists {
		// Update span count
		existing.SpanCount += span.SpanCount

		// Merge attribute keys
		for key, keyMeta := range span.AttributeKeys {
			if existingKey, exists := existing.AttributeKeys[key]; exists {
				existingKey.Count += keyMeta.Count
			} else {
				existing.AttributeKeys[key] = keyMeta
			}
		}

		// Merge resource keys
		for key, keyMeta := range span.ResourceKeys {
			if existingKey, exists := existing.ResourceKeys[key]; exists {
				existingKey.Count += keyMeta.Count
			} else {
				existing.ResourceKeys[key] = keyMeta
			}
		}

		// Merge services
		for service, count := range span.Services {
			existing.Services[service] += count
		}

		return nil
	}

	// Store new span
	s.spans[span.Name] = span
	return nil
}

// GetSpan retrieves span metadata by name.
func (s *Store) GetSpan(ctx context.Context, name string) (*models.SpanMetadata, error) {
	s.spansmu.RLock()
	defer s.spansmu.RUnlock()

	span, exists := s.spans[name]
	if !exists {
		return nil, fmt.Errorf("span %s: %w", name, ErrNotFound)
	}

	return span, nil
}

// ListSpans returns all spans, optionally filtered by service name.
func (s *Store) ListSpans(ctx context.Context, serviceName string) ([]*models.SpanMetadata, error) {
	s.spansmu.RLock()
	defer s.spansmu.RUnlock()

	spans := make([]*models.SpanMetadata, 0, len(s.spans))
	for _, span := range s.spans {
		// Filter by service if specified
		if serviceName != "" {
			if _, hasService := span.Services[serviceName]; !hasService {
				continue
			}
		}
		spans = append(spans, span)
	}

	// Sort by name for consistency
	sort.Slice(spans, func(i, j int) bool {
		return spans[i].Name < spans[j].Name
	})

	return spans, nil
}

// StoreLog stores or updates log metadata.
func (s *Store) StoreLog(ctx context.Context, log *models.LogMetadata) error {
	if log == nil {
		return errors.New("log cannot be nil")
	}

	s.logsmu.Lock()
	defer s.logsmu.Unlock()

	// Track services
	s.trackServices(log.Services)

	key := log.SeverityText
	if key == "" {
		key = "UNSET"
	}

	// If log exists, merge with existing
	if existing, exists := s.logs[key]; exists {
		// Update record count
		existing.RecordCount += log.RecordCount

		// Merge attribute keys
		for key, keyMeta := range log.AttributeKeys {
			if existingKey, exists := existing.AttributeKeys[key]; exists {
				existingKey.Count += keyMeta.Count
			} else {
				existing.AttributeKeys[key] = keyMeta
			}
		}

		// Merge resource keys
		for key, keyMeta := range log.ResourceKeys {
			if existingKey, exists := existing.ResourceKeys[key]; exists {
				existingKey.Count += keyMeta.Count
			} else {
				existing.ResourceKeys[key] = keyMeta
			}
		}

		// Merge services
		for service, count := range log.Services {
			existing.Services[service] += count
		}

		return nil
	}

	// Store new log
	s.logs[key] = log
	return nil
}

// GetLog retrieves log metadata by severity text.
func (s *Store) GetLog(ctx context.Context, severityText string) (*models.LogMetadata, error) {
	s.logsmu.RLock()
	defer s.logsmu.RUnlock()

	if severityText == "" {
		severityText = "UNSET"
	}

	log, exists := s.logs[severityText]
	if !exists {
		return nil, fmt.Errorf("log severity %s: %w", severityText, ErrNotFound)
	}

	return log, nil
}

// ListLogs returns all log metadata, optionally filtered by service name.
func (s *Store) ListLogs(ctx context.Context, serviceName string) ([]*models.LogMetadata, error) {
	s.logsmu.RLock()
	defer s.logsmu.RUnlock()

	logs := make([]*models.LogMetadata, 0, len(s.logs))
	for _, log := range s.logs {
		// Filter by service if specified
		if serviceName != "" {
			if _, hasService := log.Services[serviceName]; !hasService {
				continue
			}
		}
		logs = append(logs, log)
	}

	// Sort by severity for consistency
	sort.Slice(logs, func(i, j int) bool {
		return logs[i].SeverityText < logs[j].SeverityText
	})

	return logs, nil
}

// ListServices returns all service names seen.
func (s *Store) ListServices(ctx context.Context) ([]string, error) {
	s.servicesmu.RLock()
	defer s.servicesmu.RUnlock()

	services := make([]string, 0, len(s.services))
	for service := range s.services {
		services = append(services, service)
	}
	sort.Strings(services)

	return services, nil
}

// GetServiceOverview returns a summary of all telemetry for a service.
func (s *Store) GetServiceOverview(ctx context.Context, serviceName string) (*ServiceOverview, error) {
	if serviceName == "" {
		return nil, errors.New("service name cannot be empty")
	}

	metrics, err := s.ListMetrics(ctx, serviceName)
	if err != nil {
		return nil, fmt.Errorf("listing metrics: %w", err)
	}

	spans, err := s.ListSpans(ctx, serviceName)
	if err != nil {
		return nil, fmt.Errorf("listing spans: %w", err)
	}

	logs, err := s.ListLogs(ctx, serviceName)
	if err != nil {
		return nil, fmt.Errorf("listing logs: %w", err)
	}

	return &ServiceOverview{
		ServiceName:  serviceName,
		MetricCount:  len(metrics),
		SpanCount:    len(spans),
		LogCount:     len(logs),
		Metrics:      metrics,
		Spans:        spans,
		Logs:         logs,
		GeneratedAt:  time.Now(),
	}, nil
}

// ServiceOverview contains a summary of all telemetry for a service.
type ServiceOverview struct {
	ServiceName string                     `json:"service_name"`
	MetricCount int                        `json:"metric_count"`
	SpanCount   int                        `json:"span_count"`
	LogCount    int                        `json:"log_count"`
	Metrics     []*models.MetricMetadata   `json:"metrics"`
	Spans       []*models.SpanMetadata     `json:"spans"`
	Logs        []*models.LogMetadata      `json:"logs"`
	GeneratedAt time.Time                  `json:"generated_at"`
}

// Clear removes all stored data.
func (s *Store) Clear(ctx context.Context) error {
	s.metricsmu.Lock()
	s.spansmu.Lock()
	s.logsmu.Lock()
	s.servicesmu.Lock()
	defer s.metricsmu.Unlock()
	defer s.spansmu.Unlock()
	defer s.logsmu.Unlock()
	defer s.servicesmu.Unlock()

	s.metrics = make(map[string]*models.MetricMetadata)
	s.spans = make(map[string]*models.SpanMetadata)
	s.logs = make(map[string]*models.LogMetadata)
	s.services = make(map[string]struct{})

	return nil
}

// trackServices adds services to the global service set.
// Must be called with appropriate lock held.
func (s *Store) trackServices(services map[string]int64) {
	s.servicesmu.Lock()
	defer s.servicesmu.Unlock()

	for service := range services {
		s.services[service] = struct{}{}
	}
}
