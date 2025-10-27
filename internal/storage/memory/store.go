// Package memory provides an in-memory storage implementation for metadata.
package memory

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/fidde/otlp_cardinality_checker/internal/analyzer/autotemplate"
	"github.com/fidde/otlp_cardinality_checker/pkg/models"
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
	
	// Autotemplate configuration
	useAutoTemplate bool
	autoTemplateCfg autotemplate.Config
}

// New creates a new in-memory store.
func New() *Store {
	return NewWithAutoTemplate(false)
}

// NewWithAutoTemplate creates a store with optional autotemplate support.
func NewWithAutoTemplate(useAutoTemplate bool) *Store {
	cfg := autotemplate.DefaultConfig()
	cfg.Shards = 4
	cfg.SimThreshold = 0.7 // Increased from 0.5 for stricter matching
	
	return &Store{
		metrics:         make(map[string]*models.MetricMetadata),
		spans:           make(map[string]*models.SpanMetadata),
		logs:            make(map[string]*models.LogMetadata),
		services:        make(map[string]struct{}),
		useAutoTemplate: useAutoTemplate,
		autoTemplateCfg: cfg,
	}
}

// UseAutoTemplate returns whether autotemplate is enabled
func (s *Store) UseAutoTemplate() bool {
	return s.useAutoTemplate
}

// AutoTemplateCfg returns the autotemplate configuration
func (s *Store) AutoTemplateCfg() autotemplate.Config {
	return s.autoTemplateCfg
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
		return nil, fmt.Errorf("metric %s: %w", name, models.ErrNotFound)
	}

	return metric, nil
}

// ListMetrics returns metrics with pagination, optionally filtered by service name.
// Returns metrics, total count, and error.
func (s *Store) ListMetrics(ctx context.Context, serviceName string, limit, offset int) ([]*models.MetricMetadata, int, error) {
	s.metricsmu.RLock()
	defer s.metricsmu.RUnlock()

	// Build full list first (for filtering)
	allMetrics := make([]*models.MetricMetadata, 0, len(s.metrics))
	for _, metric := range s.metrics {
		// Filter by service if specified
		if serviceName != "" {
			if _, hasService := metric.Services[serviceName]; !hasService {
				continue
			}
		}
		allMetrics = append(allMetrics, metric)
	}

	// Sort by name for consistency
	sort.Slice(allMetrics, func(i, j int) bool {
		return allMetrics[i].Name < allMetrics[j].Name
	})
	
	total := len(allMetrics)
	
	// Apply pagination
	start := offset
	if start >= total {
		return []*models.MetricMetadata{}, total, nil
	}
	
	end := start + limit
	if end > total {
		end = total
	}

	return allMetrics[start:end], total, nil
}

// CountMetrics returns the total number of metrics, optionally filtered by service.
func (s *Store) CountMetrics(ctx context.Context, serviceName string) (int, error) {
	s.metricsmu.RLock()
	defer s.metricsmu.RUnlock()

	if serviceName == "" {
		return len(s.metrics), nil
	}
	
	count := 0
	for _, metric := range s.metrics {
		if _, hasService := metric.Services[serviceName]; hasService {
			count++
		}
	}
	
	return count, nil
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
		// Update sample count
		existing.SampleCount += span.SampleCount

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
		return nil, fmt.Errorf("span %s: %w", name, models.ErrNotFound)
	}

	return span, nil
}

// ListSpans returns all spans, optionally filtered by service name.
func (s *Store) ListSpans(ctx context.Context, serviceName string, limit, offset int) ([]*models.SpanMetadata, int, error) {
	s.spansmu.RLock()
	defer s.spansmu.RUnlock()

	// Build full list first (for filtering)
	allSpans := make([]*models.SpanMetadata, 0, len(s.spans))
	for _, span := range s.spans {
		// Filter by service if specified
		if serviceName != "" {
			if _, hasService := span.Services[serviceName]; !hasService {
				continue
			}
		}
		allSpans = append(allSpans, span)
	}

	// Sort by name for consistency
	sort.Slice(allSpans, func(i, j int) bool {
		return allSpans[i].Name < allSpans[j].Name
	})
	
	total := len(allSpans)
	
	// Apply pagination
	start := offset
	if start >= total {
		return []*models.SpanMetadata{}, total, nil
	}
	
	end := start + limit
	if end > total {
		end = total
	}

	return allSpans[start:end], total, nil
}

