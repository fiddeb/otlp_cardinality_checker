package clickhouse

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/fidde/otlp_cardinality_checker/internal/analyzer/autotemplate"
	"github.com/fidde/otlp_cardinality_checker/pkg/models"
)

// Store implements the storage.Storage interface using ClickHouse
type Store struct {
	conn   driver.Conn
	buffer *BatchBuffer
	logger *slog.Logger
	
	autoTemplate bool
	autoTmplCfg  autotemplate.Config
}

// NewStore creates a new ClickHouse storage instance
func NewStore(ctx context.Context, config *ConnectionConfig, logger *slog.Logger) (*Store, error) {
	if logger == nil {
		logger = slog.Default()
	}

	// Connect to ClickHouse
	conn, err := Connect(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("connecting to ClickHouse: %w", err)
	}

	// Initialize schema
	if err := InitializeSchema(ctx, conn); err != nil {
		conn.Close()
		return nil, fmt.Errorf("initializing schema: %w", err)
	}

	// Create batch buffer
	buffer := NewBatchBuffer(conn, logger)

	store := &Store{
		conn:         conn,
		buffer:       buffer,
		logger:       logger,
		autoTemplate: false,
		autoTmplCfg:  autotemplate.Config{},
	}

	return store, nil
}

// Metric operations

func (s *Store) StoreMetric(ctx context.Context, metric *models.MetricMetadata) error {
	now := time.Now()
	
	// Extract label and resource keys
	labelKeys := make([]string, 0, len(metric.LabelKeys))
	for k := range metric.LabelKeys {
		labelKeys = append(labelKeys, k)
	}
	
	resourceKeys := make([]string, 0, len(metric.ResourceKeys))
	for k := range metric.ResourceKeys {
		resourceKeys = append(resourceKeys, k)
	}
	
	// Extract services
	services := make([]string, 0, len(metric.Services))
	for svc := range metric.Services {
		services = append(services, svc)
	}
	
	// Get metric type string
	metricType := "unknown"
	unit := metric.Unit
	aggregationTemporality := ""
	isMonotonic := uint8(0)
	
	if metric.Data != nil {
		metricType = metric.Data.GetType()
		
		// Extract Sum-specific fields
		if sumData, ok := metric.Data.(*models.SumMetric); ok {
			aggregationTemporality = sumData.AggregationTemporality.String()
			if sumData.IsMonotonic {
				isMonotonic = 1
			}
		}
	}
	
	// Create row
	row := MetricRow{
		Name:                   metric.Name,
		ServiceName:            s.extractPrimaryService(metric.Services),
		MetricType:             metricType,
		Unit:                   unit,
		AggregationTemporality: aggregationTemporality,
		IsMonotonic:            isMonotonic,
		LabelKeys:              labelKeys,
		ResourceKeys:           resourceKeys,
		SampleCount:            uint64(metric.SampleCount),
		FirstSeen:              now,
		LastSeen:               now,
		Services:               services,
		ServiceCount:           uint32(len(services)),
	}
	
	return s.buffer.AddMetric(row)
}

