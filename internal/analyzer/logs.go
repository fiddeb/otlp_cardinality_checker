package analyzer

import (
	"fmt"

	"github.com/fidde/otlp_cardinality_checker/internal/analyzer/autotemplate"
	"github.com/fidde/otlp_cardinality_checker/pkg/models"
	collogspb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
)

// LogBodyAnalyzerInterface defines the interface for log body analyzers
type LogBodyAnalyzerInterface interface {
	AddMessage(body string)
	GetTemplates() []*LogTemplate
}

// LogsAnalyzer extracts metadata from OTLP logs.
type LogsAnalyzer struct {
	bodyAnalyzers    map[string]LogBodyAnalyzerInterface // One analyzer per severity level
	useAutoTemplate  bool                                // Whether to use autotemplate
	autoTemplateCfg  autotemplate.Config                 // Config for autotemplate
}

// NewLogsAnalyzer creates a new logs analyzer with regex-based template extraction.
func NewLogsAnalyzer() *LogsAnalyzer {
	return &LogsAnalyzer{
		bodyAnalyzers:   make(map[string]LogBodyAnalyzerInterface),
		useAutoTemplate: false,
	}
}

// NewLogsAnalyzerWithAutoTemplate creates a logs analyzer using autotemplate extraction.
func NewLogsAnalyzerWithAutoTemplate(cfg autotemplate.Config) *LogsAnalyzer {
	return &LogsAnalyzer{
		bodyAnalyzers:   make(map[string]LogBodyAnalyzerInterface),
		useAutoTemplate: true,
		autoTemplateCfg: cfg,
	}
}

// createBodyAnalyzer creates the appropriate analyzer type
func (a *LogsAnalyzer) createBodyAnalyzer() LogBodyAnalyzerInterface {
	if a.useAutoTemplate {
		return NewAutoLogBodyAnalyzerWithPatterns(a.autoTemplateCfg, nil)
	}
	return NewLogBodyAnalyzer()
}

// Analyze extracts metadata from an OTLP logs export request.
func (a *LogsAnalyzer) Analyze(req *collogspb.ExportLogsServiceRequest) ([]*models.LogMetadata, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	logMap := make(map[string]*models.LogMetadata)

	for _, resourceLogs := range req.ResourceLogs {
		// Extract resource attributes
		resourceAttrs := extractAttributes(resourceLogs.Resource.GetAttributes())
		serviceName := getServiceName(resourceAttrs)

		for _, scopeLogs := range resourceLogs.ScopeLogs {
			scopeInfo := &models.ScopeMetadata{
				Name:    scopeLogs.Scope.GetName(),
				Version: scopeLogs.Scope.GetVersion(),
			}

			for _, logRecord := range scopeLogs.LogRecords {
				severityText := logRecord.SeverityText
				if severityText == "" {
					severityText = "UNSET"
				}

				if _, exists := logMap[severityText]; !exists {
					logMap[severityText] = models.NewLogMetadata(severityText)
					logMap[severityText].ScopeInfo = scopeInfo

					// Add resource keys
					for resKey := range resourceAttrs {
						if logMap[severityText].ResourceKeys[resKey] == nil {
							logMap[severityText].ResourceKeys[resKey] = models.NewKeyMetadata()
						}
					}
				}

				metadata := logMap[severityText]
				metadata.SampleCount++

				// Track service
				if serviceName != "" {
					metadata.Services[serviceName]++
				}
				
				// Extract body template (create analyzer per severity if needed)
				body := logRecord.GetBody().GetStringValue()
				if body != "" {
					if _, exists := a.bodyAnalyzers[severityText]; !exists {
						a.bodyAnalyzers[severityText] = a.createBodyAnalyzer()
					}
					a.bodyAnalyzers[severityText].AddMessage(body)
				}

			// Extract log record attributes
			logAttrs := extractAttributes(logRecord.Attributes)
			for attrKey, attrValue := range logAttrs {
				if metadata.AttributeKeys[attrKey] == nil {
					metadata.AttributeKeys[attrKey] = models.NewKeyMetadata()
				}
				metadata.AttributeKeys[attrKey].AddValue(attrValue)
		}				// Update resource key counts
			for resKey, resValue := range resourceAttrs {
				if metadata.ResourceKeys[resKey] != nil {
					metadata.ResourceKeys[resKey].AddValue(resValue)
				}
			}
		}
	}
}	// Convert map to slice and calculate percentages
	results := make([]*models.LogMetadata, 0, len(logMap))
	for severityText, metadata := range logMap {
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
		
		// Add body templates for this severity level
		if analyzer, exists := a.bodyAnalyzers[severityText]; exists {
			templates := analyzer.GetTemplates()
			metadata.BodyTemplates = make([]*models.BodyTemplate, 0, len(templates))
			for _, tmpl := range templates {
				metadata.BodyTemplates = append(metadata.BodyTemplates, &models.BodyTemplate{
					Template:   tmpl.Template,
					Count:      tmpl.Count,
					Percentage: tmpl.Percentage,
					Example:    tmpl.SampleValues["original"],
				})
			}
		}
		
		results = append(results, metadata)
	}

	return results, nil
}