// CountSpans returns the total number of spans, optionally filtered by service.
func (s *Store) CountSpans(ctx context.Context, serviceName string) (int, error) {
	s.spansmu.RLock()
	defer s.spansmu.RUnlock()

	if serviceName == "" {
		return len(s.spans), nil
	}
	
	count := 0
	for _, span := range s.spans {
		if _, hasService := span.Services[serviceName]; hasService {
			count++
		}
	}
	
	return count, nil
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

	key := log.Severity
	if key == "" {
		key = "UNSET"
	}

	// If log exists, merge with existing
	if existing, exists := s.logs[key]; exists {
		// Update sample count
		existing.SampleCount += log.SampleCount

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

		// Update body templates (replace, not merge, since analyzer has full state)
		if len(log.BodyTemplates) > 0 {
			existing.BodyTemplates = log.BodyTemplates
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
		return nil, fmt.Errorf("log severity %s: %w", severityText, models.ErrNotFound)
	}

	// Sort body templates by count descending
	if len(log.BodyTemplates) > 0 {
		sort.Slice(log.BodyTemplates, func(i, j int) bool {
			return log.BodyTemplates[i].Count > log.BodyTemplates[j].Count
		})
	}

	return log, nil
}

// ListLogs returns all log metadata, optionally filtered by service name.
func (s *Store) ListLogs(ctx context.Context, serviceName string, limit, offset int) ([]*models.LogMetadata, int, error) {
	s.logsmu.RLock()
	defer s.logsmu.RUnlock()

	// Build full list first (for filtering)
	allLogs := make([]*models.LogMetadata, 0, len(s.logs))
	for _, log := range s.logs {
		// Filter by service if specified
		if serviceName != "" {
			if _, hasService := log.Services[serviceName]; !hasService {
				continue
			}
		}
		
		// Sort body templates by count descending
		if len(log.BodyTemplates) > 0 {
			sort.Slice(log.BodyTemplates, func(i, j int) bool {
				return log.BodyTemplates[i].Count > log.BodyTemplates[j].Count
			})
		}
		
		allLogs = append(allLogs, log)
	}

	// Sort by severity for consistency
	sort.Slice(allLogs, func(i, j int) bool {
		return allLogs[i].Severity < allLogs[j].Severity
	})
	
	total := len(allLogs)
	
	// Apply pagination
	start := offset
	if start >= total {
		return []*models.LogMetadata{}, total, nil
	}
	
	end := start + limit
	if end > total {
		end = total
	}

	return allLogs[start:end], total, nil
}

// CountLogs returns the total number of log severities, optionally filtered by service.
func (s *Store) CountLogs(ctx context.Context, serviceName string) (int, error) {
	s.logsmu.RLock()
	defer s.logsmu.RUnlock()

	if serviceName == "" {
		return len(s.logs), nil
	}
	
	count := 0
	for _, log := range s.logs {
		if _, hasService := log.Services[serviceName]; hasService {
			count++
		}
	}
	
	return count, nil
}

// GetLogPatterns returns an advanced pattern analysis view.
// Note: In-memory store has limited pattern analysis capabilities compared to SQLite.
func (s *Store) GetLogPatterns(ctx context.Context, minCount int64, minServices int) (*models.PatternExplorerResponse, error) {
	s.logsmu.RLock()
	defer s.logsmu.RUnlock()
	
	// Build pattern groups from in-memory data
	patternMap := make(map[string]*models.PatternGroup)
	
	for severity, logMeta := range s.logs {
		for _, template := range logMeta.BodyTemplates {
			// Apply count filter
			if template.Count < minCount {
				continue
			}
			
			// Initialize pattern group if needed
			if _, exists := patternMap[template.Template]; !exists {
				patternMap[template.Template] = &models.PatternGroup{
					Template:          template.Template,
					ExampleBody:       template.Example,
					TotalCount:        0,
					SeverityBreakdown: make(map[string]int64),
					Services:          []models.ServicePatternInfo{},
				}
			}
			
			pg := patternMap[template.Template]
			pg.TotalCount += template.Count
			pg.SeverityBreakdown[severity] += template.Count
			
			// Build service info from log metadata services
			for serviceName, sampleCount := range logMeta.Services {
				if serviceName == "" {
					serviceName = "unknown"
				}
				
				// Convert resource keys
				resourceKeys := make([]models.KeyInfo, 0, len(logMeta.ResourceKeys))
				for keyName, keyMeta := range logMeta.ResourceKeys {
					resourceKeys = append(resourceKeys, models.KeyInfo{
						Name:         keyName,
						Cardinality:  int(keyMeta.EstimatedCardinality),
						SampleValues: keyMeta.ValueSamples,
					})
				}
				
				// Convert attribute keys
				attrKeys := make([]models.KeyInfo, 0, len(logMeta.AttributeKeys))
				for keyName, keyMeta := range logMeta.AttributeKeys {
					attrKeys = append(attrKeys, models.KeyInfo{
						Name:         keyName,
						Cardinality:  int(keyMeta.EstimatedCardinality),
						SampleValues: keyMeta.ValueSamples,
					})
				}
				
				pg.Services = append(pg.Services, models.ServicePatternInfo{
					ServiceName:   serviceName,
					SampleCount:   sampleCount,
					Severities:    []string{severity},
					ResourceKeys:  resourceKeys,
					AttributeKeys: attrKeys,
				})
			}
		}
	}
	
	// Filter by minServices and convert to slice
	var patterns []models.PatternGroup
	for _, pg := range patternMap {
		if len(pg.Services) >= minServices {
			patterns = append(patterns, *pg)
		}
	}
	
	// Sort by total count descending
	sort.Slice(patterns, func(i, j int) bool {
		return patterns[i].TotalCount > patterns[j].TotalCount
	})
	
	return &models.PatternExplorerResponse{
		Patterns: patterns,
		Total:    len(patterns),
	}, nil
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

// GetServiceOverview returns a complete overview of all telemetry for a given service.
func (s *Store) GetServiceOverview(ctx context.Context, serviceName string) (*models.ServiceOverview, error) {
	if serviceName == "" {
		return nil, errors.New("service name cannot be empty")
	}

	// Get all without pagination for service overview (small result set per service)
	const noLimit = 10000 // High enough for service overview
	
	metrics, _, err := s.ListMetrics(ctx, serviceName, noLimit, 0)
	if err != nil {
		return nil, fmt.Errorf("listing metrics: %w", err)
	}

	spans, _, err := s.ListSpans(ctx, serviceName, noLimit, 0)
	if err != nil {
		return nil, fmt.Errorf("listing spans: %w", err)
	}

	logs, _, err := s.ListLogs(ctx, serviceName, noLimit, 0)
	if err != nil {
		return nil, fmt.Errorf("listing logs: %w", err)
	}

	return &models.ServiceOverview{
		ServiceName: serviceName,
		MetricCount: len(metrics),
		SpanCount:   len(spans),
		LogCount:    len(logs),
		Metrics:     metrics,
		Spans:       spans,
		Logs:        logs,
	}, nil
}

// GetHighCardinalityKeys returns high-cardinality keys across all signal types.
// For in-memory store, we aggregate keys from metrics, spans, and logs.
func (s *Store) GetHighCardinalityKeys(ctx context.Context, threshold int, limit int) (*models.CrossSignalCardinalityResponse, error) {
	if limit <= 0 {
		limit = 100
	}

	threshold64 := int64(threshold)
	var allKeys []models.SignalKey

	// Collect metric keys
	s.metricsmu.RLock()
	for metricName, metric := range s.metrics {
		for keyName, keyMeta := range metric.LabelKeys {
			if keyMeta.EstimatedCardinality >= threshold64 {
				allKeys = append(allKeys, models.SignalKey{
					SignalType:          "metric",
					SignalName:          metricName,
					KeyScope:            "label",
					KeyName:             keyName,
					EstimatedCardinality: int(keyMeta.EstimatedCardinality),
					KeyCount:            keyMeta.Count,
					ValueSamples:        keyMeta.ValueSamples,
				})
			}
		}
		for keyName, keyMeta := range metric.ResourceKeys {
			if keyMeta.EstimatedCardinality >= threshold64 {
				allKeys = append(allKeys, models.SignalKey{
					SignalType:          "metric",
					SignalName:          metricName,
					KeyScope:            "resource",
					KeyName:             keyName,
					EstimatedCardinality: int(keyMeta.EstimatedCardinality),
					KeyCount:            keyMeta.Count,
					ValueSamples:        keyMeta.ValueSamples,
				})
			}
		}
	}
	s.metricsmu.RUnlock()

	// Collect span keys
	s.spansmu.RLock()
	for spanName, span := range s.spans {
		for keyName, keyMeta := range span.AttributeKeys {
			if keyMeta.EstimatedCardinality >= threshold64 {
				allKeys = append(allKeys, models.SignalKey{
					SignalType:          "span",
					SignalName:          spanName,
					KeyScope:            "attribute",
					KeyName:             keyName,
					EstimatedCardinality: int(keyMeta.EstimatedCardinality),
					KeyCount:            keyMeta.Count,
					ValueSamples:        keyMeta.ValueSamples,
				})
			}
		}
		for keyName, keyMeta := range span.ResourceKeys {
			if keyMeta.EstimatedCardinality >= threshold64 {
				allKeys = append(allKeys, models.SignalKey{
					SignalType:          "span",
					SignalName:          spanName,
					KeyScope:            "resource",
					KeyName:             keyName,
					EstimatedCardinality: int(keyMeta.EstimatedCardinality),
					KeyCount:            keyMeta.Count,
					ValueSamples:        keyMeta.ValueSamples,
				})
			}
		}
	}
	s.spansmu.RUnlock()

	// Collect log keys
	s.logsmu.RLock()
	for severity, log := range s.logs {
		for keyName, keyMeta := range log.AttributeKeys {
			if keyMeta.EstimatedCardinality >= threshold64 {
				allKeys = append(allKeys, models.SignalKey{
					SignalType:          "log",
					SignalName:          severity,
					KeyScope:            "attribute",
					KeyName:             keyName,
					EstimatedCardinality: int(keyMeta.EstimatedCardinality),
					KeyCount:            keyMeta.Count,
					ValueSamples:        keyMeta.ValueSamples,
				})
			}
		}
		for keyName, keyMeta := range log.ResourceKeys {
			if keyMeta.EstimatedCardinality >= threshold64 {
				allKeys = append(allKeys, models.SignalKey{
					SignalType:          "log",
					SignalName:          severity,
					KeyScope:            "resource",
					KeyName:             keyName,
					EstimatedCardinality: int(keyMeta.EstimatedCardinality),
					KeyCount:            keyMeta.Count,
					ValueSamples:        keyMeta.ValueSamples,
				})
			}
		}
	}
	s.logsmu.RUnlock()

	// Sort by cardinality descending
	sort.Slice(allKeys, func(i, j int) bool {
		return allKeys[i].EstimatedCardinality > allKeys[j].EstimatedCardinality
	})

	// Apply limit
	if len(allKeys) > limit {
		allKeys = allKeys[:limit]
	}

	return &models.CrossSignalCardinalityResponse{
		HighCardinalityKeys: allKeys,
		Total:               len(allKeys),
		Threshold:           threshold,
	}, nil
}

// GetMetadataComplexity returns signals with high metadata complexity (many keys).
func (s *Store) GetMetadataComplexity(ctx context.Context, threshold int, limit int) (*models.MetadataComplexityResponse, error) {
	if limit <= 0 {
		limit = 100
	}

	var signals []models.SignalComplexity

	// Analyze metrics
	s.metricsmu.RLock()
	for metricName, metric := range s.metrics {
		totalKeys := len(metric.LabelKeys) + len(metric.ResourceKeys)
		if totalKeys < threshold {
			continue
		}

		sig := models.SignalComplexity{
			SignalType:        "metric",
			SignalName:        metricName,
			TotalKeys:         totalKeys,
			AttributeKeyCount: len(metric.LabelKeys),
			ResourceKeyCount:  len(metric.ResourceKeys),
		}

		// Find max cardinality and count high-cardinality keys
		for _, keyMeta := range metric.LabelKeys {
			if int(keyMeta.EstimatedCardinality) > sig.MaxCardinality {
				sig.MaxCardinality = int(keyMeta.EstimatedCardinality)
			}
			if keyMeta.EstimatedCardinality > 100 {
				sig.HighCardinalityCount++
			}
		}
		for _, keyMeta := range metric.ResourceKeys {
			if int(keyMeta.EstimatedCardinality) > sig.MaxCardinality {
				sig.MaxCardinality = int(keyMeta.EstimatedCardinality)
			}
			if keyMeta.EstimatedCardinality > 100 {
				sig.HighCardinalityCount++
			}
		}

		sig.ComplexityScore = sig.TotalKeys * sig.MaxCardinality
		signals = append(signals, sig)
	}
	s.metricsmu.RUnlock()

	// Analyze spans
	s.spansmu.RLock()
	for spanName, span := range s.spans {
		totalKeys := len(span.AttributeKeys) + len(span.ResourceKeys) + len(span.LinkAttributeKeys)
		
		// Count event keys
		eventKeys := 0
		for _, eventAttrs := range span.EventAttributeKeys {
			eventKeys += len(eventAttrs)
		}
		totalKeys += eventKeys

		if totalKeys < threshold {
			continue
		}

		sig := models.SignalComplexity{
			SignalType:        "span",
			SignalName:        spanName,
			TotalKeys:         totalKeys,
			AttributeKeyCount: len(span.AttributeKeys),
			ResourceKeyCount:  len(span.ResourceKeys),
			EventKeyCount:     eventKeys,
			LinkKeyCount:      len(span.LinkAttributeKeys),
		}

		// Find max cardinality
		for _, keyMeta := range span.AttributeKeys {
			if int(keyMeta.EstimatedCardinality) > sig.MaxCardinality {
				sig.MaxCardinality = int(keyMeta.EstimatedCardinality)
			}
			if keyMeta.EstimatedCardinality > 100 {
				sig.HighCardinalityCount++
			}
		}
		for _, keyMeta := range span.ResourceKeys {
			if int(keyMeta.EstimatedCardinality) > sig.MaxCardinality {
				sig.MaxCardinality = int(keyMeta.EstimatedCardinality)
			}
			if keyMeta.EstimatedCardinality > 100 {
				sig.HighCardinalityCount++
			}
		}
		for _, keyMeta := range span.LinkAttributeKeys {
			if int(keyMeta.EstimatedCardinality) > sig.MaxCardinality {
				sig.MaxCardinality = int(keyMeta.EstimatedCardinality)
			}
			if keyMeta.EstimatedCardinality > 100 {
				sig.HighCardinalityCount++
			}
		}

		sig.ComplexityScore = sig.TotalKeys * sig.MaxCardinality
		signals = append(signals, sig)
	}
	s.spansmu.RUnlock()

	// Analyze logs
	s.logsmu.RLock()
	for severity, log := range s.logs {
		totalKeys := len(log.AttributeKeys) + len(log.ResourceKeys)
		if totalKeys < threshold {
			continue
		}

		sig := models.SignalComplexity{
			SignalType:        "log",
			SignalName:        severity,
			TotalKeys:         totalKeys,
			AttributeKeyCount: len(log.AttributeKeys),
			ResourceKeyCount:  len(log.ResourceKeys),
		}

		// Find max cardinality
		for _, keyMeta := range log.AttributeKeys {
			if int(keyMeta.EstimatedCardinality) > sig.MaxCardinality {
				sig.MaxCardinality = int(keyMeta.EstimatedCardinality)
			}
			if keyMeta.EstimatedCardinality > 100 {
				sig.HighCardinalityCount++
			}
		}
		for _, keyMeta := range log.ResourceKeys {
			if int(keyMeta.EstimatedCardinality) > sig.MaxCardinality {
				sig.MaxCardinality = int(keyMeta.EstimatedCardinality)
			}
			if keyMeta.EstimatedCardinality > 100 {
				sig.HighCardinalityCount++
			}
		}

		sig.ComplexityScore = sig.TotalKeys * sig.MaxCardinality
		signals = append(signals, sig)
	}
	s.logsmu.RUnlock()

	// Sort by complexity score descending
	sort.Slice(signals, func(i, j int) bool {
		if signals[i].TotalKeys == signals[j].TotalKeys {
			return signals[i].MaxCardinality > signals[j].MaxCardinality
		}
		return signals[i].TotalKeys > signals[j].TotalKeys
	})

	// Apply limit
	if len(signals) > limit {
		signals = signals[:limit]
	}

	return &models.MetadataComplexityResponse{
		Signals:   signals,
		Total:     len(signals),
		Threshold: threshold,
	}, nil
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

// Close cleans up resources (no-op for in-memory store).
func (s *Store) Close() error {
	return nil
}
