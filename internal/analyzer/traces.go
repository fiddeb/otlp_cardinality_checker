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
					kindName := getSpanKind(span.Kind)
					spanMap[key] = models.NewSpanMetadata(span.Name, int32(span.Kind), kindName)
					spanMap[key].ScopeInfo = scopeInfo

					// Add resource keys
					for resKey := range resourceAttrs {
						if spanMap[key].ResourceKeys[resKey] == nil {
							spanMap[key].ResourceKeys[resKey] = models.NewKeyMetadata()
						}
					}
				}

				metadata := spanMap[key]
				metadata.SampleCount++

				// Track service
				if serviceName != "" {
					metadata.Services[serviceName]++
				}
				
				// Track trace_state presence (boolean flag, not the actual value)
				if span.TraceState != "" {
					metadata.HasTraceState = true
				}
				
				// Track parent_span_id presence (boolean flag, not the actual value)
				if len(span.ParentSpanId) > 0 && !isEmptyBytes(span.ParentSpanId) {
					metadata.HasParentSpanId = true
				}
				
				// Track status codes
				if span.Status != nil {
					statusCode := getStatusCodeName(span.Status.Code)
					if !contains(metadata.StatusCodes, statusCode) {
						metadata.StatusCodes = append(metadata.StatusCodes, statusCode)
					}
				}
				
				// Track dropped attributes statistics
				if span.DroppedAttributesCount > 0 {
					if metadata.DroppedAttributesStats == nil {
						metadata.DroppedAttributesStats = &models.DroppedCountStats{}
					}
					metadata.DroppedAttributesStats.TotalDropped += span.DroppedAttributesCount
					metadata.DroppedAttributesStats.ItemsWithDropped++
					if span.DroppedAttributesCount > metadata.DroppedAttributesStats.MaxDropped {
						metadata.DroppedAttributesStats.MaxDropped = span.DroppedAttributesCount
					}
				}
				
				// Track dropped events statistics
				if span.DroppedEventsCount > 0 {
					if metadata.DroppedEventsStats == nil {
						metadata.DroppedEventsStats = &models.DroppedCountStats{}
					}
					metadata.DroppedEventsStats.TotalDropped += span.DroppedEventsCount
					metadata.DroppedEventsStats.ItemsWithDropped++
					if span.DroppedEventsCount > metadata.DroppedEventsStats.MaxDropped {
						metadata.DroppedEventsStats.MaxDropped = span.DroppedEventsCount
					}
				}
				
				// Track dropped links statistics
				if span.DroppedLinksCount > 0 {
					if metadata.DroppedLinksStats == nil {
						metadata.DroppedLinksStats = &models.DroppedCountStats{}
					}
					metadata.DroppedLinksStats.TotalDropped += span.DroppedLinksCount
					metadata.DroppedLinksStats.ItemsWithDropped++
					if span.DroppedLinksCount > metadata.DroppedLinksStats.MaxDropped {
						metadata.DroppedLinksStats.MaxDropped = span.DroppedLinksCount
					}
				}
				
			// Update resource key counts and values
			for resKey, resValue := range resourceAttrs {
				if metadata.ResourceKeys[resKey] != nil {
					metadata.ResourceKeys[resKey].AddValue(resValue)
				}
			}

			// Extract span attributes
			spanAttrs := extractAttributes(span.Attributes)
			for attrKey, attrValue := range spanAttrs {
				if metadata.AttributeKeys[attrKey] == nil {
					metadata.AttributeKeys[attrKey] = models.NewKeyMetadata()
				}
				metadata.AttributeKeys[attrKey].AddValue(attrValue)
			}				// Extract event names and attributes
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

	// Convert map to slice and calculate percentages
	results := make([]*models.SpanMetadata, 0, len(spanMap))
	for _, metadata := range spanMap {
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

// getStatusCodeName converts OTLP status code to string.
func getStatusCodeName(code tracepb.Status_StatusCode) string {
	switch code {
	case tracepb.Status_STATUS_CODE_UNSET:
		return "UNSET"
	case tracepb.Status_STATUS_CODE_OK:
		return "OK"
	case tracepb.Status_STATUS_CODE_ERROR:
		return "ERROR"
	default:
		return "UNSET"
	}
}
