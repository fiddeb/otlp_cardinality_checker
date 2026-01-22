// Package analyzer extracts metadata from OTLP telemetry signals.
package analyzer

import (
	"context"
	"fmt"

	"github.com/fidde/otlp_cardinality_checker/pkg/models"
	colmetricspb "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	metricspb "go.opentelemetry.io/proto/otlp/metrics/v1"
)

// MetricsAnalyzer extracts metadata from OTLP metrics.
type MetricsAnalyzer struct {
	catalog AttributeCatalog
}

// NewMetricsAnalyzer creates a new metrics analyzer.
func NewMetricsAnalyzer() *MetricsAnalyzer {
	return &MetricsAnalyzer{}
}

// NewMetricsAnalyzerWithCatalog creates a new metrics analyzer with attribute catalog.
func NewMetricsAnalyzerWithCatalog(catalog AttributeCatalog) *MetricsAnalyzer {
	return &MetricsAnalyzer{
		catalog: catalog,
	}
}

// Analyze extracts metadata from an OTLP metrics export request.
func (a *MetricsAnalyzer) Analyze(req *colmetricspb.ExportMetricsServiceRequest) ([]*models.MetricMetadata, error) {
	return a.AnalyzeWithContext(context.Background(), req)
}

// AnalyzeWithContext extracts metadata with context for attribute catalog.
func (a *MetricsAnalyzer) AnalyzeWithContext(ctx context.Context, req *colmetricspb.ExportMetricsServiceRequest) ([]*models.MetricMetadata, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	var results []*models.MetricMetadata

	for _, resourceMetrics := range req.ResourceMetrics {
		// Extract resource attributes
		resourceAttrs := extractAttributes(resourceMetrics.Resource.GetAttributes())
		serviceName := getServiceName(resourceAttrs)
		
		// Feed resource attributes to catalog
		extractAttributesToCatalog(ctx, a.catalog, resourceAttrs, "metric", "resource")

		for _, scopeMetrics := range resourceMetrics.ScopeMetrics {
			scopeInfo := &models.ScopeMetadata{
				Name:    scopeMetrics.Scope.GetName(),
				Version: scopeMetrics.Scope.GetVersion(),
			}

			for _, metric := range scopeMetrics.Metrics {
				metadata := a.analyzeMetricWithContext(ctx, metric, resourceAttrs, serviceName, scopeInfo)
				if metadata != nil {
					results = append(results, metadata)
				}
			}
		}
	}

	return results, nil
}

// analyzeMetric extracts metadata from a single metric (backward compatibility).
func (a *MetricsAnalyzer) analyzeMetric(
	metric *metricspb.Metric,
	resourceAttrs map[string]string,
	serviceName string,
	scopeInfo *models.ScopeMetadata,
) *models.MetricMetadata {
	return a.analyzeMetricWithContext(context.Background(), metric, resourceAttrs, serviceName, scopeInfo)
}

// analyzeMetricWithContext extracts metadata from a single metric with context.
func (a *MetricsAnalyzer) analyzeMetricWithContext(
	ctx context.Context,
	metric *metricspb.Metric,
	resourceAttrs map[string]string,
	serviceName string,
	scopeInfo *models.ScopeMetadata,
) *models.MetricMetadata {
	// Create appropriate MetricData type based on OTLP metric type
	var metricData models.MetricData
	
	switch data := metric.Data.(type) {
	case *metricspb.Metric_Gauge:
		metricData = &models.GaugeMetric{
			DataPointCount: int64(len(data.Gauge.DataPoints)),
		}
	case *metricspb.Metric_Sum:
		metricData = &models.SumMetric{
			DataPointCount:         int64(len(data.Sum.DataPoints)),
			AggregationTemporality: models.AggregationTemporality(data.Sum.AggregationTemporality),
			IsMonotonic:           data.Sum.IsMonotonic,
		}
	case *metricspb.Metric_Histogram:
		metricData = &models.HistogramMetric{
			DataPointCount:         int64(len(data.Histogram.DataPoints)),
			AggregationTemporality: models.AggregationTemporality(data.Histogram.AggregationTemporality),
			ExplicitBounds:         extractUniqueBounds(data.Histogram.DataPoints),
		}
	case *metricspb.Metric_ExponentialHistogram:
		metricData = &models.ExponentialHistogramMetric{
			DataPointCount:         int64(len(data.ExponentialHistogram.DataPoints)),
			AggregationTemporality: models.AggregationTemporality(data.ExponentialHistogram.AggregationTemporality),
			Scales:                extractUniqueScales(data.ExponentialHistogram.DataPoints),
		}
	case *metricspb.Metric_Summary:
		metricData = &models.SummaryMetric{
			DataPointCount: int64(len(data.Summary.DataPoints)),
		}
	default:
		// Unknown metric type, use Gauge as fallback
		metricData = &models.GaugeMetric{
			DataPointCount: 0,
		}
	}
	
	metadata := models.NewMetricMetadata(metric.Name, metricData)
	metadata.Description = metric.Description
	metadata.Unit = metric.Unit
	metadata.ScopeInfo = scopeInfo

	// Track service
	if serviceName != "" {
		metadata.Services[serviceName] = 0 // Will be incremented per data point
	}

	// Extract resource keys and add their values
	for key, value := range resourceAttrs {
		if metadata.ResourceKeys[key] == nil {
			metadata.ResourceKeys[key] = models.NewKeyMetadata()
		}
		// Resource attributes are the same for all data points in this metric
		// We'll add the value once here, and increment count per data point in extract functions
		metadata.ResourceKeys[key].AddValue(value)
	}

	// Extract data point attributes based on metric type
	switch data := metric.Data.(type) {
	case *metricspb.Metric_Gauge:
		a.extractGaugeKeys(ctx, data.Gauge, metadata, serviceName)
	case *metricspb.Metric_Sum:
		a.extractSumKeys(ctx, data.Sum, metadata, serviceName)
	case *metricspb.Metric_Histogram:
		a.extractHistogramKeys(ctx, data.Histogram, metadata, serviceName)
	case *metricspb.Metric_ExponentialHistogram:
		a.extractExponentialHistogramKeys(ctx, data.ExponentialHistogram, metadata, serviceName)
	case *metricspb.Metric_Summary:
		a.extractSummaryKeys(ctx, data.Summary, metadata, serviceName)
	}

	return metadata
}

