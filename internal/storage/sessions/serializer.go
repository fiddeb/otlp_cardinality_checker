// Package sessions provides file-based session storage for saving and loading
// telemetry metadata snapshots.
package sessions

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/fidde/otlp_cardinality_checker/pkg/models"
)

// Serializer handles conversion between live store data and sessions.
type Serializer struct{}

// NewSerializer creates a new serializer.
func NewSerializer() *Serializer {
	return &Serializer{}
}

// MarshalMetrics converts MetricMetadata slice to SerializedMetric slice.
func (s *Serializer) MarshalMetrics(metrics []*models.MetricMetadata) ([]*models.SerializedMetric, error) {
	if len(metrics) == 0 {
		return nil, nil
	}

	result := make([]*models.SerializedMetric, 0, len(metrics))
	for _, m := range metrics {
		sm, err := s.marshalMetric(m)
		if err != nil {
			return nil, fmt.Errorf("marshaling metric %s: %w", m.Name, err)
		}
		result = append(result, sm)
	}
	return result, nil
}

// marshalMetric converts a single MetricMetadata to SerializedMetric.
func (s *Serializer) marshalMetric(m *models.MetricMetadata) (*models.SerializedMetric, error) {
	sm := &models.SerializedMetric{
		Name:         m.Name,
		Description:  m.Description,
		Unit:         m.Unit,
		Type:         m.GetType(),
		LabelKeys:    make(map[string]*models.SerializedKey),
		ResourceKeys: make(map[string]*models.SerializedKey),
		SampleCount:  m.SampleCount,
		Services:     m.Services,
		ActiveSeries: m.ActiveSeries,
	}

	// Serialize label keys with HLL
	for name, key := range m.LabelKeys {
		sk, err := models.SerializeKeyMetadata(key)
		if err != nil {
			return nil, err
		}
		sm.LabelKeys[name] = sk
	}

	// Serialize resource keys with HLL
	for name, key := range m.ResourceKeys {
		sk, err := models.SerializeKeyMetadata(key)
		if err != nil {
			return nil, err
		}
		sm.ResourceKeys[name] = sk
	}

	// Serialize series HLL if present
	seriesHLL := m.GetSeriesHLL()
	if seriesHLL != nil {
		hll, err := models.MarshalHLL(seriesHLL)
		if err != nil {
			return nil, err
		}
		sm.SeriesHLL = hll
	}

	return sm, nil
}

// UnmarshalMetrics converts SerializedMetric slice to MetricMetadata slice.
func (s *Serializer) UnmarshalMetrics(metrics []*models.SerializedMetric) ([]*models.MetricMetadata, error) {
	if len(metrics) == 0 {
		return nil, nil
	}

	result := make([]*models.MetricMetadata, 0, len(metrics))
	for _, sm := range metrics {
		m, err := s.unmarshalMetric(sm)
		if err != nil {
			return nil, fmt.Errorf("unmarshaling metric %s: %w", sm.Name, err)
		}
		result = append(result, m)
	}
	return result, nil
}

// unmarshalMetric converts a single SerializedMetric to MetricMetadata.
func (s *Serializer) unmarshalMetric(sm *models.SerializedMetric) (*models.MetricMetadata, error) {
	// Create basic metric - we don't restore the full MetricData since we only need Type string
	var metricData models.MetricData

	switch sm.Type {
	case "Gauge":
		metricData = &models.GaugeMetric{}
	case "Sum":
		metricData = &models.SumMetric{}
	case "Histogram":
		metricData = &models.HistogramMetric{}
	case "ExponentialHistogram":
		metricData = &models.ExponentialHistogramMetric{}
	case "Summary":
		metricData = &models.SummaryMetric{}
	default:
		metricData = nil
	}

	m := models.NewMetricMetadata(sm.Name, metricData)
	m.Description = sm.Description
	m.Unit = sm.Unit
	// Note: Type is stored in sm.Type but we don't need to restore MetricData for sessions
	m.SampleCount = sm.SampleCount
	m.Services = sm.Services
	m.ActiveSeries = sm.ActiveSeries

	// Deserialize label keys
	for name, sk := range sm.LabelKeys {
		km, err := models.DeserializeKeyMetadata(sk)
		if err != nil {
			return nil, err
		}
		m.LabelKeys[name] = km
	}

	// Deserialize resource keys
	for name, sk := range sm.ResourceKeys {
		km, err := models.DeserializeKeyMetadata(sk)
		if err != nil {
			return nil, err
		}
		m.ResourceKeys[name] = km
	}

	// Deserialize series HLL if present
	if sm.SeriesHLL != nil {
		hll, err := models.UnmarshalHLL(sm.SeriesHLL)
		if err != nil {
			return nil, err
		}
		m.SetSeriesHLL(hll)
	}

	return m, nil
}