func (s *Store) GetMetric(ctx context.Context, name string) (*models.MetricMetadata, error) {
	query := `
		SELECT 
			name, metric_type, unit, aggregation_temporality, is_monotonic,
			label_keys, resource_keys, sample_count,
			first_seen, last_seen, services, service_count
		FROM metrics FINAL
		WHERE name = ?
		LIMIT 1
	`
	
	row := s.conn.QueryRow(ctx, query, name)
	
	var (
		metricName             string
		metricType             string
		unit                   string
		aggregationTemporality string
		isMonotonic            uint8
		labelKeys              []string
		resourceKeys           []string
		sampleCount            uint64
		firstSeen              time.Time
		lastSeen               time.Time
		services               []string
		serviceCount           uint32
	)
	
	err := row.Scan(
		&metricName, &metricType, &unit, &aggregationTemporality, &isMonotonic,
		&labelKeys, &resourceKeys, &sampleCount,
		&firstSeen, &lastSeen, &services, &serviceCount,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, models.ErrNotFound
		}
		return nil, err
	}
	
	// Convert to MetricMetadata
	metric := &models.MetricMetadata{
		Name:         metricName,
		Unit:         unit,
		LabelKeys:    make(map[string]*models.KeyMetadata),
		ResourceKeys: make(map[string]*models.KeyMetadata),
		SampleCount:  int64(sampleCount),
		Services:     make(map[string]int64),
	}
	
	// Populate label keys
	for _, key := range labelKeys {
		metric.LabelKeys[key] = &models.KeyMetadata{
			Count: 0, // TODO: Query from attribute_values table
		}
	}
	
	// Populate resource keys
	for _, key := range resourceKeys {
		metric.ResourceKeys[key] = &models.KeyMetadata{
			Count: 0, // TODO: Query from attribute_values table
		}
	}
	
	// Populate services
	for _, svc := range services {
		metric.Services[svc] = 0 // Count not stored per-service in denormalized schema
	}
	
	// Create appropriate Data type
	// Parse aggregation temporality string back to enum
	var aggTemp models.AggregationTemporality
	switch aggregationTemporality {
	case "DELTA":
		aggTemp = models.AggregationTemporalityDelta
	case "CUMULATIVE":
		aggTemp = models.AggregationTemporalityCumulative
	default:
		aggTemp = models.AggregationTemporalityUnspecified
	}
	
	switch metricType {
	case "Gauge":
		metric.Data = &models.GaugeMetric{}
	case "Sum":
		metric.Data = &models.SumMetric{
			AggregationTemporality: aggTemp,
			IsMonotonic:            isMonotonic == 1,
		}
	case "Histogram":
		metric.Data = &models.HistogramMetric{
			AggregationTemporality: aggTemp,
		}
	case "Summary":
		metric.Data = &models.SummaryMetric{}
	case "ExponentialHistogram":
		metric.Data = &models.ExponentialHistogramMetric{
			AggregationTemporality: aggTemp,
		}
	}
	
	return metric, nil
}

func (s *Store) ListMetrics(ctx context.Context, serviceName string) ([]*models.MetricMetadata, error) {
	query := `
		SELECT 
			name, metric_type, unit, aggregation_temporality, is_monotonic,
			label_keys, resource_keys, sample_count,
			first_seen, last_seen, services, service_count
		FROM metrics FINAL
	`
	
	args := []interface{}{}
	if serviceName != "" {
		query += " WHERE service_name = ?"
		args = append(args, serviceName)
	}
	
	query += " ORDER BY name"
	
	rows, err := s.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var metrics []*models.MetricMetadata
	
	for rows.Next() {
		var (
			metricName             string
			metricType             string
			unit                   string
			aggregationTemporality string
			isMonotonic            uint8
			labelKeys              []string
			resourceKeys           []string
			sampleCount            uint64
			firstSeen              time.Time
			lastSeen               time.Time
			services               []string
			serviceCount           uint32
		)
		
		err := rows.Scan(
			&metricName, &metricType, &unit, &aggregationTemporality, &isMonotonic,
			&labelKeys, &resourceKeys, &sampleCount,
			&firstSeen, &lastSeen, &services, &serviceCount,
		)
		if err != nil {
			return nil, err
		}
		
		metric := &models.MetricMetadata{
			Name:         metricName,
			Unit:         unit,
			LabelKeys:    make(map[string]*models.KeyMetadata),
			ResourceKeys: make(map[string]*models.KeyMetadata),
			SampleCount:  int64(sampleCount),
			Services:     make(map[string]int64),
		}
		
		for _, key := range labelKeys {
			metric.LabelKeys[key] = &models.KeyMetadata{Count: 0}
		}
		
		for _, key := range resourceKeys {
			metric.ResourceKeys[key] = &models.KeyMetadata{Count: 0}
		}
		
		for _, svc := range services {
			metric.Services[svc] = 0
		}
		
		// Parse aggregation temporality
		var aggTemp models.AggregationTemporality
		switch aggregationTemporality {
		case "DELTA":
			aggTemp = models.AggregationTemporalityDelta
		case "CUMULATIVE":
			aggTemp = models.AggregationTemporalityCumulative
		default:
			aggTemp = models.AggregationTemporalityUnspecified
		}
		
		// Set Data type
		switch metricType {
		case "Gauge":
			metric.Data = &models.GaugeMetric{}
		case "Sum":
			metric.Data = &models.SumMetric{
				AggregationTemporality: aggTemp,
				IsMonotonic:            isMonotonic == 1,
			}
		case "Histogram":
			metric.Data = &models.HistogramMetric{
				AggregationTemporality: aggTemp,
			}
		case "Summary":
			metric.Data = &models.SummaryMetric{}
		case "ExponentialHistogram":
			metric.Data = &models.ExponentialHistogramMetric{
				AggregationTemporality: aggTemp,
			}
		}
		
		metrics = append(metrics, metric)
	}
	
	return metrics, rows.Err()
}

