package models

// MetricData is an interface that represents the different types of metric data.
// This matches the OTLP proto oneof structure for metric data types.
type MetricData interface {
	// GetType returns the metric type name (Gauge, Sum, Histogram, etc.)
	GetType() string

	// GetDataPointCount returns the number of data points
	GetDataPointCount() int64
}

// AggregationTemporality defines how a metric aggregator reports aggregated values.
// Matches opentelemetry.proto.metrics.v1.AggregationTemporality
type AggregationTemporality int32

const (
	// AggregationTemporalityUnspecified is the default, should not be used
	AggregationTemporalityUnspecified AggregationTemporality = 0

	// AggregationTemporalityDelta reports changes since last report time
	AggregationTemporalityDelta AggregationTemporality = 1

	// AggregationTemporalityCumulative reports cumulative changes since a fixed start time
	AggregationTemporalityCumulative AggregationTemporality = 2
)

func (a AggregationTemporality) String() string {
	switch a {
	case AggregationTemporalityDelta:
		return "DELTA"
	case AggregationTemporalityCumulative:
		return "CUMULATIVE"
	default:
		return "UNSPECIFIED"
	}
}

// GaugeMetric represents a scalar metric that always exports the "current value".
// Matches opentelemetry.proto.metrics.v1.Gauge
type GaugeMetric struct {
	// DataPointCount is the number of data points observed
	DataPointCount int64 `json:"data_point_count"`
}

func (g *GaugeMetric) GetType() string {
	return "Gauge"
}

func (g *GaugeMetric) GetDataPointCount() int64 {
	return g.DataPointCount
}

// SumMetric represents a scalar metric calculated as a sum of all reported measurements.
// Matches opentelemetry.proto.metrics.v1.Sum
type SumMetric struct {
	// DataPointCount is the number of data points observed
	DataPointCount int64 `json:"data_point_count"`

	// AggregationTemporality describes if the aggregator reports delta or cumulative changes
	AggregationTemporality AggregationTemporality `json:"aggregation_temporality"`

	// IsMonotonic indicates whether the sum is monotonic (only increases)
	IsMonotonic bool `json:"is_monotonic"`
}

func (s *SumMetric) GetType() string {
	return "Sum"
}

func (s *SumMetric) GetDataPointCount() int64 {
	return s.DataPointCount
}

// HistogramMetric represents a metric calculated by aggregating as a histogram.
// Matches opentelemetry.proto.metrics.v1.Histogram
type HistogramMetric struct {
	// DataPointCount is the number of data points observed
	DataPointCount int64 `json:"data_point_count"`

	// AggregationTemporality describes if the aggregator reports delta or cumulative changes
	AggregationTemporality AggregationTemporality `json:"aggregation_temporality"`

	// ExplicitBounds contains the set of bucket boundaries observed across data points.
	// This is a union of all explicit_bounds arrays seen.
	ExplicitBounds []float64 `json:"explicit_bounds,omitempty"`
}

func (h *HistogramMetric) GetType() string {
	return "Histogram"
}

func (h *HistogramMetric) GetDataPointCount() int64 {
	return h.DataPointCount
}

// ExponentialHistogramMetric represents an exponential histogram.
// Matches opentelemetry.proto.metrics.v1.ExponentialHistogram
type ExponentialHistogramMetric struct {
	// DataPointCount is the number of data points observed
	DataPointCount int64 `json:"data_point_count"`

	// AggregationTemporality describes if the aggregator reports delta or cumulative changes
	AggregationTemporality AggregationTemporality `json:"aggregation_temporality"`

	// Scale describes the resolution of the histogram (observed scales)
	Scales []int32 `json:"scales,omitempty"`
}

func (e *ExponentialHistogramMetric) GetType() string {
	return "ExponentialHistogram"
}

func (e *ExponentialHistogramMetric) GetDataPointCount() int64 {
	return e.DataPointCount
}

// SummaryMetric represents quantile summary data (Prometheus/OpenMetrics style).
// Matches opentelemetry.proto.metrics.v1.Summary
type SummaryMetric struct {
	// DataPointCount is the number of data points observed
	DataPointCount int64 `json:"data_point_count"`

	// Note: Summary metrics are always cumulative, no aggregation_temporality field
}

func (s *SummaryMetric) GetType() string {
	return "Summary"
}

func (s *SummaryMetric) GetDataPointCount() int64 {
	return s.DataPointCount
}

// maxExpHistBuckets is the default maximum bucket count used by the OTel SDK
// for exponential histograms (max_size parameter). Used as an upper bound
// when estimating Prometheus series from exponential histograms.
const maxExpHistBuckets = 160

// EstimatePrometheusActiveSeries estimates Prometheus series count based on OTLP series.
// For histograms, this accounts for bucket series plus _sum and _count.
func EstimatePrometheusActiveSeries(activeSeriesOTLP int64, data MetricData) int64 {
	if activeSeriesOTLP <= 0 {
		return activeSeriesOTLP
	}
	if data == nil {
		return activeSeriesOTLP
	}

	switch metric := data.(type) {
	case *HistogramMetric:
		bucketCount := len(metric.ExplicitBounds) + 1
		return activeSeriesOTLP * int64(bucketCount+2)
	case *ExponentialHistogramMetric:
		bucketCount := estimateExpHistBuckets(metric.Scales)
		return activeSeriesOTLP * int64(bucketCount+2)
	default:
		return activeSeriesOTLP
	}
}

// estimateExpHistBuckets estimates the Prometheus bucket count for an exponential
// histogram based on observed scales. Uses the maximum scale to compute resolution:
// buckets = 2^(maxScale+1), capped at the OTel SDK default max_size (160).
// Returns at least 1 bucket if any scales are present.
func estimateExpHistBuckets(scales []int32) int {
	if len(scales) == 0 {
		return 1
	}

	// Find maximum scale
	maxScale := scales[0]
	for _, s := range scales[1:] {
		if s > maxScale {
			maxScale = s
		}
	}

	// For negative or zero scales, use minimal bucket count
	if maxScale <= 0 {
		return 1
	}

	// 2^(maxScale+1), capped at OTel SDK default
	buckets := 1 << (maxScale + 1)
	if buckets > maxExpHistBuckets {
		buckets = maxExpHistBuckets
	}
	return buckets
}