// MarshalSpans converts SpanMetadata slice to SerializedSpan slice.
func (s *Serializer) MarshalSpans(spans []*models.SpanMetadata) ([]*models.SerializedSpan, error) {
	if len(spans) == 0 {
		return nil, nil
	}

	result := make([]*models.SerializedSpan, 0, len(spans))
	for _, sp := range spans {
		ss, err := s.marshalSpan(sp)
		if err != nil {
			return nil, fmt.Errorf("marshaling span %s: %w", sp.Name, err)
		}
		result = append(result, ss)
	}
	return result, nil
}

// marshalSpan converts a single SpanMetadata to SerializedSpan.
func (s *Serializer) marshalSpan(sp *models.SpanMetadata) (*models.SerializedSpan, error) {
	ss := &models.SerializedSpan{
		Name:               sp.Name,
		Kind:               sp.Kind,
		KindName:           sp.KindName,
		AttributeKeys:      make(map[string]*models.SerializedKey),
		EventNames:         sp.EventNames,
		EventAttributeKeys: make(map[string]map[string]*models.SerializedKey),
		LinkAttributeKeys:  make(map[string]*models.SerializedKey),
		ResourceKeys:       make(map[string]*models.SerializedKey),
		StatusCodes:        sp.StatusCodes,
		NamePatterns:       sp.NamePatterns,
		SampleCount:        sp.SampleCount,
		Services:           sp.Services,
	}

	// Serialize attribute keys
	for name, key := range sp.AttributeKeys {
		sk, err := models.SerializeKeyMetadata(key)
		if err != nil {
			return nil, err
		}
		ss.AttributeKeys[name] = sk
	}

	// Serialize resource keys
	for name, key := range sp.ResourceKeys {
		sk, err := models.SerializeKeyMetadata(key)
		if err != nil {
			return nil, err
		}
		ss.ResourceKeys[name] = sk
	}

	// Serialize link attribute keys
	for name, key := range sp.LinkAttributeKeys {
		sk, err := models.SerializeKeyMetadata(key)
		if err != nil {
			return nil, err
		}
		ss.LinkAttributeKeys[name] = sk
	}

	// Serialize event attribute keys
	for eventName, eventKeys := range sp.EventAttributeKeys {
		ss.EventAttributeKeys[eventName] = make(map[string]*models.SerializedKey)
		for keyName, key := range eventKeys {
			sk, err := models.SerializeKeyMetadata(key)
			if err != nil {
				return nil, err
			}
			ss.EventAttributeKeys[eventName][keyName] = sk
		}
	}

	return ss, nil
}

// UnmarshalSpans converts SerializedSpan slice to SpanMetadata slice.
func (s *Serializer) UnmarshalSpans(spans []*models.SerializedSpan) ([]*models.SpanMetadata, error) {
	if len(spans) == 0 {
		return nil, nil
	}

	result := make([]*models.SpanMetadata, 0, len(spans))
	for _, ss := range spans {
		sp, err := s.unmarshalSpan(ss)
		if err != nil {
			return nil, fmt.Errorf("unmarshaling span %s: %w", ss.Name, err)
		}
		result = append(result, sp)
	}
	return result, nil
}