// Helper to extract primary service name from services map
func (s *Store) extractPrimaryService(services map[string]int64) string {
	if len(services) == 0 {
		return "unknown"
	}
	
	// Return first service (could be improved with sorting by count)
	for svc := range services {
		return svc
	}
	
	return "unknown"
}

// Span operations - basic implementations

func (s *Store) StoreSpan(ctx context.Context, span *models.SpanMetadata) error {
	now := time.Now()
	
	attributeKeys := make([]string, 0, len(span.AttributeKeys))
	for k := range span.AttributeKeys {
		attributeKeys = append(attributeKeys, k)
	}
	
	resourceKeys := make([]string, 0, len(span.ResourceKeys))
	for k := range span.ResourceKeys {
		resourceKeys = append(resourceKeys, k)
	}
	
	services := make([]string, 0, len(span.Services))
	for svc := range span.Services {
		services = append(services, svc)
	}
	
	statusCodes := span.StatusCodes
	if statusCodes == nil {
		statusCodes = []string{}
	}
	
	row := SpanRow{
		Name:          span.Name,
		ServiceName:   s.extractPrimaryService(span.Services),
		Kind:          uint8(span.Kind),
		KindName:      span.KindName,
		AttributeKeys: attributeKeys,
		ResourceKeys:  resourceKeys,
		EventNames:    span.EventNames,
		HasLinks:      0,
		StatusCodes:   statusCodes,
		SampleCount:   uint64(span.SampleCount),
		FirstSeen:     now,
		LastSeen:      now,
		Services:      services,
		ServiceCount:  uint32(len(services)),
	}
	
	if len(span.LinkAttributeKeys) > 0 {
		row.HasLinks = 1
	}
	
	// Add dropped stats
	if span.DroppedAttributesStats != nil {
		row.DroppedAttrsTotal = uint64(span.DroppedAttributesStats.TotalDropped)
		row.DroppedAttrsMax = span.DroppedAttributesStats.MaxDropped
	}
	if span.DroppedEventsStats != nil {
		row.DroppedEventsTotal = uint64(span.DroppedEventsStats.TotalDropped)
		row.DroppedEventsMax = span.DroppedEventsStats.MaxDropped
	}
	if span.DroppedLinksStats != nil {
		row.DroppedLinksTotal = uint64(span.DroppedLinksStats.TotalDropped)
		row.DroppedLinksMax = span.DroppedLinksStats.MaxDropped
	}
	
	return s.buffer.AddSpan(row)
}

