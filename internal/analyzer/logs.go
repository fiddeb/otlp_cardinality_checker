package analyzer

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/fidde/otlp_cardinality_checker/internal/patterns"
	"github.com/fidde/otlp_cardinality_checker/pkg/autotemplate"
	"github.com/fidde/otlp_cardinality_checker/pkg/models"
	collogspb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
)

// LogBodyAnalyzerInterface defines the interface for log body analyzers
type LogBodyAnalyzerInterface interface {
	AddMessage(body string)
	GetTemplates() []*LogTemplate
}

// templateSyncInterval is how often GetTemplates() is called per key on the
// ingestion path. Keeping templates slightly stale (10s) eliminates the
// tokensToString / slice-build overhead that otherwise fires on every batch.
const templateSyncInterval = 10 * time.Second

// LogsAnalyzer extracts metadata from OTLP logs.
type LogsAnalyzer struct {
	mu                  sync.RWMutex                        // Protects bodyAnalyzers map
	bodyAnalyzers       map[string]LogBodyAnalyzerInterface // One analyzer per service+severity combination
	useAutoTemplate     bool                                // Whether to use autotemplate
	autoTemplateCfg     autotemplate.Config                 // Config for autotemplate
	patterns            []patterns.CompiledPattern          // Pre-masking patterns
	catalog             AttributeCatalog                    // Attribute catalog for global tracking
	podLogEnrichment    bool                                // Enrichment for pod logs
	podLogServiceLabels []string                            // Ordered label priority list

	lastTemplateSyncMu sync.Mutex
	lastTemplateSync   map[string]time.Time // key -> last time GetTemplates was called
}

// SetPodLogEnrichment configures pod log enrichment on an existing analyzer.
func (a *LogsAnalyzer) SetPodLogEnrichment(enabled bool, labels []string) {
	a.podLogEnrichment = enabled
	a.podLogServiceLabels = labels
}

// inferSeverityFromBody scans a log body for level keywords and returns a
// normalised severity string. Returns "UNSET" when no keyword is recognised.
// Patterns are evaluated in priority order: ERROR > WARN > INFO > DEBUG.
func inferSeverityFromBody(body string) string {
	lower := strings.ToLower(body)
	switch {
	case strings.Contains(lower, "error"):
		return "ERROR"
	case strings.Contains(lower, "warn"):
		return "WARN"
	case strings.Contains(lower, "info"):
		return "INFO"
	case strings.Contains(lower, "debug"):
		return "DEBUG"
	default:
		return "UNSET"
	}
}

// NewLogsAnalyzerWithCatalog creates a logs analyzer with attribute catalog.
func NewLogsAnalyzerWithCatalog(catalog AttributeCatalog) *LogsAnalyzer {
	return &LogsAnalyzer{
		bodyAnalyzers:    make(map[string]LogBodyAnalyzerInterface),
		useAutoTemplate:  false,
		catalog:          catalog,
		lastTemplateSync: make(map[string]time.Time),
	}
}

// NewLogsAnalyzerWithAutoTemplateAndCatalog creates a logs analyzer with autotemplate and catalog.
func NewLogsAnalyzerWithAutoTemplateAndCatalog(cfg autotemplate.Config, pats []patterns.CompiledPattern, catalog AttributeCatalog) *LogsAnalyzer {
	return &LogsAnalyzer{
		bodyAnalyzers:    make(map[string]LogBodyAnalyzerInterface),
		useAutoTemplate:  true,
		autoTemplateCfg:  cfg,
		patterns:         pats,
		catalog:          catalog,
		lastTemplateSync: make(map[string]time.Time),
	}
}

// createBodyAnalyzer creates the appropriate analyzer type
func (a *LogsAnalyzer) createBodyAnalyzer() LogBodyAnalyzerInterface {
	if a.useAutoTemplate {
		return NewAutoLogBodyAnalyzerWithPatterns(a.autoTemplateCfg, a.patterns)
	}
	return NewLogBodyAnalyzerWithPatterns(a.patterns)
}

