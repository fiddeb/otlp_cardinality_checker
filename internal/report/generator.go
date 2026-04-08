package report

import (
	"context"
	"sort"
	"time"

	"github.com/fidde/otlp_cardinality_checker/internal/storage"
	"github.com/fidde/otlp_cardinality_checker/internal/version"
	"github.com/fidde/otlp_cardinality_checker/pkg/models"
)

// Generator builds a Report from storage data.
type Generator struct {
	store storage.Storage
}

// NewGenerator creates a new report generator.
func NewGenerator(store storage.Storage) *Generator {
	return &Generator{store: store}
}

// Generate queries storage and builds a Report.
func (g *Generator) Generate(ctx context.Context, duration time.Duration) (*Report, error) {
	metrics, err := g.store.ListMetrics(ctx, "")
	if err != nil {
		return nil, err
	}
	spans, err := g.store.ListSpans(ctx, "")
	if err != nil {
		return nil, err
	}
	logs, err := g.store.ListLogs(ctx, "")
	if err != nil {
		return nil, err
	}
	attrs, err := g.store.ListAttributes(ctx, nil)
	if err != nil {
		return nil, err
	}

	rpt := &Report{
		Version:     "1.0",
		GeneratedAt: time.Now().UTC(),
		OCCVersion:  version.Version,
	}
	if duration > 0 {
		rpt.Duration = duration.String()
	}

	rpt.Metrics = buildMetricItems(metrics)
	rpt.Spans = buildSpanItems(spans)
	rpt.Logs = buildLogItems(logs)
	rpt.Attributes = buildAttrItems(attrs)

	rpt.Summary = buildSummary(rpt)

	return rpt, nil
}

func buildMetricItems(metrics []*models.MetricMetadata) []MetricItem {
	items := make([]MetricItem, 0, len(metrics))
	for _, m := range metrics {
		cardinality := m.GetActiveSeries()
		keys := sortedKeys(m.LabelKeys)
		metricType := ""
		if m.Data != nil {
			metricType = m.Data.GetType()
		}
		items = append(items, MetricItem{
			Name:                 m.Name,
			Type:                 metricType,
			LabelKeys:            keys,
			SampleCount:          m.SampleCount,
			EstimatedCardinality: cardinality,
			Severity:             CardinalitySeverity(cardinality),
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].EstimatedCardinality > items[j].EstimatedCardinality
	})
	return items
}

func buildSpanItems(spans []*models.SpanMetadata) []SpanItem {
	items := make([]SpanItem, 0, len(spans))
	for _, s := range spans {
		cardinality := maxKeyCardinality(s.AttributeKeys)
		keys := sortedKeys(s.AttributeKeys)
		items = append(items, SpanItem{
			Name:                 s.Name,
			AttributeKeys:        keys,
			SpanCount:            s.SampleCount,
			EstimatedCardinality: cardinality,
			Severity:             CardinalitySeverity(cardinality),
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].EstimatedCardinality > items[j].EstimatedCardinality
	})
	return items
}

func buildLogItems(logs []*models.LogMetadata) []LogItem {
	items := make([]LogItem, 0, len(logs))
	for _, l := range logs {
		cardinality := maxKeyCardinality(l.AttributeKeys)
		keys := sortedKeys(l.AttributeKeys)
		items = append(items, LogItem{
			Severity:             l.Severity,
			AttributeKeys:        keys,
			LogCount:             l.SampleCount,
			EstimatedCardinality: cardinality,
			SeverityLevel:        CardinalitySeverity(cardinality),
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].EstimatedCardinality > items[j].EstimatedCardinality
	})
	return items
}

func buildAttrItems(attrs []*models.AttributeMetadata) []AttrItem {
	items := make([]AttrItem, 0, len(attrs))
	for _, a := range attrs {
		items = append(items, AttrItem{
			Key:                   a.Key,
			SignalTypes:           a.SignalTypes,
			EstimatedUniqueValues: a.EstimatedCardinality,
			Severity:              CardinalitySeverity(a.EstimatedCardinality),
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].EstimatedUniqueValues > items[j].EstimatedUniqueValues
	})
	return items
}

func buildSummary(rpt *Report) Summary {
	s := Summary{
		TotalMetrics:     len(rpt.Metrics),
		TotalSpanNames:   len(rpt.Spans),
		TotalLogPatterns: len(rpt.Logs),
		TotalAttributes:  len(rpt.Attributes),
	}
	for _, m := range rpt.Metrics {
		s.Samples.Metrics += m.SampleCount
		if m.Severity == SeverityWarning || m.Severity == SeverityCritical {
			s.HighCardinalityCount++
		}
	}
	for _, sp := range rpt.Spans {
		s.Samples.Spans += sp.SpanCount
		if sp.Severity == SeverityWarning || sp.Severity == SeverityCritical {
			s.HighCardinalityCount++
		}
	}
	for _, l := range rpt.Logs {
		s.Samples.Logs += l.LogCount
		if l.SeverityLevel == SeverityWarning || l.SeverityLevel == SeverityCritical {
			s.HighCardinalityCount++
		}
	}
	return s
}

// maxKeyCardinality returns the maximum EstimatedCardinality across all keys.
func maxKeyCardinality(keys map[string]*models.KeyMetadata) int64 {
	var max int64
	for _, k := range keys {
		if c := k.Cardinality(); c > max {
			max = c
		}
	}
	return max
}

// sortedKeys returns sorted key names from a KeyMetadata map.
func sortedKeys(keys map[string]*models.KeyMetadata) []string {
	result := make([]string, 0, len(keys))
	for k := range keys {
		result = append(result, k)
	}
	sort.Strings(result)
	return result
}