func (s *Store) GetSpan(ctx context.Context, name string) (*models.SpanMetadata, error) {
	query := `
		SELECT 
			name, kind, kind_name, attribute_keys, resource_keys,
			event_names, has_links, status_codes,
			dropped_attrs_total, dropped_attrs_max,
			dropped_events_total, dropped_events_max,
			dropped_links_total, dropped_links_max,
			sample_count, services
		FROM spans FINAL
		WHERE name = ?
		LIMIT 1
	`
	
	row := s.conn.QueryRow(ctx, query, name)
	
	var (
		spanName          string
		kind              uint8
		kindName          string
		attributeKeys     []string
		resourceKeys      []string
		eventNames        []string
		hasLinks          uint8
		statusCodes       []string
		droppedAttrsTotal uint64
		droppedAttrsMax   uint32
		droppedEventsTotal uint64
		droppedEventsMax   uint32
		droppedLinksTotal  uint64
		droppedLinksMax    uint32
		sampleCount        uint64
		services           []string
	)
	
	err := row.Scan(
		&spanName, &kind, &kindName, &attributeKeys, &resourceKeys,
		&eventNames, &hasLinks, &statusCodes,
		&droppedAttrsTotal, &droppedAttrsMax,
		&droppedEventsTotal, &droppedEventsMax,
		&droppedLinksTotal, &droppedLinksMax,
		&sampleCount, &services,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, models.ErrNotFound
		}
		return nil, err
	}
	
	span := &models.SpanMetadata{
		Name:          spanName,
		Kind:          int32(kind),
		KindName:      kindName,
		AttributeKeys: make(map[string]*models.KeyMetadata),
		ResourceKeys:  make(map[string]*models.KeyMetadata),
		EventNames:    eventNames,
		StatusCodes:   statusCodes,
		SampleCount:   int64(sampleCount),
		Services:      make(map[string]int64),
	}
	
	for _, key := range attributeKeys {
		span.AttributeKeys[key] = &models.KeyMetadata{Count: 0}
	}
	
	for _, key := range resourceKeys {
		span.ResourceKeys[key] = &models.KeyMetadata{Count: 0}
	}
	
	for _, svc := range services {
		span.Services[svc] = 0
	}
	
	if droppedAttrsTotal > 0 {
		span.DroppedAttributesStats = &models.DroppedCountStats{
			TotalDropped: uint32(droppedAttrsTotal),
			MaxDropped:   droppedAttrsMax,
		}
	}
	
	if droppedEventsTotal > 0 {
		span.DroppedEventsStats = &models.DroppedCountStats{
			TotalDropped: uint32(droppedEventsTotal),
			MaxDropped:   droppedEventsMax,
		}
	}
	
	if droppedLinksTotal > 0 {
		span.DroppedLinksStats = &models.DroppedCountStats{
			TotalDropped: uint32(droppedLinksTotal),
			MaxDropped:   droppedLinksMax,
		}
	}
	
	return span, nil
}

func (s *Store) ListSpans(ctx context.Context, serviceName string) ([]*models.SpanMetadata, error) {
	query := `
		SELECT 
			name, kind, kind_name, attribute_keys, resource_keys,
			event_names, has_links, status_codes,
			dropped_attrs_total, dropped_attrs_max,
			dropped_events_total, dropped_events_max,
			dropped_links_total, dropped_links_max,
			sample_count, services
		FROM spans FINAL
	`
	
	args := []interface{}{}
	if serviceName != "" {
		query += " WHERE service_name = ?"
		args = append(args, serviceName)
	}
	
	query += " ORDER BY name"
	
	rows, err := s.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var spans []*models.SpanMetadata
	
	for rows.Next() {
		var (
			spanName          string
			kind              uint8
			kindName          string
			attributeKeys     []string
			resourceKeys      []string
			eventNames        []string
			hasLinks          uint8
			statusCodes       []string
			droppedAttrsTotal uint64
			droppedAttrsMax   uint32
			droppedEventsTotal uint64
			droppedEventsMax   uint32
			droppedLinksTotal  uint64
			droppedLinksMax    uint32
			sampleCount        uint64
			services           []string
		)
		
		err := rows.Scan(
			&spanName, &kind, &kindName, &attributeKeys, &resourceKeys,
			&eventNames, &hasLinks, &statusCodes,
			&droppedAttrsTotal, &droppedAttrsMax,
			&droppedEventsTotal, &droppedEventsMax,
			&droppedLinksTotal, &droppedLinksMax,
			&sampleCount, &services,
		)
		if err != nil {
			return nil, err
		}
		
		span := &models.SpanMetadata{
			Name:          spanName,
			Kind:          int32(kind),
			KindName:      kindName,
			AttributeKeys: make(map[string]*models.KeyMetadata),
			ResourceKeys:  make(map[string]*models.KeyMetadata),
			EventNames:    eventNames,
			StatusCodes:   statusCodes,
			SampleCount:   int64(sampleCount),
			Services:      make(map[string]int64),
		}
		
		for _, key := range attributeKeys {
			span.AttributeKeys[key] = &models.KeyMetadata{Count: 0}
		}
		
		for _, key := range resourceKeys {
			span.ResourceKeys[key] = &models.KeyMetadata{Count: 0}
		}
		
		for _, svc := range services {
			span.Services[svc] = 0
		}
		
		if droppedAttrsTotal > 0 {
			span.DroppedAttributesStats = &models.DroppedCountStats{
				TotalDropped: uint32(droppedAttrsTotal),
				MaxDropped:   droppedAttrsMax,
			}
		}
		
		if droppedEventsTotal > 0 {
			span.DroppedEventsStats = &models.DroppedCountStats{
				TotalDropped: uint32(droppedEventsTotal),
				MaxDropped:   droppedEventsMax,
			}
		}
		
		if droppedLinksTotal > 0 {
			span.DroppedLinksStats = &models.DroppedCountStats{
				TotalDropped: uint32(droppedLinksTotal),
				MaxDropped:   droppedLinksMax,
			}
		}
		
		spans = append(spans, span)
	}
	
	return spans, rows.Err()
}