// unmarshalSpan converts a single SerializedSpan to SpanMetadata.
func (s *Serializer) unmarshalSpan(ss *models.SerializedSpan) (*models.SpanMetadata, error) {
	sp := &models.SpanMetadata{
		Name:               ss.Name,
		Kind:               ss.Kind,
		KindName:           ss.KindName,
		AttributeKeys:      make(map[string]*models.KeyMetadata),
		EventNames:         ss.EventNames,
		EventAttributeKeys: make(map[string]map[string]*models.KeyMetadata),
		LinkAttributeKeys:  make(map[string]*models.KeyMetadata),
		ResourceKeys:       make(map[string]*models.KeyMetadata),
		StatusCodes:        ss.StatusCodes,
		NamePatterns:       ss.NamePatterns,
		SampleCount:        ss.SampleCount,
		Services:           ss.Services,
	}

	// Deserialize attribute keys
	for name, sk := range ss.AttributeKeys {
		km, err := models.DeserializeKeyMetadata(sk)
		if err != nil {
			return nil, err
		}
		sp.AttributeKeys[name] = km
	}

	// Deserialize resource keys
	for name, sk := range ss.ResourceKeys {
		km, err := models.DeserializeKeyMetadata(sk)
		if err != nil {
			return nil, err
		}
		sp.ResourceKeys[name] = km
	}

	// Deserialize link attribute keys
	for name, sk := range ss.LinkAttributeKeys {
		km, err := models.DeserializeKeyMetadata(sk)
		if err != nil {
			return nil, err
		}
		sp.LinkAttributeKeys[name] = km
	}

	// Deserialize event attribute keys
	for eventName, eventKeys := range ss.EventAttributeKeys {
		sp.EventAttributeKeys[eventName] = make(map[string]*models.KeyMetadata)
		for keyName, sk := range eventKeys {
			km, err := models.DeserializeKeyMetadata(sk)
			if err != nil {
				return nil, err
			}
			sp.EventAttributeKeys[eventName][keyName] = km
		}
	}

	return sp, nil
}

// MarshalLogs converts LogMetadata slice to SerializedLog slice.
func (s *Serializer) MarshalLogs(logs []*models.LogMetadata) ([]*models.SerializedLog, error) {
	if len(logs) == 0 {
		return nil, nil
	}

	result := make([]*models.SerializedLog, 0, len(logs))
	for _, l := range logs {
		sl, err := s.marshalLog(l)
		if err != nil {
			return nil, fmt.Errorf("marshaling log %s: %w", l.Severity, err)
		}
		result = append(result, sl)
	}
	return result, nil
}

// marshalLog converts a single LogMetadata to SerializedLog.
func (s *Serializer) marshalLog(l *models.LogMetadata) (*models.SerializedLog, error) {
	sl := &models.SerializedLog{
		Severity:       l.Severity,
		SeverityNumber: l.SeverityNumber,
		AttributeKeys:  make(map[string]*models.SerializedKey),
		ResourceKeys:   make(map[string]*models.SerializedKey),
		BodyTemplates:  l.BodyTemplates,
		EventNames:     l.EventNames,
		SampleCount:    l.SampleCount,
		Services:       l.Services,
	}

	// Serialize attribute keys
	for name, key := range l.AttributeKeys {
		sk, err := models.SerializeKeyMetadata(key)
		if err != nil {
			return nil, err
		}
		sl.AttributeKeys[name] = sk
	}

	// Serialize resource keys
	for name, key := range l.ResourceKeys {
		sk, err := models.SerializeKeyMetadata(key)
		if err != nil {
			return nil, err
		}
		sl.ResourceKeys[name] = sk
	}

	return sl, nil
}

// UnmarshalLogs converts SerializedLog slice to LogMetadata slice.
func (s *Serializer) UnmarshalLogs(logs []*models.SerializedLog) ([]*models.LogMetadata, error) {
	if len(logs) == 0 {
		return nil, nil
	}

	result := make([]*models.LogMetadata, 0, len(logs))
	for _, sl := range logs {
		l, err := s.unmarshalLog(sl)
		if err != nil {
			return nil, fmt.Errorf("unmarshaling log %s: %w", sl.Severity, err)
		}
		result = append(result, l)
	}
	return result, nil
}

