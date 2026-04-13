// Package memory provides an in-memory storage implementation for metadata.
package memory

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/fidde/otlp_cardinality_checker/pkg/autotemplate"
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

	// Attributes storage: attribute key -> metadata
	attributes map[string]*models.AttributeMetadata
	attributesmu sync.RWMutex

	// Services tracks all service names seen
	services map[string]struct{}
	servicesmu sync.RWMutex

	// Deep watch: key -> watched attribute
	watched       map[string]*models.WatchedAttribute
	watchedmu     sync.RWMutex
	maxWatchedFields int

	// Autotemplate configuration
	useAutoTemplate bool
	autoTemplateCfg autotemplate.Config

	// Pod log enrichment
	podLogEnrichment    bool
	podLogServiceLabels []string
}

// NewWithConfig creates a store with all configuration options.
func NewWithConfig(useAutoTemplate bool, maxWatchedFields int, podLogEnrichment bool, podLogServiceLabels []string) *Store {
	if maxWatchedFields <= 0 {
		maxWatchedFields = 10
	}
	cfg := autotemplate.DefaultConfig()
	cfg.SimThreshold = 0.7 // Increased from 0.5 for stricter matching

	return &Store{
		metrics:             make(map[string]*models.MetricMetadata),
		spans:               make(map[string]*models.SpanMetadata),
		logs:                make(map[string]*models.LogMetadata),
		attributes:          make(map[string]*models.AttributeMetadata),
		services:            make(map[string]struct{}),
		watched:             make(map[string]*models.WatchedAttribute),
		maxWatchedFields:    maxWatchedFields,
		useAutoTemplate:     useAutoTemplate,
		autoTemplateCfg:     cfg,
		podLogEnrichment:    podLogEnrichment,
		podLogServiceLabels: podLogServiceLabels,
	}
}

// NewWithAutoTemplate creates a store with optional autotemplate support.
func NewWithAutoTemplate(useAutoTemplate bool, maxWatchedFields int) *Store {
	return NewWithConfig(useAutoTemplate, maxWatchedFields, false, nil)
}

// UseAutoTemplate returns whether autotemplate is enabled
func (s *Store) UseAutoTemplate() bool {
	return s.useAutoTemplate
}

// AutoTemplateCfg returns the autotemplate configuration
func (s *Store) AutoTemplateCfg() autotemplate.Config {
	return s.autoTemplateCfg
}

// PodLogEnrichment returns whether pod log enrichment is enabled.
func (s *Store) PodLogEnrichment() bool {
	return s.podLogEnrichment
}