// Log operations

func (s *Store) StoreLog(ctx context.Context, log *models.LogMetadata) error {
	now := time.Now()
	
	attributeKeys := make([]string, 0, len(log.AttributeKeys))
	for k := range log.AttributeKeys {
		attributeKeys = append(attributeKeys, k)
	}
	
	resourceKeys := make([]string, 0, len(log.ResourceKeys))
	for k := range log.ResourceKeys {
		resourceKeys = append(resourceKeys, k)
	}
	
	services := make([]string, 0, len(log.Services))
	for svc := range log.Services {
		services = append(services, svc)
	}
	
	// Extract first body template as example (if any)
	exampleBody := ""
	if len(log.BodyTemplates) > 0 {
		exampleBody = log.BodyTemplates[0].Example
	}
	
	hasTraceContext := uint8(0)
	if log.HasTraceContext {
		hasTraceContext = 1
	}
	
	hasSpanContext := uint8(0)
	if log.HasSpanContext {
		hasSpanContext = 1
	}
	
	// Use severity text as pattern_template for now
	// In production, this would use the Drain algorithm output
	patternTemplate := log.Severity
	
	row := LogRow{
		PatternTemplate: patternTemplate,
		Severity:        log.Severity,
		SeverityNumber:  uint8(log.SeverityNumber),
		ServiceName:     s.extractPrimaryService(log.Services),
		AttributeKeys:   attributeKeys,
		ResourceKeys:    resourceKeys,
		ExampleBody:     exampleBody,
		HasTraceContext: hasTraceContext,
		HasSpanContext:  hasSpanContext,
		SampleCount:     uint64(log.SampleCount),
		FirstSeen:       now,
		LastSeen:        now,
		Services:        services,
		ServiceCount:    uint32(len(services)),
	}
	
	if log.DroppedAttributesStats != nil {
		row.DroppedAttrsTotal = uint64(log.DroppedAttributesStats.TotalDropped)
		row.DroppedAttrsMax = log.DroppedAttributesStats.MaxDropped
	}
	
	return s.buffer.AddLog(row)
}

func (s *Store) GetLog(ctx context.Context, severityText string) (*models.LogMetadata, error) {
	query := `
		SELECT 
			pattern_template, severity, severity_number, attribute_keys, resource_keys,
			example_body, has_trace_context, has_span_context,
			dropped_attrs_total, dropped_attrs_max,
			sample_count, services
		FROM logs FINAL
		WHERE severity = ?
		LIMIT 1
	`
	
	row := s.conn.QueryRow(ctx, query, severityText)
	
	var (
		patternTemplate   string
		severity          string
		severityNumber    uint8
		attributeKeys     []string
		resourceKeys      []string
		exampleBody       string
		hasTraceContext   uint8
		hasSpanContext    uint8
		droppedAttrsTotal uint64
		droppedAttrsMax   uint32
		sampleCount       uint64
		services          []string
	)
	
	err := row.Scan(
		&patternTemplate, &severity, &severityNumber, &attributeKeys, &resourceKeys,
		&exampleBody, &hasTraceContext, &hasSpanContext,
		&droppedAttrsTotal, &droppedAttrsMax,
		&sampleCount, &services,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, models.ErrNotFound
		}
		return nil, err
	}
	
	logMeta := &models.LogMetadata{
		Severity:         severity,
		SeverityNumber:   int32(severityNumber),
		AttributeKeys:    make(map[string]*models.KeyMetadata),
		ResourceKeys:     make(map[string]*models.KeyMetadata),
		HasTraceContext:  hasTraceContext == 1,
		HasSpanContext:   hasSpanContext == 1,
		SampleCount:      int64(sampleCount),
		Services:         make(map[string]int64),
	}
	
	for _, key := range attributeKeys {
		logMeta.AttributeKeys[key] = &models.KeyMetadata{Count: 0}
	}
	
	for _, key := range resourceKeys {
		logMeta.ResourceKeys[key] = &models.KeyMetadata{Count: 0}
	}
	
	for _, svc := range services {
		logMeta.Services[svc] = 0
	}
	
	if droppedAttrsTotal > 0 {
		logMeta.DroppedAttributesStats = &models.DroppedAttributesStats{
			TotalDropped: uint32(droppedAttrsTotal),
			MaxDropped:   droppedAttrsMax,
		}
	}
	
	return logMeta, nil
}