// extractGaugeKeys extracts label keys from gauge data points.
func (a *MetricsAnalyzer) extractGaugeKeys(ctx context.Context, gauge *metricspb.Gauge, metadata *models.MetricMetadata, serviceName string) {
	for _, dp := range gauge.DataPoints {
		metadata.SampleCount++
		if serviceName != "" {
			metadata.Services[serviceName]++
		}

		attrs := extractAttributes(dp.Attributes)
		
		// Track unique series combination
		fingerprint := models.CreateSeriesFingerprintFast(attrs)
		metadata.AddSeriesFingerprint(fingerprint)
		
		// Feed attributes to catalog
		extractAttributesToCatalog(ctx, a.catalog, attrs, "metric", "attribute")
		
		for key, value := range attrs {
			if metadata.LabelKeys[key] == nil {
				metadata.LabelKeys[key] = models.NewKeyMetadata()
			}
			metadata.LabelKeys[key].AddValue(value)
		}
	}

	// Update percentages for label keys
	for _, keyMeta := range metadata.LabelKeys {
		keyMeta.UpdatePercentage(metadata.SampleCount)
	}
	
	// Update percentages for resource keys
	// Resource keys already have Count set by AddValue() in analyzeMetric()
	for _, keyMeta := range metadata.ResourceKeys {
		keyMeta.UpdatePercentage(metadata.SampleCount)
	}
}

// extractSumKeys extracts label keys from sum data points.
func (a *MetricsAnalyzer) extractSumKeys(ctx context.Context, sum *metricspb.Sum, metadata *models.MetricMetadata, serviceName string) {
	for _, dp := range sum.DataPoints {
		metadata.SampleCount++
		if serviceName != "" {
			metadata.Services[serviceName]++
		}

		attrs := extractAttributes(dp.Attributes)
		
		// Track unique series combination
		fingerprint := models.CreateSeriesFingerprintFast(attrs)
		metadata.AddSeriesFingerprint(fingerprint)
		
		// Feed attributes to catalog
		extractAttributesToCatalog(ctx, a.catalog, attrs, "metric", "attribute")
		
		for key, value := range attrs {
			if metadata.LabelKeys[key] == nil {
				metadata.LabelKeys[key] = models.NewKeyMetadata()
			}
			metadata.LabelKeys[key].AddValue(value)
		}
	}

	// Update percentages for label keys
	for _, keyMeta := range metadata.LabelKeys {
		keyMeta.UpdatePercentage(metadata.SampleCount)
	}
	
	// Update percentages for resource keys
	// Resource keys already have Count set by AddValue() in analyzeMetric()
	for _, keyMeta := range metadata.ResourceKeys {
		keyMeta.UpdatePercentage(metadata.SampleCount)
	}
}

// extractHistogramKeys extracts label keys from histogram data points.
func (a *MetricsAnalyzer) extractHistogramKeys(ctx context.Context, histogram *metricspb.Histogram, metadata *models.MetricMetadata, serviceName string) {
	for _, dp := range histogram.DataPoints {
		metadata.SampleCount++
		if serviceName != "" {
			metadata.Services[serviceName]++
		}

		attrs := extractAttributes(dp.Attributes)
		
		// Track unique series combination
		fingerprint := models.CreateSeriesFingerprintFast(attrs)
		metadata.AddSeriesFingerprint(fingerprint)
		
		// Feed attributes to catalog
		extractAttributesToCatalog(ctx, a.catalog, attrs, "metric", "attribute")
		
		for key, value := range attrs {
			if metadata.LabelKeys[key] == nil {
				metadata.LabelKeys[key] = models.NewKeyMetadata()
			}
			metadata.LabelKeys[key].AddValue(value)
		}
	}

	// Update percentages for label keys
	for _, keyMeta := range metadata.LabelKeys {
		keyMeta.UpdatePercentage(metadata.SampleCount)
	}
	
	// Update percentages for resource keys
	// Resource keys already have Count set by AddValue() in analyzeMetric()
	for _, keyMeta := range metadata.ResourceKeys {
		keyMeta.UpdatePercentage(metadata.SampleCount)
	}
}

