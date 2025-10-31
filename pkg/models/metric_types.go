package models

import (
	"encoding/json"
	"fmt"
)

// MetricData is an interface that represents the different types of metric data.
// This matches the OTLP proto oneof structure for metric data types.
type MetricData interface {
	// GetType returns the metric type name (Gauge, Sum, Histogram, etc.)
	GetType() string
	
	// GetDataPointCount returns the number of data points
	GetDataPointCount() int64
}

// metricDataJSON is used for JSON marshaling/unmarshaling
type metricDataJSON struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// MarshalMetricData marshals a MetricData to JSON
func MarshalMetricData(md MetricData) ([]byte, error) {
	if md == nil {
		return json.Marshal(nil)
	}
	
	dataBytes, err := json.Marshal(md)
	if err != nil {
		return nil, err
	}
	
	wrapper := metricDataJSON{
		Type: md.GetType(),
		Data: dataBytes,
	}
	
	return json.Marshal(wrapper)
}

// UnmarshalMetricData unmarshals JSON to a MetricData
func UnmarshalMetricData(data []byte) (MetricData, error) {
	if len(data) == 0 || string(data) == "null" {
		return nil, nil
	}
	
	var wrapper metricDataJSON
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return nil, err
	}
	
	switch wrapper.Type {
	case "Gauge":
		var gauge GaugeMetric
		if err := json.Unmarshal(wrapper.Data, &gauge); err != nil {
			return nil, err
		}
		return &gauge, nil
		
	case "Sum":
		var sum SumMetric
		if err := json.Unmarshal(wrapper.Data, &sum); err != nil {
			return nil, err
		}
		return &sum, nil
		
	case "Histogram":
		var hist HistogramMetric
		if err := json.Unmarshal(wrapper.Data, &hist); err != nil {
			return nil, err
		}
		return &hist, nil
		
	case "ExponentialHistogram":
		var expHist ExponentialHistogramMetric
		if err := json.Unmarshal(wrapper.Data, &expHist); err != nil {
			return nil, err
		}
		return &expHist, nil
		
	case "Summary":
		var summary SummaryMetric
		if err := json.Unmarshal(wrapper.Data, &summary); err != nil {
			return nil, err
		}
		return &summary, nil
		
	default:
		return nil, fmt.Errorf("unknown metric type: %s", wrapper.Type)
	}
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