func (s *Store) ListLogs(ctx context.Context, serviceName string) ([]*models.LogMetadata, error) {
	query := `
		SELECT 
			pattern_template, severity, severity_number, attribute_keys, resource_keys,
			example_body, has_trace_context, has_span_context,
			dropped_attrs_total, dropped_attrs_max,
			sample_count, services
		FROM logs FINAL
	`
	
	args := []interface{}{}
	if serviceName != "" {
		query += " WHERE service_name = ?"
		args = append(args, serviceName)
	}
	
	query += " ORDER BY severity, pattern_template"
	
	rows, err := s.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var logs []*models.LogMetadata
	
	for rows.Next() {
		var (
			patternTemplate   string
			severity          string
			severityNumber    uint8
			attributeKeys     []string
			resourceKeys      []string
			exampleBody       string
			hasTraceContext   uint8
			hasSpanContext    uint8
			droppedAttrsTotal uint64
			droppedAttrsMax   uint32
			sampleCount       uint64
			services          []string
		)
		
		err := rows.Scan(
			&patternTemplate, &severity, &severityNumber, &attributeKeys, &resourceKeys,
			&exampleBody, &hasTraceContext, &hasSpanContext,
			&droppedAttrsTotal, &droppedAttrsMax,
			&sampleCount, &services,
		)
		if err != nil {
			return nil, err
		}
		
		logMeta := &models.LogMetadata{
			Severity:         severity,
			SeverityNumber:   int32(severityNumber),
			AttributeKeys:    make(map[string]*models.KeyMetadata),
			ResourceKeys:     make(map[string]*models.KeyMetadata),
			HasTraceContext:  hasTraceContext == 1,
			HasSpanContext:   hasSpanContext == 1,
			SampleCount:      int64(sampleCount),
			Services:         make(map[string]int64),
		}
		
		for _, key := range attributeKeys {
			logMeta.AttributeKeys[key] = &models.KeyMetadata{Count: 0}
		}
		
		for _, key := range resourceKeys {
			logMeta.ResourceKeys[key] = &models.KeyMetadata{Count: 0}
		}
		
		for _, svc := range services {
			logMeta.Services[svc] = 0
		}
		
		if droppedAttrsTotal > 0 {
			logMeta.DroppedAttributesStats = &models.DroppedAttributesStats{
				TotalDropped: uint32(droppedAttrsTotal),
				MaxDropped:   droppedAttrsMax,
			}
		}
		
		logs = append(logs, logMeta)
	}
	
	return logs, rows.Err()
}

// Advanced query operations - stubs for now

func (s *Store) GetLogPatterns(ctx context.Context, minCount int64, minServices int) (*models.PatternExplorerResponse, error) {
	return &models.PatternExplorerResponse{}, nil
}

func (s *Store) GetHighCardinalityKeys(ctx context.Context, threshold int, limit int) (*models.CrossSignalCardinalityResponse, error) {
	return &models.CrossSignalCardinalityResponse{}, nil
}

func (s *Store) GetMetadataComplexity(ctx context.Context, threshold int, limit int) (*models.MetadataComplexityResponse, error) {
	return &models.MetadataComplexityResponse{}, nil
}

// Attribute operations

func (s *Store) StoreAttributeValue(ctx context.Context, key, value, signalType, scope string) error {
	now := time.Now()
	
	row := AttributeRow{
		Key:              key,
		Value:            value,
		SignalType:       signalType,
		Scope:            scope,
		ObservationCount: 1,
		FirstSeen:        now,
		LastSeen:         now,
	}
	
	return s.buffer.AddAttribute(row)
}

