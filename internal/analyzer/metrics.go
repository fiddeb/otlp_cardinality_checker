// Package analyzer extracts metadata from OTLP telemetry signals.
package analyzer

import (
	"fmt"

	"github.com/fidde/otlp_cardinality_checker/pkg/models"
	colmetricspb "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	metricspb "go.opentelemetry.io/proto/otlp/metrics/v1"
)

// MetricsAnalyzer extracts metadata from OTLP metrics.
type MetricsAnalyzer struct{}

// NewMetricsAnalyzer creates a new metrics analyzer.
func NewMetricsAnalyzer() *MetricsAnalyzer {
	return &MetricsAnalyzer{}
}

// Analyze extracts metadata from an OTLP metrics export request.
func (a *MetricsAnalyzer) Analyze(req *colmetricspb.ExportMetricsServiceRequest) ([]*models.MetricMetadata, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	var results []*models.MetricMetadata

	for _, resourceMetrics := range req.ResourceMetrics {
		// Extract resource attributes
		resourceAttrs := extractAttributes(resourceMetrics.Resource.GetAttributes())
		serviceName := getServiceName(resourceAttrs)

		for _, scopeMetrics := range resourceMetrics.ScopeMetrics {
			scopeInfo := &models.ScopeMetadata{
				Name:    scopeMetrics.Scope.GetName(),
				Version: scopeMetrics.Scope.GetVersion(),
			}

			for _, metric := range scopeMetrics.Metrics {
				metadata := a.analyzeMetric(metric, resourceAttrs, serviceName, scopeInfo)
				if metadata != nil {
					results = append(results, metadata)
				}
			}
		}
	}

	return results, nil
}

// analyzeMetric extracts metadata from a single metric.
func (a *MetricsAnalyzer) analyzeMetric(
	metric *metricspb.Metric,
	resourceAttrs map[string]string,
	serviceName string,
	scopeInfo *models.ScopeMetadata,
) *models.MetricMetadata {
	metadata := models.NewMetricMetadata(metric.Name, getMetricType(metric))
	metadata.Description = metric.Description
	metadata.Unit = metric.Unit
	metadata.ScopeInfo = scopeInfo

	// Track service
	if serviceName != "" {
		metadata.Services[serviceName] = 0 // Will be incremented per data point
	}

	// Extract resource keys
	for key := range resourceAttrs {
		if metadata.ResourceKeys[key] == nil {
			metadata.ResourceKeys[key] = models.NewKeyMetadata()
		}
	}

	// Extract data point attributes based on metric type
	switch data := metric.Data.(type) {
	case *metricspb.Metric_Gauge:
		a.extractGaugeKeys(data.Gauge, metadata, serviceName)
	case *metricspb.Metric_Sum:
		a.extractSumKeys(data.Sum, metadata, serviceName)
	case *metricspb.Metric_Histogram:
		a.extractHistogramKeys(data.Histogram, metadata, serviceName)
	case *metricspb.Metric_ExponentialHistogram:
		a.extractExponentialHistogramKeys(data.ExponentialHistogram, metadata, serviceName)
	case *metricspb.Metric_Summary:
		a.extractSummaryKeys(data.Summary, metadata, serviceName)
	}

	return metadata
}

// extractGaugeKeys extracts label keys from gauge data points.
func (a *MetricsAnalyzer) extractGaugeKeys(gauge *metricspb.Gauge, metadata *models.MetricMetadata, serviceName string) {
	for _, dp := range gauge.DataPoints {
		metadata.SampleCount++
		if serviceName != "" {
			metadata.Services[serviceName]++
		}

		attrs := extractAttributes(dp.Attributes)
		for key, value := range attrs {
			if metadata.LabelKeys[key] == nil {
				metadata.LabelKeys[key] = models.NewKeyMetadata()
			}
			metadata.LabelKeys[key].AddValue(value)
		}
	}

	// Update percentages
	for _, keyMeta := range metadata.LabelKeys {
		keyMeta.UpdatePercentage(metadata.SampleCount)
	}
}

// extractSumKeys extracts label keys from sum data points.
func (a *MetricsAnalyzer) extractSumKeys(sum *metricspb.Sum, metadata *models.MetricMetadata, serviceName string) {
	for _, dp := range sum.DataPoints {
		metadata.SampleCount++
		if serviceName != "" {
			metadata.Services[serviceName]++
		}

		attrs := extractAttributes(dp.Attributes)
		for key, value := range attrs {
			if metadata.LabelKeys[key] == nil {
				metadata.LabelKeys[key] = models.NewKeyMetadata()
			}
			metadata.LabelKeys[key].AddValue(value)
		}
	}

	// Update percentages
	for _, keyMeta := range metadata.LabelKeys {
		keyMeta.UpdatePercentage(metadata.SampleCount)
	}
}