// Analyze extracts metadata from an OTLP logs export request.
func (a *LogsAnalyzer) Analyze(req *collogspb.ExportLogsServiceRequest) ([]*models.LogMetadata, error) {
	return a.AnalyzeWithContext(context.Background(), req)
}

// AnalyzeWithContext extracts metadata with context for attribute catalog.
func (a *LogsAnalyzer) AnalyzeWithContext(ctx context.Context, req *collogspb.ExportLogsServiceRequest) ([]*models.LogMetadata, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	// Batch catalog deduplicates writes: each unique (signalType,scope,key,value)
	// tuple is forwarded to the real catalog exactly once per request. This
	// collapses O(records×attrs) lock acquisitions to O(unique tuples).
	batch := newBatchCatalog(a.catalog)

	// Map: service+severity -> LogMetadata
	// Key format: "service|severity"
	logMap := make(map[string]*models.LogMetadata)
	
	// Track which services have which severities for body template processing
	serviceSeverities := make(map[string]map[string]bool) // service -> {severity -> true}

	for _, resourceLogs := range req.ResourceLogs {
		// Extract resource attributes
		resourceAttrs := extractAttributes(resourceLogs.Resource.GetAttributes())

		var serviceName string
		if a.podLogEnrichment {
			serviceName = getServiceName(resourceAttrs, a.podLogServiceLabels...)
		} else {
			serviceName = getServiceName(resourceAttrs)
		}

		// Feed resource attributes to catalog
		extractAttributesToCatalog(ctx, batch, resourceAttrs, "log", "resource")

		for _, scopeLogs := range resourceLogs.ScopeLogs {
			scopeInfo := &models.ScopeMetadata{
				Name:    scopeLogs.Scope.GetName(),
				Version: scopeLogs.Scope.GetVersion(),
			}

			for _, logRecord := range scopeLogs.LogRecords {
				severityText := logRecord.SeverityText
				if severityText == "" {
					if a.podLogEnrichment {
						body := logRecord.GetBody().GetStringValue()
						severityText = inferSeverityFromBody(body)
					} else {
						severityText = "UNSET"
					}
				}

				// Create unique key per service+severity
				key := serviceName + "|" + severityText

				if _, exists := logMap[key]; !exists {
					logMap[key] = models.NewLogMetadata(severityText)
					logMap[key].ScopeInfo = scopeInfo
					logMap[key].Services[serviceName] = 0
					logMap[key].SeverityNumber = int32(logRecord.SeverityNumber)

					// Add resource keys for this service
					for resKey := range resourceAttrs {
						if logMap[key].ResourceKeys[resKey] == nil {
							logMap[key].ResourceKeys[resKey] = models.NewKeyMetadata()
						}
					}
				}

				metadata := logMap[key]
				metadata.SampleCount++
				metadata.Services[serviceName]++
				
				// Track trace/span context presence
				if len(logRecord.TraceId) > 0 && !isEmptyBytes(logRecord.TraceId) {
					metadata.HasTraceContext = true
				}
				if len(logRecord.SpanId) > 0 && !isEmptyBytes(logRecord.SpanId) {
					metadata.HasSpanContext = true
				}
				
				// Track dropped attributes statistics
				if logRecord.DroppedAttributesCount > 0 {
					if metadata.DroppedAttributesStats == nil {
						metadata.DroppedAttributesStats = &models.DroppedAttributesStats{}
					}
					metadata.DroppedAttributesStats.TotalDropped += logRecord.DroppedAttributesCount
					metadata.DroppedAttributesStats.RecordsWithDropped++
					if logRecord.DroppedAttributesCount > metadata.DroppedAttributesStats.MaxDropped {
						metadata.DroppedAttributesStats.MaxDropped = logRecord.DroppedAttributesCount
					}
				}
				
				// Track service-severity combination
				if serviceSeverities[serviceName] == nil {
					serviceSeverities[serviceName] = make(map[string]bool)
				}
				serviceSeverities[serviceName][severityText] = true
				
				// Extract body template (create analyzer per service+severity if needed)
				body := logRecord.GetBody().GetStringValue()
				if body != "" {
					analyzerKey := key // Use same key as logMap (service+severity)

					// Double-checked locking: optimistic RLock for the common
					// post-warmup path (key already exists), upgrade to write
					// lock only when we need to create a new entry.
					a.mu.RLock()
					analyzer, exists := a.bodyAnalyzers[analyzerKey]
					a.mu.RUnlock()
					if !exists {
						a.mu.Lock()
						analyzer, exists = a.bodyAnalyzers[analyzerKey]
						if !exists {
							analyzer = a.createBodyAnalyzer()
							a.bodyAnalyzers[analyzerKey] = analyzer
						}
						a.mu.Unlock()
					}

					analyzer.AddMessage(body)
				}

				// Process log record attributes directly from proto (avoids map allocation)
				forEachAttribute(logRecord.Attributes, func(attrKey, attrValue string) {
					// Feed to catalog
					_ = batch.StoreAttributeValue(ctx, attrKey, attrValue, "log", "attribute")

					// Track event.name separately
					if attrKey == "event.name" && attrValue != "" {
						if !contains(metadata.EventNames, attrValue) {
							metadata.EventNames = append(metadata.EventNames, attrValue)
						}
					}
					
					if metadata.AttributeKeys[attrKey] == nil {
						metadata.AttributeKeys[attrKey] = models.NewKeyMetadata()
					}
					metadata.AttributeKeys[attrKey].AddValue(attrValue)
				})
				
				// Update resource key counts
				for resKey, resValue := range resourceAttrs {
					if metadata.ResourceKeys[resKey] != nil {
						metadata.ResourceKeys[resKey].AddValue(resValue)
					}
				}
			}
		}
	}

	// Convert map to slice and calculate percentages
	results := make([]*models.LogMetadata, 0, len(logMap))
	for key, metadata := range logMap {
		// Calculate percentages for attribute keys
		for _, keyMeta := range metadata.AttributeKeys {
			if metadata.SampleCount > 0 {
				keyMeta.Percentage = float64(keyMeta.Count) / float64(metadata.SampleCount) * 100
			}
		}
		
		// Calculate percentages for resource keys
		for _, keyMeta := range metadata.ResourceKeys {
			if metadata.SampleCount > 0 {
				keyMeta.Percentage = float64(keyMeta.Count) / float64(metadata.SampleCount) * 100
			}
		}
		
		// Add body templates for this service+severity combination.
		// Templates are throttled: refreshed at most once per templateSyncInterval
		// to avoid the expensive GetClusters()/tokensToString work on every batch.
		a.mu.RLock()
		analyzer, exists := a.bodyAnalyzers[key]
		a.mu.RUnlock()

		if exists {
			a.lastTemplateSyncMu.Lock()
			lastSync := a.lastTemplateSync[key]
			now := time.Now()
			refresh := now.Sub(lastSync) >= templateSyncInterval
			if refresh {
				a.lastTemplateSync[key] = now
			}
			a.lastTemplateSyncMu.Unlock()

			if refresh {
				templates := analyzer.GetTemplates()
				metadata.BodyTemplates = make([]*models.BodyTemplate, 0, len(templates))
				for _, tmpl := range templates {
					metadata.BodyTemplates = append(metadata.BodyTemplates, &models.BodyTemplate{
						Template:   tmpl.Template,
						Count:      tmpl.Count,
						Percentage: tmpl.Percentage,
						Example:    tmpl.ExampleBody,
					})
				}
			}
		}
		
		results = append(results, metadata)
	}

	return results, nil
}

// isEmptyBytes checks if a byte slice is empty or all zeros
func isEmptyBytes(b []byte) bool {
	if len(b) == 0 {
		return true
	}
	for _, v := range b {
		if v != 0 {
			return false
		}
	}
	return true
}

// contains checks if a string slice contains a value
func contains(slice []string, value string) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}