func (s *Store) GetAttribute(ctx context.Context, key string) (*models.AttributeMetadata, error) {
	query := `
		SELECT
			key,
			uniqExact(value) AS cardinality,
			groupArray(5)(value) AS samples,
			sum(observation_count) AS total_observations,
			min(first_seen) AS first_seen,
			max(last_seen) AS last_seen
		FROM attribute_values
		WHERE key = ?
		GROUP BY key
	`
	
	row := s.conn.QueryRow(ctx, query, key)
	
	var (
		keyName      string
		cardinality  uint64
		samples      []string
		observations uint64
		firstSeen    time.Time
		lastSeen     time.Time
	)
	
	err := row.Scan(&keyName, &cardinality, &samples, &observations, &firstSeen, &lastSeen)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, models.ErrNotFound
		}
		return nil, err
	}
	
	attr := &models.AttributeMetadata{
		Key:                  keyName,
		EstimatedCardinality: int64(cardinality),
		ValueSamples:         samples,
		Count:                int64(observations),
		FirstSeen:            firstSeen,
		LastSeen:             lastSeen,
	}
	
	return attr, nil
}

func (s *Store) ListAttributes(ctx context.Context, filter *models.AttributeFilter) ([]*models.AttributeMetadata, error) {
	query := `
		SELECT
			key,
			uniqExact(value) AS cardinality,
			groupArray(5)(value) AS samples,
			sum(observation_count) AS total_observations,
			min(first_seen) AS first_seen,
			max(last_seen) AS last_seen
		FROM attribute_values
	`
	
	var conditions []string
	var args []interface{}
	
	if filter != nil {
		if filter.SignalType != "" {
			conditions = append(conditions, "signal_type = ?")
			args = append(args, filter.SignalType)
		}
		if filter.Scope != "" {
			conditions = append(conditions, "scope = ?")
			args = append(args, filter.Scope)
		}
	}
	
	if len(conditions) > 0 {
		query += " WHERE " + conditions[0]
		for i := 1; i < len(conditions); i++ {
			query += " AND " + conditions[i]
		}
	}
	
	query += " GROUP BY key ORDER BY cardinality DESC"
	
	if filter != nil && filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
	}
	
	rows, err := s.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var attributes []*models.AttributeMetadata
	
	for rows.Next() {
		var (
			keyName      string
			cardinality  uint64
			samples      []string
			observations uint64
			firstSeen    time.Time
			lastSeen     time.Time
		)
		
		err := rows.Scan(&keyName, &cardinality, &samples, &observations, &firstSeen, &lastSeen)
		if err != nil {
			return nil, err
		}
		
		attr := &models.AttributeMetadata{
			Key:                  keyName,
			EstimatedCardinality: int64(cardinality),
			ValueSamples:         samples,
			Count:                int64(observations),
			FirstSeen:            firstSeen,
			LastSeen:             lastSeen,
		}
		
		attributes = append(attributes, attr)
	}
	
	return attributes, rows.Err()
}

// Service operations

func (s *Store) ListServices(ctx context.Context) ([]string, error) {
	query := "SELECT DISTINCT name FROM services FINAL ORDER BY name"
	
	rows, err := s.conn.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var services []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		services = append(services, name)
	}
	
	return services, rows.Err()
}

func (s *Store) GetServiceOverview(ctx context.Context, serviceName string) (*models.ServiceOverview, error) {
	// TODO: Implement service overview
	return &models.ServiceOverview{
		ServiceName: serviceName,
	}, nil
}

// Configuration

func (s *Store) UseAutoTemplate() bool {
	return s.autoTemplate
}

func (s *Store) AutoTemplateCfg() autotemplate.Config {
	return s.autoTmplCfg
}

// Utility operations

func (s *Store) Clear(ctx context.Context) error {
	tables := []string{"metrics", "spans", "logs", "attribute_values", "services"}
	
	for _, table := range tables {
		if err := s.conn.Exec(ctx, fmt.Sprintf("TRUNCATE TABLE %s", table)); err != nil {
			return fmt.Errorf("truncating table %s: %w", table, err)
		}
	}
	
	return nil
}

func (s *Store) Close() error {
	// Flush remaining buffer
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	if err := s.buffer.Close(ctx); err != nil {
		s.logger.Error("error flushing buffer on close", "error", err)
	}
	
	return s.conn.Close()
}