// PodLogServiceLabels returns the ordered list of attribute keys for service name discovery.
func (s *Store) PodLogServiceLabels() []string {
	return s.podLogServiceLabels
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
		// Update sample count
		existing.SampleCount += span.SampleCount

		// Merge attribute keys (full HLL/samples/taint merge)
		for key, keyMeta := range span.AttributeKeys {
			if existingKey, exists := existing.AttributeKeys[key]; exists {
				models.MergeKeyMetadata(existingKey, keyMeta)
			} else {
				existing.AttributeKeys[key] = keyMeta
			}
		}

		// Merge resource keys (full HLL/samples/taint merge)
		for key, keyMeta := range span.ResourceKeys {
			if existingKey, exists := existing.ResourceKeys[key]; exists {
				models.MergeKeyMetadata(existingKey, keyMeta)
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

	// Extract service name from Services map (should have exactly one after analyzer change)
	var serviceName string
	for svc := range log.Services {
		serviceName = svc
		break
	}
	
	severity := log.Severity
	if severity == "" {
		severity = "UNSET"
	}
	
	// Use service|severity as key to match analyzer grouping
	key := serviceName + "|" + severity

	// If log exists, merge with existing
	if existing, exists := s.logs[key]; exists {
		// Update sample count
		existing.SampleCount += log.SampleCount

		// Merge attribute keys (full HLL/samples/taint merge)
		for key, keyMeta := range log.AttributeKeys {
			if existingKey, exists := existing.AttributeKeys[key]; exists {
				models.MergeKeyMetadata(existingKey, keyMeta)
			} else {
				existing.AttributeKeys[key] = keyMeta
			}
		}

		// Merge resource keys (full HLL/samples/taint merge)
		for key, keyMeta := range log.ResourceKeys {
			if existingKey, exists := existing.ResourceKeys[key]; exists {
				models.MergeKeyMetadata(existingKey, keyMeta)
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

	// Aggregate all service+severity combinations for this severity
	var aggregated *models.LogMetadata
	for _, log := range s.logs {
		if log.Severity == severityText {
			if aggregated == nil {
				// First match - clone it
				aggregated = &models.LogMetadata{
					Severity:       log.Severity,
					SeverityNumber: log.SeverityNumber,
					AttributeKeys:  make(map[string]*models.KeyMetadata),
					ResourceKeys:   make(map[string]*models.KeyMetadata),
					Services:       make(map[string]int64),
					EventNames:     []string{},
					BodyTemplates:  []*models.BodyTemplate{},
				}
			}
			
			// Merge counts
			aggregated.SampleCount += log.SampleCount
			
			// Merge services
			for svc, count := range log.Services {
				aggregated.Services[svc] += count
			}
			
			// Merge attribute keys (clone to avoid mutating stored data under RLock)
			for attrKey, keyMeta := range log.AttributeKeys {
				if existing, exists := aggregated.AttributeKeys[attrKey]; exists {
					existing.Count += keyMeta.Count
				} else {
					aggregated.AttributeKeys[attrKey] = &models.KeyMetadata{
						Count:                keyMeta.Count,
						Percentage:           keyMeta.Percentage,
						EstimatedCardinality: keyMeta.Cardinality(),
						ValueSamples:         keyMeta.GetSortedSamples(),
						HasInvalidUTF8:       keyMeta.HasInvalidUTF8,
					}
				}
			}
			
			// Merge resource keys (clone to avoid mutating stored data under RLock)
			for resKey, keyMeta := range log.ResourceKeys {
				if existing, exists := aggregated.ResourceKeys[resKey]; exists {
					existing.Count += keyMeta.Count
				} else {
					aggregated.ResourceKeys[resKey] = &models.KeyMetadata{
						Count:                keyMeta.Count,
						Percentage:           keyMeta.Percentage,
						EstimatedCardinality: keyMeta.Cardinality(),
						ValueSamples:         keyMeta.GetSortedSamples(),
						HasInvalidUTF8:       keyMeta.HasInvalidUTF8,
					}
				}
			}
			
			// Collect all body templates
			aggregated.BodyTemplates = append(aggregated.BodyTemplates, log.BodyTemplates...)
			
			// Merge other fields
			if log.HasTraceContext {
				aggregated.HasTraceContext = true
			}
			if log.HasSpanContext {
				aggregated.HasSpanContext = true
			}
			if aggregated.ScopeInfo == nil && log.ScopeInfo != nil {
				aggregated.ScopeInfo = log.ScopeInfo
			}
		}
	}
	
	if aggregated == nil {
		return nil, fmt.Errorf("log severity %s: %w", severityText, models.ErrNotFound)
	}

	// Sort body templates by count descending
	if len(aggregated.BodyTemplates) > 0 {
		sort.Slice(aggregated.BodyTemplates, func(i, j int) bool {
			return aggregated.BodyTemplates[i].Count > aggregated.BodyTemplates[j].Count
		})
	}

	return aggregated, nil
}

// GetLogByServiceAndSeverity retrieves log metadata for a specific service+severity combination.
func (s *Store) GetLogByServiceAndSeverity(ctx context.Context, serviceName, severityText string) (*models.LogMetadata, error) {
	s.logsmu.RLock()
	defer s.logsmu.RUnlock()

	if severityText == "" {
		severityText = "UNSET"
	}
	
	key := serviceName + "|" + severityText
	log, exists := s.logs[key]
	if !exists {
		return nil, fmt.Errorf("log service=%s severity=%s: %w", serviceName, severityText, models.ErrNotFound)
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
		
		// Sort body templates by count descending
		if len(log.BodyTemplates) > 0 {
			sort.Slice(log.BodyTemplates, func(i, j int) bool {
				return log.BodyTemplates[i].Count > log.BodyTemplates[j].Count
			})
		}
		
		logs = append(logs, log)
	}

	// Sort by severity for consistency
	sort.Slice(logs, func(i, j int) bool {
		return logs[i].Severity < logs[j].Severity
	})

	return logs, nil
}

// CountLogPatterns returns the number of unique log templates without building the full pattern response.
func (s *Store) CountLogPatterns(ctx context.Context) (int, error) {
	s.logsmu.RLock()
	defer s.logsmu.RUnlock()

	seen := make(map[string]struct{})
	for _, logMeta := range s.logs {
		for _, tmpl := range logMeta.BodyTemplates {
			seen[tmpl.Template] = struct{}{}
		}
	}
	return len(seen), nil
}

// GetLogPatterns returns an advanced pattern analysis view.
// Note: In-memory store has limited pattern analysis capabilities compared to SQLite.
func (s *Store) GetLogPatterns(ctx context.Context, minCount int64, minServices int) (*models.PatternExplorerResponse, error) {
	s.logsmu.RLock()
	defer s.logsmu.RUnlock()
	
	// Build pattern groups from in-memory data
	patternMap := make(map[string]*models.PatternGroup)
	
	for _, logMeta := range s.logs {
		// Get actual severity from metadata (map key is "service|severity")
		severity := logMeta.Severity
		
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
		
		// Add histogram bucket count to total keys
		if metric.Data != nil {
			if histMetric, ok := metric.Data.(*models.HistogramMetric); ok && len(histMetric.ExplicitBounds) > 0 {
				// Number of buckets = explicit bounds + 1 (infinity bucket)
				totalKeys += len(histMetric.ExplicitBounds) + 1
			} else if expHistMetric, ok := metric.Data.(*models.ExponentialHistogramMetric); ok && len(expHistMetric.Scales) > 0 {
				// For exponential histograms, add a fixed count per scale
				totalKeys += len(expHistMetric.Scales) * 10 // Approximate bucket count per scale
			}
		}
		
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
	s.attributesmu.Lock()
	s.servicesmu.Lock()
	s.watchedmu.Lock()
	defer s.metricsmu.Unlock()
	defer s.spansmu.Unlock()
	defer s.logsmu.Unlock()
	defer s.attributesmu.Unlock()
	defer s.servicesmu.Unlock()
	defer s.watchedmu.Unlock()

	s.metrics = make(map[string]*models.MetricMetadata)
	s.spans = make(map[string]*models.SpanMetadata)
	s.logs = make(map[string]*models.LogMetadata)
	s.attributes = make(map[string]*models.AttributeMetadata)
	s.services = make(map[string]struct{})
	s.watched = make(map[string]*models.WatchedAttribute)

	return nil
}

// StoreAttributeValue stores or updates an attribute key-value observation.
func (s *Store) StoreAttributeValue(ctx context.Context, key, value, signalType, scope string) error {
	if key == "" {
		return errors.New("attribute key cannot be empty")
	}

	// Fast path: key already exists — hold the read-lock only long enough to
	// retrieve the pointer, then release before the per-key AddValue (which
	// has its own mutex). This allows concurrent writes to different attribute
	// keys without a global write-lock stall.
	s.attributesmu.RLock()
	attr := s.attributes[key]
	s.attributesmu.RUnlock()

	if attr == nil {
		// Slow path: first time we see this key. Write-lock, re-check, create.
		s.attributesmu.Lock()
		attr = s.attributes[key]
		if attr == nil {
			attr = models.NewAttributeMetadata(key)
			s.attributes[key] = attr
		}
		s.attributesmu.Unlock()
	}

	// AddValue is safe to call without attributesmu: it uses its own lock.
	attr.AddValue(value, signalType, scope)

	// Deep watch: if this key is actively watched, record value frequency.
	s.watchedmu.RLock()
	w, watched := s.watched[key]
	s.watchedmu.RUnlock()
	if watched {
		w.AddValue(value)
	}

	return nil
}

// GetAttribute retrieves attribute metadata by key.
func (s *Store) GetAttribute(ctx context.Context, key string) (*models.AttributeMetadata, error) {
	s.attributesmu.RLock()
	defer s.attributesmu.RUnlock()

	attr, exists := s.attributes[key]
	if !exists {
		return nil, fmt.Errorf("attribute not found: %s", key)
	}

	return attr, nil
}

// ListAttributes lists all attributes with optional filtering.
func (s *Store) ListAttributes(ctx context.Context, filter *models.AttributeFilter) ([]*models.AttributeMetadata, error) {
	s.attributesmu.RLock()
	defer s.attributesmu.RUnlock()

	// Collect all attributes
	attrs := make([]*models.AttributeMetadata, 0, len(s.attributes))
	for _, attr := range s.attributes {
		// Apply filters
		if filter != nil {
			// Filter by signal type
			if filter.SignalType != "" {
				found := false
				for _, st := range attr.SignalTypes {
					if st == filter.SignalType {
						found = true
						break
					}
				}
				if !found {
					continue
				}
			}

			// Filter by scope
			if filter.Scope != "" && attr.Scope != filter.Scope && attr.Scope != "both" {
				continue
			}

			// Filter by cardinality range
			if filter.MinCardinality > 0 && attr.EstimatedCardinality < filter.MinCardinality {
				continue
			}
			if filter.MaxCardinality > 0 && attr.EstimatedCardinality > filter.MaxCardinality {
				continue
			}
		}

		attrs = append(attrs, attr)
	}

	// Sort results
	sortBy := "cardinality"
	sortOrder := "desc"
	if filter != nil {
		if filter.SortBy != "" {
			sortBy = filter.SortBy
		}
		if filter.SortOrder != "" {
			sortOrder = filter.SortOrder
		}
	}

	sort.Slice(attrs, func(i, j int) bool {
		var less bool
		switch sortBy {
		case "cardinality":
			less = attrs[i].EstimatedCardinality < attrs[j].EstimatedCardinality
		case "count":
			less = attrs[i].Count < attrs[j].Count
		case "key":
			less = strings.ToLower(attrs[i].Key) < strings.ToLower(attrs[j].Key)
		case "first_seen":
			less = attrs[i].FirstSeen.Before(attrs[j].FirstSeen)
		case "last_seen":
			less = attrs[i].LastSeen.Before(attrs[j].LastSeen)
		default:
			less = attrs[i].EstimatedCardinality < attrs[j].EstimatedCardinality
		}

		if sortOrder == "desc" {
			return !less
		}
		return less
	})

	// Apply pagination
	if filter != nil && (filter.Limit > 0 || filter.Offset > 0) {
		start := filter.Offset
		if start > len(attrs) {
			start = len(attrs)
		}

		end := len(attrs)
		if filter.Limit > 0 {
			end = start + filter.Limit
			if end > len(attrs) {
				end = len(attrs)
			}
		}

		attrs = attrs[start:end]
	}

	return attrs, nil
}

// WatchAttribute activates deep watch for an attribute key.
// Returns an error if the maximum number of watched fields is already reached.
// Idempotent: activating an already-watched key re-activates it without resetting its data.
func (s *Store) WatchAttribute(ctx context.Context, key string) error {
	if key == "" {
		return errors.New("attribute key cannot be empty")
	}

	s.watchedmu.Lock()
	defer s.watchedmu.Unlock()

	if existing, ok := s.watched[key]; ok {
		// Already tracked: just re-activate if it was inactive.
		existing.SetActive(true)
		return nil
	}

	if len(s.watched) >= s.maxWatchedFields {
		// Count only active watches toward the limit so that deactivated
		// entries (with preserved values) do not block re-activation.
		activeCount := 0
		for _, w := range s.watched {
			_, _, _, _, active, _, _ := w.Snapshot()
			if active {
				activeCount++
			}
		}
		if activeCount >= s.maxWatchedFields {
			return fmt.Errorf("maximum watched fields limit (%d) reached", s.maxWatchedFields)
		}
	}

	s.watched[key] = models.NewWatchedAttribute(key, 10000)
	return nil
}

// UnwatchAttribute deactivates deep watch for the key, preserving all
// collected values so they remain visible in the Value Explorer.
func (s *Store) UnwatchAttribute(ctx context.Context, key string) error {
	if key == "" {
		return errors.New("attribute key cannot be empty")
	}

	s.watchedmu.Lock()
	defer s.watchedmu.Unlock()

	existing, ok := s.watched[key]
	if !ok {
		return fmt.Errorf("attribute %q: %w", key, models.ErrNotFound)
	}

	existing.SetActive(false)
	return nil
}

// GetWatchedAttribute returns the WatchedAttribute for a specific key.
func (s *Store) GetWatchedAttribute(ctx context.Context, key string) (*models.WatchedAttribute, error) {
	if key == "" {
		return nil, errors.New("attribute key cannot be empty")
	}

	s.watchedmu.RLock()
	w, ok := s.watched[key]
	s.watchedmu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("attribute %q: %w", key, models.ErrNotFound)
	}

	return w, nil
}

// ListWatchedAttributes returns all currently watched attributes.
func (s *Store) ListWatchedAttributes(ctx context.Context) ([]*models.WatchedAttribute, error) {
	s.watchedmu.RLock()
	defer s.watchedmu.RUnlock()

	result := make([]*models.WatchedAttribute, 0, len(s.watched))
	for _, w := range s.watched {
		result = append(result, w)
	}
	return result, nil
}

// MergeWatchedAttribute inserts or merges a WatchedAttribute from a session restore.
// The restored entry is always set to Active=false (read-only historical data).
// Implements api.StoreAccessor extended interface.
func (s *Store) MergeWatchedAttribute(ctx context.Context, w *models.WatchedAttribute) error {
	if w == nil {
		return nil
	}
	w.SetActive(false)

	s.watchedmu.Lock()
	defer s.watchedmu.Unlock()

	s.watched[w.Key] = w
	return nil
}

// GetWatchedAll returns all watched attributes for session saving.
// Implements api.StoreAccessor extended interface.
func (s *Store) GetWatchedAll(ctx context.Context) ([]*models.WatchedAttribute, error) {
	return s.ListWatchedAttributes(ctx)
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

// spanKindToString converts a span kind int32 to a human-readable string.
func spanKindToString(kind int32) string {
	switch kind {
	case 0:
		return "Unspecified"
	case 1:
		return "Internal"
	case 2:
		return "Server"
	case 3:
		return "Client"
	case 4:
		return "Producer"
	case 5:
		return "Consumer"
	default:
		return "Unknown"
	}
}

// Close cleans up resources (no-op for in-memory store).
func (s *Store) Close() error {
	return nil
}

// GetSpanPatterns aggregates span names into patterns globally.
// Returns patterns with matching spans grouped together.
func (s *Store) GetSpanPatterns(ctx context.Context) (*models.SpanPatternResponse, error) {
	s.spansmu.RLock()
	defer s.spansmu.RUnlock()

	// Group spans by their extracted patterns
	patternMap := make(map[string]*models.SpanPatternGroup)

	for _, span := range s.spans {
		// Get the pattern template for this span name
		var pattern string
		if len(span.NamePatterns) > 0 {
			pattern = span.NamePatterns[0].Template
		} else {
			// If no pattern was extracted, the span name itself is the pattern
			pattern = span.Name
		}

		// Get or create pattern group
		pg, exists := patternMap[pattern]
		if !exists {
			pg = &models.SpanPatternGroup{
				Pattern:       pattern,
				MatchingSpans: []models.SpanPatternMatch{},
				TotalSamples:  0,
				SpanCount:     0,
			}
			patternMap[pattern] = pg
		}

		// Extract service names
		services := make([]string, 0, len(span.Services))
		for svc := range span.Services {
			services = append(services, svc)
		}

		// Convert span kind to string
		kindStr := spanKindToString(span.Kind)

		// Add this span to the pattern group
		pg.MatchingSpans = append(pg.MatchingSpans, models.SpanPatternMatch{
			SpanName:    span.Name,
			SampleCount: span.SampleCount,
			Services:    services,
			Kind:        kindStr,
		})
		pg.TotalSamples += span.SampleCount
		pg.SpanCount++
	}

	// Convert to slice and sort by span count (most matches first)
	patterns := make([]models.SpanPatternGroup, 0, len(patternMap))
	for _, pg := range patternMap {
		// Sort matching spans by sample count
		sort.Slice(pg.MatchingSpans, func(i, j int) bool {
			return pg.MatchingSpans[i].SampleCount > pg.MatchingSpans[j].SampleCount
		})
		patterns = append(patterns, *pg)
	}

	// Sort patterns: multi-span patterns first, then by total samples
	sort.Slice(patterns, func(i, j int) bool {
		if patterns[i].SpanCount != patterns[j].SpanCount {
			return patterns[i].SpanCount > patterns[j].SpanCount
		}
		return patterns[i].TotalSamples > patterns[j].TotalSamples
	})

	return &models.SpanPatternResponse{
		Patterns: patterns,
		Total:    len(patterns),
	}, nil
}

// GetAll returns all metadata from the store for session saving.
// Implements api.StoreAccessor interface.
func (s *Store) GetAll(ctx context.Context) (
	metrics []*models.MetricMetadata,
	spans []*models.SpanMetadata,
	logs []*models.LogMetadata,
	attrs []*models.AttributeMetadata,
	services []string,
	err error,
) {
	// Get metrics
	s.metricsmu.RLock()
	metrics = make([]*models.MetricMetadata, 0, len(s.metrics))
	for _, m := range s.metrics {
		metrics = append(metrics, m)
	}
	s.metricsmu.RUnlock()

	// Get spans
	s.spansmu.RLock()
	spans = make([]*models.SpanMetadata, 0, len(s.spans))
	for _, sp := range s.spans {
		spans = append(spans, sp)
	}
	s.spansmu.RUnlock()

	// Get logs
	s.logsmu.RLock()
	logs = make([]*models.LogMetadata, 0, len(s.logs))
	for _, l := range s.logs {
		logs = append(logs, l)
	}
	s.logsmu.RUnlock()

	// Get attributes
	s.attributesmu.RLock()
	attrs = make([]*models.AttributeMetadata, 0, len(s.attributes))
	for _, a := range s.attributes {
		attrs = append(attrs, a)
	}
	s.attributesmu.RUnlock()

	// Get services
	s.servicesmu.RLock()
	services = make([]string, 0, len(s.services))
	for svc := range s.services {
		services = append(services, svc)
	}
	s.servicesmu.RUnlock()

	return metrics, spans, logs, attrs, services, nil
}

// MergeMetric merges a metric into the store.
// Implements api.StoreAccessor interface.
func (s *Store) MergeMetric(ctx context.Context, metric *models.MetricMetadata) error {
	return s.StoreMetric(ctx, metric)
}

// MergeSpan merges a span into the store.
// Implements api.StoreAccessor interface.
func (s *Store) MergeSpan(ctx context.Context, span *models.SpanMetadata) error {
	return s.StoreSpan(ctx, span)
}

// MergeLog merges a log into the store.
// Implements api.StoreAccessor interface.
func (s *Store) MergeLog(ctx context.Context, log *models.LogMetadata) error {
	return s.StoreLog(ctx, log)
}

// MergeAttribute merges an attribute into the store.
// Implements api.StoreAccessor interface.
func (s *Store) MergeAttribute(ctx context.Context, attr *models.AttributeMetadata) error {
	if attr == nil {
		return nil
	}

	s.attributesmu.Lock()
	defer s.attributesmu.Unlock()

	existing, exists := s.attributes[attr.Key]
	if !exists {
		s.attributes[attr.Key] = attr
		return nil
	}

	// Merge the attribute using existing function
	models.MergeAttributeMetadata(existing, attr)
	return nil
}
