package analyzer

import (
	"fmt"

	"github.com/fidde/otlp_cardinality_checker/pkg/models"
	coltracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
)

// TracesAnalyzer extracts metadata from OTLP traces.
type TracesAnalyzer struct{}

// NewTracesAnalyzer creates a new traces analyzer.
func NewTracesAnalyzer() *TracesAnalyzer {
	return &TracesAnalyzer{}
}

// Analyze extracts metadata from an OTLP traces export request.
func (a *TracesAnalyzer) Analyze(req *coltracepb.ExportTraceServiceRequest) ([]*models.SpanMetadata, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	spanMap := make(map[string]*models.SpanMetadata)

	for _, resourceSpans := range req.ResourceSpans {
		// Extract resource attributes
		resourceAttrs := extractAttributes(resourceSpans.Resource.GetAttributes())
		serviceName := getServiceName(resourceAttrs)

		for _, scopeSpans := range resourceSpans.ScopeSpans {
			scopeInfo := &models.ScopeMetadata{
				Name:    scopeSpans.Scope.GetName(),
				Version: scopeSpans.Scope.GetVersion(),
			}

			for _, span := range scopeSpans.Spans {
				key := span.Name
				if _, exists := spanMap[key]; !exists {
					spanMap[key] = models.NewSpanMetadata(span.Name, getSpanKind(span.Kind))
					spanMap[key].ScopeInfo = scopeInfo

					// Add resource keys and their values
					for resKey, resValue := range resourceAttrs {
						if spanMap[key].ResourceKeys[resKey] == nil {
							spanMap[key].ResourceKeys[resKey] = models.NewKeyMetadata()
						}
						// Resource attributes are the same for all spans with this name
						spanMap[key].ResourceKeys[resKey].AddValue(resValue)
					}
				}

				metadata := spanMap[key]
				metadata.SpanCount++

				// Track service
				if serviceName != "" {
					metadata.Services[serviceName]++
				}

				// Extract span attributes
				spanAttrs := extractAttributes(span.Attributes)
				for attrKey, attrValue := range spanAttrs {
					if metadata.AttributeKeys[attrKey] == nil {
						metadata.AttributeKeys[attrKey] = models.NewKeyMetadata()
					}
					metadata.AttributeKeys[attrKey].AddValue(attrValue)
				}

				// Extract event names and attributes
				for _, event := range span.Events {
					// Track event name
					found := false
					for _, name := range metadata.EventNames {
						if name == event.Name {
							found = true
							break
						}
					}
					if !found {
						metadata.EventNames = append(metadata.EventNames, event.Name)
					}

					// Track event attributes
					if metadata.EventAttributeKeys[event.Name] == nil {
						metadata.EventAttributeKeys[event.Name] = make(map[string]*models.KeyMetadata)
					}

					eventAttrs := extractAttributes(event.Attributes)
					for key, value := range eventAttrs {
						if metadata.EventAttributeKeys[event.Name][key] == nil {
							metadata.EventAttributeKeys[event.Name][key] = models.NewKeyMetadata()
						}
						metadata.EventAttributeKeys[event.Name][key].AddValue(value)
					}
				}

				// Extract link attributes
				for _, link := range span.Links {
					linkAttrs := extractAttributes(link.Attributes)
					for key, value := range linkAttrs {
						if metadata.LinkAttributeKeys[key] == nil {
							metadata.LinkAttributeKeys[key] = models.NewKeyMetadata()
						}
						metadata.LinkAttributeKeys[key].AddValue(value)
					}
				}
			}
		}
	}

	// Convert map to slice
	results := make([]*models.SpanMetadata, 0, len(spanMap))
	for _, metadata := range spanMap {
		results = append(results, metadata)
	}

	return results, nil
}

// getSpanKind converts OTLP span kind to string.
func getSpanKind(kind tracepb.Span_SpanKind) string {
	switch kind {
	case tracepb.Span_SPAN_KIND_INTERNAL:
		return "Internal"
	case tracepb.Span_SPAN_KIND_SERVER:
		return "Server"
	case tracepb.Span_SPAN_KIND_CLIENT:
		return "Client"
	case tracepb.Span_SPAN_KIND_PRODUCER:
		return "Producer"
	case tracepb.Span_SPAN_KIND_CONSUMER:
		return "Consumer"
	default:
		return "Unspecified"
	}
}