// unmarshalLog converts a single SerializedLog to LogMetadata.
func (s *Serializer) unmarshalLog(sl *models.SerializedLog) (*models.LogMetadata, error) {
	l := &models.LogMetadata{
		Severity:       sl.Severity,
		SeverityNumber: sl.SeverityNumber,
		AttributeKeys:  make(map[string]*models.KeyMetadata),
		ResourceKeys:   make(map[string]*models.KeyMetadata),
		BodyTemplates:  sl.BodyTemplates,
		EventNames:     sl.EventNames,
		SampleCount:    sl.SampleCount,
		Services:       sl.Services,
	}

	// Deserialize attribute keys
	for name, sk := range sl.AttributeKeys {
		km, err := models.DeserializeKeyMetadata(sk)
		if err != nil {
			return nil, err
		}
		l.AttributeKeys[name] = km
	}

	// Deserialize resource keys
	for name, sk := range sl.ResourceKeys {
		km, err := models.DeserializeKeyMetadata(sk)
		if err != nil {
			return nil, err
		}
		l.ResourceKeys[name] = km
	}

	return l, nil
}

// MarshalAttributes converts AttributeMetadata slice to SerializedAttribute slice.
func (s *Serializer) MarshalAttributes(attrs []*models.AttributeMetadata) ([]*models.SerializedAttribute, error) {
	if len(attrs) == 0 {
		return nil, nil
	}

	result := make([]*models.SerializedAttribute, 0, len(attrs))
	for _, a := range attrs {
		sa, err := s.marshalAttribute(a)
		if err != nil {
			return nil, fmt.Errorf("marshaling attribute %s: %w", a.Key, err)
		}
		result = append(result, sa)
	}
	return result, nil
}

// marshalAttribute converts a single AttributeMetadata to SerializedAttribute.
func (s *Serializer) marshalAttribute(a *models.AttributeMetadata) (*models.SerializedAttribute, error) {
	sa := &models.SerializedAttribute{
		Key:                  a.Key,
		Count:                a.Count,
		EstimatedCardinality: a.EstimatedCardinality,
		ValueSamples:         a.ValueSamples,
		SignalTypes:          a.SignalTypes,
		Scope:                a.Scope,
		FirstSeen:            a.FirstSeen,
		LastSeen:             a.LastSeen,
	}

	// Serialize HLL if present
	hllBytes, err := a.MarshalHLL()
	if err != nil {
		return nil, err
	}
	if len(hllBytes) > 0 {
		sa.HLL = &models.SerializedHLL{
			Precision: hllBytes[0],
			Registers: base64.StdEncoding.EncodeToString(hllBytes[1:]),
		}
	}

	return sa, nil
}

// UnmarshalAttributes converts SerializedAttribute slice to AttributeMetadata slice.
func (s *Serializer) UnmarshalAttributes(attrs []*models.SerializedAttribute) ([]*models.AttributeMetadata, error) {
	if len(attrs) == 0 {
		return nil, nil
	}

	result := make([]*models.AttributeMetadata, 0, len(attrs))
	for _, sa := range attrs {
		a, err := s.unmarshalAttribute(sa)
		if err != nil {
			return nil, fmt.Errorf("unmarshaling attribute %s: %w", sa.Key, err)
		}
		result = append(result, a)
	}
	return result, nil
}

// unmarshalAttribute converts a single SerializedAttribute to AttributeMetadata.
func (s *Serializer) unmarshalAttribute(sa *models.SerializedAttribute) (*models.AttributeMetadata, error) {
	a := models.NewAttributeMetadata(sa.Key)
	a.Count = sa.Count
	a.EstimatedCardinality = sa.EstimatedCardinality
	a.ValueSamples = sa.ValueSamples
	a.SignalTypes = sa.SignalTypes
	a.Scope = sa.Scope
	a.FirstSeen = sa.FirstSeen
	a.LastSeen = sa.LastSeen

	// Deserialize HLL if present
	if sa.HLL != nil {
		registers, err := base64.StdEncoding.DecodeString(sa.HLL.Registers)
		if err != nil {
			return nil, err
		}
		hllData := make([]byte, 1+len(registers))
		hllData[0] = sa.HLL.Precision
		copy(hllData[1:], registers)

		if err := a.UnmarshalHLL(hllData); err != nil {
			return nil, err
		}
	}

	return a, nil
}

// CreateSessionOptions defines what to include when creating a session.
type CreateSessionOptions struct {
	Name        string
	Description string
	Signals     []string // empty = all
	Services    []string // empty = all
}