// extractHistogramKeys extracts label keys from histogram data points.
func (a *MetricsAnalyzer) extractHistogramKeys(histogram *metricspb.Histogram, metadata *models.MetricMetadata, serviceName string) {
	for _, dp := range histogram.DataPoints {
		metadata.SampleCount++
		if serviceName != "" {
			metadata.Services[serviceName]++
		}

		attrs := extractAttributes(dp.Attributes)
		for key, value := range attrs {
			if metadata.LabelKeys[key] == nil {
				metadata.LabelKeys[key] = models.NewKeyMetadata()
			}
			metadata.LabelKeys[key].AddValue(value)
		}
	}

	// Update percentages
	for _, keyMeta := range metadata.LabelKeys {
		keyMeta.UpdatePercentage(metadata.SampleCount)
	}
}

// extractExponentialHistogramKeys extracts label keys from exponential histogram data points.
func (a *MetricsAnalyzer) extractExponentialHistogramKeys(histogram *metricspb.ExponentialHistogram, metadata *models.MetricMetadata, serviceName string) {
	for _, dp := range histogram.DataPoints {
		metadata.SampleCount++
		if serviceName != "" {
			metadata.Services[serviceName]++
		}

		attrs := extractAttributes(dp.Attributes)
		for key, value := range attrs {
			if metadata.LabelKeys[key] == nil {
				metadata.LabelKeys[key] = models.NewKeyMetadata()
			}
			metadata.LabelKeys[key].AddValue(value)
		}
	}

	// Update percentages
	for _, keyMeta := range metadata.LabelKeys {
		keyMeta.UpdatePercentage(metadata.SampleCount)
	}
}

// extractSummaryKeys extracts label keys from summary data points.
func (a *MetricsAnalyzer) extractSummaryKeys(summary *metricspb.Summary, metadata *models.MetricMetadata, serviceName string) {
	for _, dp := range summary.DataPoints {
		metadata.SampleCount++
		if serviceName != "" {
			metadata.Services[serviceName]++
		}

		attrs := extractAttributes(dp.Attributes)
		for key, value := range attrs {
			if metadata.LabelKeys[key] == nil {
				metadata.LabelKeys[key] = models.NewKeyMetadata()
			}
			metadata.LabelKeys[key].AddValue(value)
		}
	}

	// Update percentages
	for _, keyMeta := range metadata.LabelKeys {
		keyMeta.UpdatePercentage(metadata.SampleCount)
	}
}

// getMetricType returns the metric type as a string.
func getMetricType(metric *metricspb.Metric) string {
	switch metric.Data.(type) {
	case *metricspb.Metric_Gauge:
		return "Gauge"
	case *metricspb.Metric_Sum:
		return "Sum"
	case *metricspb.Metric_Histogram:
		return "Histogram"
	case *metricspb.Metric_ExponentialHistogram:
		return "ExponentialHistogram"
	case *metricspb.Metric_Summary:
		return "Summary"
	default:
		return "Unknown"
	}
}

// extractAttributes converts OTLP KeyValue attributes to a map.
func extractAttributes(attrs []*commonpb.KeyValue) map[string]string {
	result := make(map[string]string, len(attrs))
	for _, attr := range attrs {
		result[attr.Key] = attributeValueToString(attr.Value)
	}
	return result
}

// attributeValueToString converts an OTLP attribute value to string.
func attributeValueToString(value *commonpb.AnyValue) string {
	if value == nil {
		return ""
	}

	switch v := value.Value.(type) {
	case *commonpb.AnyValue_StringValue:
		return v.StringValue
	case *commonpb.AnyValue_IntValue:
		return fmt.Sprintf("%d", v.IntValue)
	case *commonpb.AnyValue_DoubleValue:
		return fmt.Sprintf("%f", v.DoubleValue)
	case *commonpb.AnyValue_BoolValue:
		return fmt.Sprintf("%t", v.BoolValue)
	default:
		return fmt.Sprintf("%v", value)
	}
}

// getServiceName extracts service.name from resource attributes.
func getServiceName(attrs map[string]string) string {
	if name, ok := attrs["service.name"]; ok {
		return name
	}
	return ""
}