// extractExponentialHistogramKeys extracts label keys from exponential histogram data points.
func (a *MetricsAnalyzer) extractExponentialHistogramKeys(ctx context.Context, histogram *metricspb.ExponentialHistogram, metadata *models.MetricMetadata, serviceName string) {
	for _, dp := range histogram.DataPoints {
		metadata.SampleCount++
		if serviceName != "" {
			metadata.Services[serviceName]++
		}

		attrs := extractAttributes(dp.Attributes)
		
		// Track unique series combination
		fingerprint := models.CreateSeriesFingerprintFast(attrs)
		metadata.AddSeriesFingerprint(fingerprint)
		
		// Feed attributes to catalog
		extractAttributesToCatalog(ctx, a.catalog, attrs, "metric", "attribute")
		
		for key, value := range attrs {
			if metadata.LabelKeys[key] == nil {
				metadata.LabelKeys[key] = models.NewKeyMetadata()
			}
			metadata.LabelKeys[key].AddValue(value)
		}
	}

	// Update percentages for label keys
	for _, keyMeta := range metadata.LabelKeys {
		keyMeta.UpdatePercentage(metadata.SampleCount)
	}
	
	// Update percentages for resource keys
	// Resource keys already have Count set by AddValue() in analyzeMetric()
	for _, keyMeta := range metadata.ResourceKeys {
		keyMeta.UpdatePercentage(metadata.SampleCount)
	}
}

// extractSummaryKeys extracts label keys from summary data points.
func (a *MetricsAnalyzer) extractSummaryKeys(ctx context.Context, summary *metricspb.Summary, metadata *models.MetricMetadata, serviceName string) {
	for _, dp := range summary.DataPoints {
		metadata.SampleCount++
		if serviceName != "" {
			metadata.Services[serviceName]++
		}

		attrs := extractAttributes(dp.Attributes)
		
		// Track unique series combination
		fingerprint := models.CreateSeriesFingerprintFast(attrs)
		metadata.AddSeriesFingerprint(fingerprint)
		
		// Feed attributes to catalog
		extractAttributesToCatalog(ctx, a.catalog, attrs, "metric", "attribute")
		
		for key, value := range attrs {
			if metadata.LabelKeys[key] == nil {
				metadata.LabelKeys[key] = models.NewKeyMetadata()
			}
			metadata.LabelKeys[key].AddValue(value)
		}
	}

	// Update percentages for label keys
	for _, keyMeta := range metadata.LabelKeys {
		keyMeta.UpdatePercentage(metadata.SampleCount)
	}
	
	// Update percentages for resource keys
	// Resource keys already have Count set by AddValue() in analyzeMetric()
	for _, keyMeta := range metadata.ResourceKeys {
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

// extractUniqueBounds extracts all unique explicit bounds from histogram data points
func extractUniqueBounds(dataPoints []*metricspb.HistogramDataPoint) []float64 {
	boundsSet := make(map[float64]bool)
	for _, dp := range dataPoints {
		for _, bound := range dp.ExplicitBounds {
			boundsSet[bound] = true
		}
	}
	
	// Convert to sorted slice
	bounds := make([]float64, 0, len(boundsSet))
	for bound := range boundsSet {
		bounds = append(bounds, bound)
	}
	
	// Sort bounds
	for i := 0; i < len(bounds); i++ {
		for j := i + 1; j < len(bounds); j++ {
			if bounds[j] < bounds[i] {
				bounds[i], bounds[j] = bounds[j], bounds[i]
			}
		}
	}
	
	return bounds
}

// extractUniqueScales extracts all unique scales from exponential histogram data points
func extractUniqueScales(dataPoints []*metricspb.ExponentialHistogramDataPoint) []int32 {
	scalesSet := make(map[int32]bool)
	for _, dp := range dataPoints {
		scalesSet[dp.Scale] = true
	}
	
	// Convert to sorted slice
	scales := make([]int32, 0, len(scalesSet))
	for scale := range scalesSet {
		scales = append(scales, scale)
	}
	
	// Sort scales
	for i := 0; i < len(scales); i++ {
		for j := i + 1; j < len(scales); j++ {
			if scales[j] < scales[i] {
				scales[i], scales[j] = scales[j], scales[i]
			}
		}
	}
	
	return scales
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