// CreateSession creates a new session from the current store state.
// This is the main entry point for snapshotting.
func (s *Serializer) CreateSession(
	ctx context.Context,
	opts CreateSessionOptions,
	metrics []*models.MetricMetadata,
	spans []*models.SpanMetadata,
	logs []*models.LogMetadata,
	attributes []*models.AttributeMetadata,
	services []string,
) (*models.Session, error) {
	session := &models.Session{
		Version:     CurrentVersion,
		ID:          opts.Name,
		Description: opts.Description,
		Created:     time.Now().UTC(),
		Signals:     opts.Signals,
		Data:        models.SessionData{},
		Stats:       models.SessionStats{Services: services},
	}

	// Determine which signals to include
	includeMetrics := len(opts.Signals) == 0 || containsString(opts.Signals, "metrics")
	includeSpans := len(opts.Signals) == 0 || containsString(opts.Signals, "spans")
	includeLogs := len(opts.Signals) == 0 || containsString(opts.Signals, "logs")
	includeAttributes := len(opts.Signals) == 0 || containsString(opts.Signals, "attributes")

	// Set signals list if it wasn't specified
	if len(session.Signals) == 0 {
		session.Signals = []string{}
		if includeMetrics {
			session.Signals = append(session.Signals, "metrics")
		}
		if includeSpans {
			session.Signals = append(session.Signals, "spans")
		}
		if includeLogs {
			session.Signals = append(session.Signals, "logs")
		}
		if includeAttributes {
			session.Signals = append(session.Signals, "attributes")
		}
	}

	// Filter and serialize metrics
	if includeMetrics && len(metrics) > 0 {
		filteredMetrics := filterMetricsByService(metrics, opts.Services)
		serialized, err := s.MarshalMetrics(filteredMetrics)
		if err != nil {
			return nil, fmt.Errorf("marshaling metrics: %w", err)
		}
		session.Data.Metrics = serialized
		// Count total data points, not just unique metric names
		for _, m := range filteredMetrics {
			session.Stats.MetricsCount += int(m.SampleCount)
		}
	}

	// Filter and serialize spans
	if includeSpans && len(spans) > 0 {
		filteredSpans := filterSpansByService(spans, opts.Services)
		serialized, err := s.MarshalSpans(filteredSpans)
		if err != nil {
			return nil, fmt.Errorf("marshaling spans: %w", err)
		}
		session.Data.Spans = serialized
		// Count total spans observed, not just unique span names
		for _, sp := range filteredSpans {
			session.Stats.SpansCount += int(sp.SampleCount)
		}
	}

	// Filter and serialize logs
	if includeLogs && len(logs) > 0 {
		filteredLogs := filterLogsByService(logs, opts.Services)
		serialized, err := s.MarshalLogs(filteredLogs)
		if err != nil {
			return nil, fmt.Errorf("marshaling logs: %w", err)
		}
		session.Data.Logs = serialized
		// Count total log messages, not just unique severity levels
		for _, l := range filteredLogs {
			session.Stats.LogsCount += int(l.SampleCount)
		}
	}

	// Serialize attributes
	if includeAttributes && len(attributes) > 0 {
		serialized, err := s.MarshalAttributes(attributes)
		if err != nil {
			return nil, fmt.Errorf("marshaling attributes: %w", err)
		}
		session.Data.Attributes = serialized
		session.Stats.AttributesCount = len(serialized)
	}

	return session, nil
}

// Helper functions

func containsString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

func filterMetricsByService(metrics []*models.MetricMetadata, services []string) []*models.MetricMetadata {
	if len(services) == 0 {
		return metrics
	}
	result := make([]*models.MetricMetadata, 0)
	for _, m := range metrics {
		if models.FilterByService(m.Services, services) {
			result = append(result, m)
		}
	}
	return result
}

func filterSpansByService(spans []*models.SpanMetadata, services []string) []*models.SpanMetadata {
	if len(services) == 0 {
		return spans
	}
	result := make([]*models.SpanMetadata, 0)
	for _, s := range spans {
		if models.FilterByService(s.Services, services) {
			result = append(result, s)
		}
	}
	return result
}

func filterLogsByService(logs []*models.LogMetadata, services []string) []*models.LogMetadata {
	if len(services) == 0 {
		return logs
	}
	result := make([]*models.LogMetadata, 0)
	for _, l := range logs {
		if models.FilterByService(l.Services, services) {
			result = append(result, l)
		}
	}
	return result
}
