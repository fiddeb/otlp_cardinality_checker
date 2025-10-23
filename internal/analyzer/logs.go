package analyzer

import (
	"fmt"

	"github.com/fidde/otlp_cardinality_checker/pkg/models"
	collogspb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
)

// LogsAnalyzer extracts metadata from OTLP logs.
type LogsAnalyzer struct{}

// NewLogsAnalyzer creates a new logs analyzer.
func NewLogsAnalyzer() *LogsAnalyzer {
	return &LogsAnalyzer{}
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
				metadata.RecordCount++

				// Track service
				if serviceName != "" {
					metadata.Services[serviceName]++
				}

				// Extract log record attributes
				logAttrs := extractAttributes(logRecord.Attributes)
				for attrKey, attrValue := range logAttrs {
					if metadata.AttributeKeys[attrKey] == nil {
						metadata.AttributeKeys[attrKey] = models.NewKeyMetadata()
					}
					metadata.AttributeKeys[attrKey].AddValue(attrValue)
				}

				// Update resource key counts
				for resKey, resValue := range resourceAttrs {
					if metadata.ResourceKeys[resKey] != nil {
						metadata.ResourceKeys[resKey].AddValue(resValue)
					}
				}
			}
		}
	}

	// Convert map to slice
	results := make([]*models.LogMetadata, 0, len(logMap))
	for _, metadata := range logMap {
		results = append(results, metadata)
	}

	return results, nil
}
