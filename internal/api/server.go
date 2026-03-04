// Package api provides REST API handlers for querying metadata.
package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/fidde/otlp_cardinality_checker/internal/storage"
	"github.com/fidde/otlp_cardinality_checker/internal/storage/sessions"
	"github.com/fidde/otlp_cardinality_checker/pkg/models"
	"github.com/fidde/otlp_cardinality_checker/web"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Server is the REST API server.
type Server struct {
	store          storage.Storage
	router         *chi.Mux
	server         *http.Server
	sessionHandler *SessionHandler
}

// dbProvider interface for storage backends that provide direct SQL database access.
type dbProvider interface {
	DB() *sql.DB
}

// PaginationParams contains pagination parameters from query string.
type PaginationParams struct {
	Limit  int
	Offset int
}

// PaginatedResponse wraps a paginated response with metadata.
type PaginatedResponse struct {
	Data    interface{} `json:"data"`
	Total   int         `json:"total"`
	Limit   int         `json:"limit"`
	Offset  int         `json:"offset"`
	HasMore bool        `json:"has_more"`
}

// parsePaginationParams extracts pagination parameters from request.
// Defaults: limit=100, offset=0, max_limit=1000
func parsePaginationParams(r *http.Request) PaginationParams {
	const (
		defaultLimit = 100
		maxLimit     = 1000
	)

	limit := defaultLimit
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
			if limit > maxLimit {
				limit = maxLimit
			}
		}
	}

	offset := 0
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if parsed, err := strconv.Atoi(offsetStr); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	return PaginationParams{
		Limit:  limit,
		Offset: offset,
	}
}

// paginateSlice applies pagination to a slice.
func paginateSlice[T any](items []T, params PaginationParams) ([]T, PaginatedResponse) {
	total := len(items)
	start := params.Offset
	end := start + params.Limit

	// Bounds check
	if start >= total {
		return []T{}, PaginatedResponse{
			Data:    []T{},
			Total:   total,
			Limit:   params.Limit,
			Offset:  params.Offset,
			HasMore: false,
		}
	}

	if end > total {
		end = total
	}

	page := items[start:end]
	hasMore := end < total

	return page, PaginatedResponse{
		Data:    page,
		Total:   total,
		Limit:   params.Limit,
		Offset:  params.Offset,
		HasMore: hasMore,
	}
}

// NewServer creates a new API server.
func NewServer(addr string, store storage.Storage) *Server {
	s := &Server{
		store:  store,
		router: chi.NewRouter(),
	}

	// Middleware
	s.router.Use(middleware.RequestID)
	s.router.Use(middleware.RealIP)
	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)
	s.router.Use(middleware.Timeout(60 * time.Second))

	// Initialize session storage
	sessionStore, err := sessions.New()
	if err != nil {
		log.Printf("Warning: Could not initialize session storage: %v", err)
	} else {
		// Check if store implements StoreAccessor for full session support
		if storeAccessor, ok := store.(StoreAccessor); ok {
			s.sessionHandler = NewSessionHandlerWithStore(sessionStore, storeAccessor)
		} else {
			// Fallback to read-only session handler
			mainStoreGetter := func() ([]*models.MetricMetadata, []*models.SpanMetadata, []*models.LogMetadata, []*models.AttributeMetadata, []string, error) {
				ctx := context.Background()
				metrics, _ := store.ListMetrics(ctx, "")
				spans, _ := store.ListSpans(ctx, "")
				logs, _ := store.ListLogs(ctx, "")
				attrs, _ := store.ListAttributes(ctx, nil)
				services, _ := store.ListServices(ctx)
				return metrics, spans, logs, attrs, services, nil
			}
			s.sessionHandler = NewSessionHandler(sessionStore, mainStoreGetter)
		}
	}

	// API routes
	s.router.Route("/api/v1", func(r chi.Router) {
		// Health endpoint
		r.Get("/health", s.HandleHealth)

		// Version endpoint
		r.Get("/version", s.HandleVersion)

		// Metrics endpoints
		r.Get("/metrics", s.listMetrics)
		r.Get("/metrics/{name}", s.getMetric)

		// Spans endpoints
		r.Get("/spans", s.listSpans)
		r.Get("/spans/{name}", s.getSpan)
		r.Get("/span-patterns", s.getSpanPatterns)

		// Logs endpoints
		r.Get("/logs", s.listLogs)
		// IMPORTANT: More specific routes must come BEFORE generic {severity} route
		r.Get("/logs/by-service", s.listLogsByService)                                     // NEW: Service-based navigation
		r.Get("/logs/service/{service}/severity/{severity}", s.getLogByServiceAndSeverity) // NEW
		r.Get("/logs/patterns", s.getLogPatterns)
		r.Get("/logs/patterns/{severity}/{template}", s.getPatternDetails)
		r.Get("/logs/{severity}", s.getLog) // Generic route - must be last

		// Services endpoints
		r.Get("/services", s.listServices)
		r.Get("/services/{name}/overview", s.getServiceOverview)

		// Cardinality analysis endpoints
		r.Get("/cardinality/high", s.getHighCardinalityKeys)
		r.Get("/cardinality/complexity", s.getMetadataComplexity)

		// Attribute catalog endpoints
		r.Get("/attributes", s.listAttributes)
		r.Get("/attributes/{key}", s.getAttribute)

		// Deep watch endpoints — more specific routes before generic {key}
		r.Post("/attributes/{key}/watch", s.handleWatchAttribute)
		r.Delete("/attributes/{key}/watch", s.handleUnwatchAttribute)
		r.Get("/attributes/{key}/watch", s.handleGetWatchedAttribute)

		// Admin endpoints
		r.Post("/admin/clear", s.clearAllData)

		// Sessions endpoints
		if s.sessionHandler != nil {
			r.Get("/sessions", s.sessionHandler.ListSessions)
			r.Post("/sessions", s.sessionHandler.CreateSession)
			r.Get("/sessions/diff", s.sessionHandler.DiffSessions)
			r.Post("/sessions/import", s.sessionHandler.ImportSession)
			r.Get("/sessions/{name}", s.sessionHandler.GetSessionMetadata)
			r.Delete("/sessions/{name}", s.sessionHandler.DeleteSession)
			r.Post("/sessions/{name}/load", s.sessionHandler.LoadSession)
			r.Post("/sessions/{name}/merge", s.sessionHandler.MergeSession)
			r.Get("/sessions/{name}/export", s.sessionHandler.ExportSession)
		}
	})

	// Serve embedded static files with SPA fallback
	staticFS, err := web.NewStaticFileSystem()
	if err != nil {
		log.Printf("Warning: Could not load embedded UI: %v", err)
	} else {
		// Serve static files from embedded filesystem
		fileServer := http.FileServer(staticFS)

		s.router.Get("/*", func(w http.ResponseWriter, r *http.Request) {
			// Check if file exists in embedded FS
			if staticFS.Exists("", r.URL.Path) {
				fileServer.ServeHTTP(w, r)
				return
			}

			// SPA fallback: serve index.html for routes not matching static files
			f, err := staticFS.Open("/index.html")
			if err != nil {
				http.Error(w, "UI not available", http.StatusNotFound)
				return
			}
			defer f.Close()

			// Get file info for http.ServeContent
			stat, err := f.Stat()
			if err != nil {
				http.Error(w, "UI not available", http.StatusInternalServerError)
				return
			}

			http.ServeContent(w, r, "index.html", stat.ModTime(), f.(interface {
				Seek(int64, int) (int64, error)
			}).(http.File))
		})
	}

	s.server = &http.Server{
		Addr:    addr,
		Handler: s.router,
	}

	return s
}

// Start starts the API server.
func (s *Server) Start() error {
	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the API server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

// parseInt parses a string to int with default value.
func parseInt(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return val
}

// parseInt64 parses a string to int64 with default value.
func parseInt64(s string, defaultVal int64) int64 {
	if s == "" {
		return defaultVal
	}
	val, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return defaultVal
	}
	return val
}

// listMetrics returns all metrics, optionally filtered by service.
// Supports pagination via ?limit=N&offset=M query parameters.
func (s *Server) listMetrics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	serviceName := r.URL.Query().Get("service")
	params := parsePaginationParams(r)

	metrics, err := s.store.ListMetrics(ctx, serviceName)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Add convenience "type" field at top level for UI compatibility
	type MetricResponse struct {
		*models.MetricMetadata
		Type                   string `json:"type"`
		ActiveSeriesOTLP       int64  `json:"active_series_otlp"`
		ActiveSeriesPrometheus int64  `json:"active_series_prometheus"`
	}

	metricsWithType := make([]*MetricResponse, len(metrics))
	for i, m := range metrics {
		metricType := "Unknown"
		if m.Data != nil {
			metricType = m.Data.GetType()
		}
		otlpSeries := m.GetActiveSeries()
		promSeries := models.EstimatePrometheusActiveSeries(otlpSeries, m.Data)
		metricsWithType[i] = &MetricResponse{
			MetricMetadata:         m,
			Type:                   metricType,
			ActiveSeriesOTLP:       otlpSeries,
			ActiveSeriesPrometheus: promSeries,
		}
	}

	// Apply pagination
	_, response := paginateSlice(metricsWithType, params)
	s.respondJSON(w, http.StatusOK, response)
}

// getMetric returns a specific metric by name.
func (s *Server) getMetric(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	name := chi.URLParam(r, "name")

	metric, err := s.store.GetMetric(ctx, name)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			s.respondError(w, http.StatusNotFound, "metric not found")
			return
		}
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Add convenience "type" field at top level for UI compatibility
	type MetricResponse struct {
		*models.MetricMetadata
		Type                   string `json:"type"`
		ActiveSeriesOTLP       int64  `json:"active_series_otlp"`
		ActiveSeriesPrometheus int64  `json:"active_series_prometheus"`
	}

	metricType := "Unknown"
	if metric.Data != nil {
		metricType = metric.Data.GetType()
	}
	otlpSeries := metric.GetActiveSeries()
	promSeries := models.EstimatePrometheusActiveSeries(otlpSeries, metric.Data)

	response := &MetricResponse{
		MetricMetadata:         metric,
		Type:                   metricType,
		ActiveSeriesOTLP:       otlpSeries,
		ActiveSeriesPrometheus: promSeries,
	}

	s.respondJSON(w, http.StatusOK, response)
}

// listSpans returns all spans, optionally filtered by service.
// Supports pagination via ?limit=N&offset=M query parameters.
func (s *Server) listSpans(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	serviceName := r.URL.Query().Get("service")
	params := parsePaginationParams(r)

	spans, err := s.store.ListSpans(ctx, serviceName)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Apply pagination
	_, response := paginateSlice(spans, params)
	s.respondJSON(w, http.StatusOK, response)
}

// getSpan returns a specific span by name.
func (s *Server) getSpan(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	name := chi.URLParam(r, "name")

	// URL decode the span name
	decodedName, err := url.QueryUnescape(name)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid span name encoding")
		return
	}

	span, err := s.store.GetSpan(ctx, decodedName)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			s.respondError(w, http.StatusNotFound, "span not found")
			return
		}
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.respondJSON(w, http.StatusOK, span)
}

// getSpanPatterns returns aggregated span name patterns.
// GET /api/v1/span-patterns
func (s *Server) getSpanPatterns(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	patterns, err := s.store.GetSpanPatterns(ctx)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.respondJSON(w, http.StatusOK, patterns)
}

// listLogs returns all log metadata, optionally filtered by service.
// Supports pagination via ?limit=N&offset=M query parameters.
func (s *Server) listLogs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	serviceName := r.URL.Query().Get("service")
	params := parsePaginationParams(r)

	logs, err := s.store.ListLogs(ctx, serviceName)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Apply pagination
	_, response := paginateSlice(logs, params)
	s.respondJSON(w, http.StatusOK, response)
}

// getLog returns a specific log metadata by severity.
func (s *Server) getLog(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	severity := chi.URLParam(r, "severity")

	// URL decode the severity
	decodedSeverity, err := url.QueryUnescape(severity)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid severity encoding")
		return
	}

	log, err := s.store.GetLog(ctx, decodedSeverity)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			s.respondError(w, http.StatusNotFound, "log severity not found")
			return
		}
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.respondJSON(w, http.StatusOK, log)
}

// listLogsByService returns log data grouped by service_name instead of severity.
// This provides better performance when dealing with high-cardinality severities like UNSET.
func (s *Server) listLogsByService(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	params := parsePaginationParams(r)

	// Try to get database handle (works with SQLite store)
	var db *sql.DB
	if dbProv, ok := s.store.(dbProvider); ok {
		db = dbProv.DB()
	}

	// If no SQL database available (memory backend), fallback to ListLogs
	if db == nil {
		logs, err := s.store.ListLogs(ctx, "")
		if err != nil {
			s.respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		// Transform logs to service-based view
		type ServiceLogData struct {
			ServiceName string `json:"service_name"`
			Severity    string `json:"severity"`
			SampleCount int64  `json:"sample_count"`
		}

		var data []ServiceLogData
		for _, log := range logs {
			for serviceName, count := range log.Services {
				data = append(data, ServiceLogData{
					ServiceName: serviceName,
					Severity:    log.Severity,
					SampleCount: count,
				})
			}
		}

		// Apply pagination
		_, response := paginateSlice(data, params)
		s.respondJSON(w, http.StatusOK, response)
		return
	}

	// Query log_services table directly (SQLite path)
	query := `
		SELECT service_name, severity, sample_count
		FROM log_services
		ORDER BY service_name ASC, severity ASC
		LIMIT ? OFFSET ?
	`

	rows, err := db.QueryContext(ctx, query, params.Limit, params.Offset)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	type ServiceLogData struct {
		ServiceName string `json:"service_name"`
		Severity    string `json:"severity"`
		SampleCount int64  `json:"sample_count"`
	}

	var data []ServiceLogData
	for rows.Next() {
		var d ServiceLogData
		if err := rows.Scan(&d.ServiceName, &d.Severity, &d.SampleCount); err != nil {
			s.respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		data = append(data, d)
	}

	// Get total count
	var total int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM log_services").Scan(&total)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := PaginatedResponse{
		Data:    data,
		Total:   total,
		Limit:   params.Limit,
		Offset:  params.Offset,
		HasMore: params.Offset+len(data) < total,
	}

	s.respondJSON(w, http.StatusOK, response)
}

// getLogByServiceAndSeverity returns log data for a specific service and severity combination
func (s *Server) getLogByServiceAndSeverity(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	service := chi.URLParam(r, "service")
	severity := chi.URLParam(r, "severity")

	// URL decode parameters
	decodedService, err := url.QueryUnescape(service)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid service encoding")
		return
	}
	decodedSeverity, err := url.QueryUnescape(severity)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid severity encoding")
		return
	}

	// Get database handle (works with SQLite store)
	var db *sql.DB
	if dbProv, ok := s.store.(dbProvider); ok {
		db = dbProv.DB()
	}

	// If no SQL database available (memory backend), fallback to GetLog
	if db == nil {
		log, err := s.store.GetLog(ctx, decodedSeverity)
		if err != nil {
			if errors.Is(err, models.ErrNotFound) {
				s.respondError(w, http.StatusNotFound, "log severity not found")
				return
			}
			s.respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		// Check if service exists in this severity
		sampleCount, exists := log.Services[decodedService]
		if !exists {
			s.respondError(w, http.StatusNotFound, "no data found for this service and severity")
			return
		}

		// Build response from log data
		type LogServiceData struct {
			Severity      string                         `json:"severity"`
			ServiceName   string                         `json:"service_name"`
			SampleCount   int64                          `json:"sample_count"`
			BodyTemplates []*models.BodyTemplate         `json:"body_templates,omitempty"`
			AttributeKeys map[string]*models.KeyMetadata `json:"attribute_keys,omitempty"`
			ResourceKeys  map[string]*models.KeyMetadata `json:"resource_keys,omitempty"`
		}

		data := LogServiceData{
			Severity:      decodedSeverity,
			ServiceName:   decodedService,
			SampleCount:   sampleCount,
			BodyTemplates: log.BodyTemplates,
			AttributeKeys: log.AttributeKeys,
			ResourceKeys:  log.ResourceKeys,
		}

		s.respondJSON(w, http.StatusOK, data)
		return
	}

	// Query for this specific service+severity combination (SQLite path)
	type LogServiceData struct {
		Severity      string                        `json:"severity"`
		ServiceName   string                        `json:"service_name"`
		SampleCount   int64                         `json:"sample_count"`
		BodyTemplates []models.BodyTemplate         `json:"body_templates,omitempty"`
		AttributeKeys map[string]models.KeyMetadata `json:"attribute_keys,omitempty"`
		ResourceKeys  map[string]models.KeyMetadata `json:"resource_keys,omitempty"`
	}

	var data LogServiceData
	data.Severity = decodedSeverity
	data.ServiceName = decodedService

	// Get sample count
	err = db.QueryRowContext(ctx, `
		SELECT sample_count
		FROM log_services
		WHERE service_name = ? AND severity = ?
	`, decodedService, decodedSeverity).Scan(&data.SampleCount)

	if err == sql.ErrNoRows {
		s.respondError(w, http.StatusNotFound, "no data found for this service and severity")
		return
	}
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Get body templates for this service+severity
	templateRows, err := db.QueryContext(ctx, `
		SELECT template, example, count, percentage
		FROM log_body_templates
		WHERE service_name = ? AND severity = ?
		ORDER BY count DESC
		LIMIT 100
	`, decodedService, decodedSeverity)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer templateRows.Close()

	for templateRows.Next() {
		var tmpl models.BodyTemplate
		if err := templateRows.Scan(&tmpl.Template, &tmpl.Example, &tmpl.Count, &tmpl.Percentage); err != nil {
			s.respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		data.BodyTemplates = append(data.BodyTemplates, tmpl)
	}

	// Get attribute keys for this service+severity
	data.AttributeKeys = make(map[string]models.KeyMetadata)
	attrRows, err := db.QueryContext(ctx, `
		SELECT key_name, key_count, estimated_cardinality
		FROM log_service_keys
		WHERE service_name = ? AND severity = ? AND key_scope = 'attribute'
	`, decodedService, decodedSeverity)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer attrRows.Close()

	for attrRows.Next() {
		var keyName string
		var keyCount int64
		var estCard int64
		if err := attrRows.Scan(&keyName, &keyCount, &estCard); err != nil {
			s.respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		data.AttributeKeys[keyName] = models.KeyMetadata{
			Count:                keyCount,
			EstimatedCardinality: estCard,
		}
	}

	// Get resource keys for this service+severity
	data.ResourceKeys = make(map[string]models.KeyMetadata)
	resRows, err := db.QueryContext(ctx, `
		SELECT key_name, key_count, estimated_cardinality
		FROM log_service_keys
		WHERE service_name = ? AND severity = ? AND key_scope = 'resource'
	`, decodedService, decodedSeverity)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer resRows.Close()

	for resRows.Next() {
		var keyName string
		var keyCount int64
		var estCard int64
		if err := resRows.Scan(&keyName, &keyCount, &estCard); err != nil {
			s.respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		data.ResourceKeys[keyName] = models.KeyMetadata{
			Count:                keyCount,
			EstimatedCardinality: estCard,
		}
	}

	s.respondJSON(w, http.StatusOK, data)
}

// getLogPatterns returns advanced pattern analysis view grouped by service.
func (s *Server) getLogPatterns(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters with defaults
	minCount := int64(10)
	if minCountStr := r.URL.Query().Get("minCount"); minCountStr != "" {
		if parsed, err := strconv.ParseInt(minCountStr, 10, 64); err == nil && parsed > 0 {
			minCount = parsed
		}
	}

	minServices := 1
	if minServicesStr := r.URL.Query().Get("minServices"); minServicesStr != "" {
		if parsed, err := strconv.Atoi(minServicesStr); err == nil && parsed > 0 {
			minServices = parsed
		}
	}

	patterns, err := s.store.GetLogPatterns(ctx, minCount, minServices)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.respondJSON(w, http.StatusOK, patterns)
}

// getPatternDetails returns detailed information about a specific log pattern.
// This shows all unique attributes grouped by service for the given severity+template.
func (s *Server) getPatternDetails(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	severity := chi.URLParam(r, "severity")
	template := chi.URLParam(r, "template")

	// URL decode parameters
	decodedSeverity, err := url.QueryUnescape(severity)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid severity encoding")
		return
	}

	decodedTemplate, err := url.QueryUnescape(template)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid template encoding")
		return
	}

	// Get all patterns (no filters) and find the matching one
	allPatterns, err := s.store.GetLogPatterns(ctx, 0, 0)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Find the pattern that matches template and has this severity
	var matchedPattern *models.PatternGroup
	for _, pattern := range allPatterns.Patterns {
		if pattern.Template == decodedTemplate {
			// Check if this pattern has the requested severity
			if _, hasSeverity := pattern.SeverityBreakdown[decodedSeverity]; hasSeverity {
				matchedPattern = &pattern
				break
			}
		}
	}

	if matchedPattern == nil {
		s.respondError(w, http.StatusNotFound, "pattern not found for this severity")
		return
	}

	// Filter services to only show those that have this severity
	filteredServices := []models.ServicePatternInfo{}
	for _, service := range matchedPattern.Services {
		// Check if this service has logs with the requested severity
		hasSeverity := false
		for _, sev := range service.Severities {
			if sev == decodedSeverity {
				hasSeverity = true
				break
			}
		}
		if hasSeverity {
			filteredServices = append(filteredServices, service)
		}
	}

	// Build response with filtered services
	response := map[string]interface{}{
		"template":     matchedPattern.Template,
		"example_body": matchedPattern.ExampleBody,
		"severity":     decodedSeverity,
		"total_count":  matchedPattern.SeverityBreakdown[decodedSeverity],
		"services":     filteredServices,
	}

	s.respondJSON(w, http.StatusOK, response)
}

// listServices returns all service names.
func (s *Server) listServices(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	services, err := s.store.ListServices(ctx)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"data":  services,
		"total": len(services),
	})
}

// getServiceOverview returns a complete overview of telemetry for a service.
func (s *Server) getServiceOverview(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	name := chi.URLParam(r, "name")

	overview, err := s.store.GetServiceOverview(ctx, name)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.respondJSON(w, http.StatusOK, overview)
}

// getHighCardinalityKeys returns keys with high cardinality across all signal types.
// Query parameters:
//   - threshold: minimum cardinality (default: 100)
//   - limit: max results to return (default: 100, max: 1000)
func (s *Server) getHighCardinalityKeys(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse threshold parameter
	threshold := 100
	if thresholdStr := r.URL.Query().Get("threshold"); thresholdStr != "" {
		if parsed, err := strconv.Atoi(thresholdStr); err == nil && parsed > 0 {
			threshold = parsed
		}
	}

	// Parse limit parameter
	limit := 100
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
			if limit > 1000 {
				limit = 1000
			}
		}
	}

	response, err := s.store.GetHighCardinalityKeys(ctx, threshold, limit)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.respondJSON(w, http.StatusOK, response)
}

// getMetadataComplexity returns signals with high metadata complexity.
// Query parameters:
//   - threshold: minimum total key count (default: 10)
//   - limit: max results to return (default: 100, max: 1000)
func (s *Server) getMetadataComplexity(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse threshold parameter
	threshold := 10
	if thresholdStr := r.URL.Query().Get("threshold"); thresholdStr != "" {
		if parsed, err := strconv.Atoi(thresholdStr); err == nil && parsed > 0 {
			threshold = parsed
		}
	}

	// Parse limit parameter
	limit := 100
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
			if limit > 1000 {
				limit = 1000
			}
		}
	}

	response, err := s.store.GetMetadataComplexity(ctx, threshold, limit)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.respondJSON(w, http.StatusOK, response)
}

// health returns the health status of the API.
func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	s.respondJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

// respondJSON writes a JSON response.
func (s *Server) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// respondError writes an error response.
func (s *Server) respondError(w http.ResponseWriter, status int, message string) {
	s.respondJSON(w, status, map[string]string{
		"error": message,
	})
}

// listAttributes returns list of all attributes with optional filtering.
// GET /api/v1/attributes?signal_type=metric&scope=resource&sort_by=cardinality&sort_order=desc&limit=100&offset=0
func (s *Server) listAttributes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	filter := &models.AttributeFilter{
		SignalType:     r.URL.Query().Get("signal_type"),
		Scope:          r.URL.Query().Get("scope"),
		MinCardinality: parseInt64(r.URL.Query().Get("min_cardinality"), 0),
		MaxCardinality: parseInt64(r.URL.Query().Get("max_cardinality"), 0),
		SortBy:         r.URL.Query().Get("sort_by"),
		SortOrder:      r.URL.Query().Get("sort_order"),
		Limit:          parseInt(r.URL.Query().Get("limit"), 100),
		Offset:         parseInt(r.URL.Query().Get("offset"), 0),
	}

	// Get attributes from storage
	attributes, err := s.store.ListAttributes(ctx, filter)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to list attributes: %v", err))
		return
	}

	// Count total (for pagination)
	totalFilter := &models.AttributeFilter{
		SignalType:     filter.SignalType,
		Scope:          filter.Scope,
		MinCardinality: filter.MinCardinality,
		MaxCardinality: filter.MaxCardinality,
	}
	allAttributes, err := s.store.ListAttributes(ctx, totalFilter)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "Failed to count attributes")
		return
	}

	// Build watched-key set for O(1) lookup.
	watchedKeys := make(map[string]bool)
	if watched, err := s.store.ListWatchedAttributes(ctx); err == nil {
		for _, wa := range watched {
			watchedKeys[wa.Key] = true
		}
	}

	type attributeWithWatch struct {
		*models.AttributeMetadata
		Watched bool `json:"watched"`
	}

	data := make([]attributeWithWatch, len(attributes))
	for i, a := range attributes {
		data[i] = attributeWithWatch{
			AttributeMetadata: a,
			Watched:           watchedKeys[a.Key],
		}
	}

	total := len(allAttributes)
	hasMore := filter.Offset+len(attributes) < total

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"data":     data,
		"total":    total,
		"limit":    filter.Limit,
		"offset":   filter.Offset,
		"has_more": hasMore,
	})
}

// getAttribute returns details for a specific attribute key.
// GET /api/v1/attributes/{key}
func (s *Server) getAttribute(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	key := chi.URLParam(r, "key")

	if key == "" {
		s.respondError(w, http.StatusBadRequest, "Attribute key is required")
		return
	}

	// URL decode the key for dot-separated names.
	decodedKey, err := url.QueryUnescape(key)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid key encoding")
		return
	}

	attribute, err := s.store.GetAttribute(ctx, decodedKey)
	if err != nil {
		s.respondError(w, http.StatusNotFound, fmt.Sprintf("Attribute not found: %v", err))
		return
	}

	_, watchErr := s.store.GetWatchedAttribute(ctx, decodedKey)
	watched := watchErr == nil

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"data":    attribute,
		"watched": watched,
	})
}

// handleWatchAttribute enables deep watch for a specific attribute key.
// POST /api/v1/attributes/{key}/watch
// Returns 200 (already watching), 201 (newly activated), 409 (conflict), 429 (limit).
func (s *Server) handleWatchAttribute(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	key := chi.URLParam(r, "key")

	decodedKey, err := url.QueryUnescape(key)
	if err != nil || decodedKey == "" {
		s.respondError(w, http.StatusBadRequest, "invalid attribute key")
		return
	}

	if err := s.store.WatchAttribute(ctx, decodedKey); err != nil {
		if errors.Is(err, models.ErrNotFound) {
			s.respondError(w, http.StatusNotFound, err.Error())
			return
		}
		// Limit exceeded — treat as 429 Too Many Requests.
		s.respondError(w, http.StatusTooManyRequests, err.Error())
		return
	}

	s.respondJSON(w, http.StatusOK, map[string]string{
		"message": fmt.Sprintf("deep watch activated for %q", decodedKey),
		"key":     decodedKey,
	})
}

// handleUnwatchAttribute disables deep watch for a specific attribute key.
// DELETE /api/v1/attributes/{key}/watch
// Returns 204 on success, 404 if key not watched.
func (s *Server) handleUnwatchAttribute(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	key := chi.URLParam(r, "key")

	decodedKey, err := url.QueryUnescape(key)
	if err != nil || decodedKey == "" {
		s.respondError(w, http.StatusBadRequest, "invalid attribute key")
		return
	}

	if err := s.store.UnwatchAttribute(ctx, decodedKey); err != nil {
		if errors.Is(err, models.ErrNotFound) {
			s.respondError(w, http.StatusNotFound, "attribute not watched")
			return
		}
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleGetWatchedAttribute returns deep-watch data (Value Explorer) for a key.
// GET /api/v1/attributes/{key}/watch?sort_by=count&sort_direction=desc&page=1&page_size=100&q=prefix
func (s *Server) handleGetWatchedAttribute(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	key := chi.URLParam(r, "key")

	decodedKey, err := url.QueryUnescape(key)
	if err != nil || decodedKey == "" {
		s.respondError(w, http.StatusBadRequest, "invalid attribute key")
		return
	}

	wa, err := s.store.GetWatchedAttribute(ctx, decodedKey)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			s.respondError(w, http.StatusNotFound, "attribute not watched")
			return
		}
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Read query params.
	sortBy := r.URL.Query().Get("sort_by") // count | value
	if sortBy == "" {
		sortBy = "count"
	}
	sortDir := r.URL.Query().Get("sort_direction") // asc | desc
	if sortDir == "" {
		sortDir = "desc"
	}
	q := r.URL.Query().Get("q") // prefix filter
	page := parseInt(r.URL.Query().Get("page"), 1)
	pageSize := parseInt(r.URL.Query().Get("page_size"), 100)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 10000 {
		pageSize = 100
	}

	// Snapshot under lock.
	waKey, valsCopy, uniqueCount, totalObs, active, overflow, since := wa.Snapshot()

	// Build and filter value pairs.
	type ValueEntry struct {
		Value string `json:"value"`
		Count int64  `json:"count"`
	}
	entries := make([]ValueEntry, 0, len(valsCopy))
	for v, c := range valsCopy {
		if q != "" && !strings.HasPrefix(v, q) {
			continue
		}
		entries = append(entries, ValueEntry{Value: v, Count: c})
	}

	// Sort.
	sort.Slice(entries, func(i, j int) bool {
		switch sortBy {
		case "value":
			if sortDir == "asc" {
				return entries[i].Value < entries[j].Value
			}
			return entries[i].Value > entries[j].Value
		default: // count
			if sortDir == "asc" {
				return entries[i].Count < entries[j].Count
			}
			return entries[i].Count > entries[j].Count
		}
	})

	// Paginate.
	total := len(entries)
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}
	pageEntries := entries[start:end]

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"key":                waKey,
		"watching_since":     since,
		"unique_count":       uniqueCount,
		"total_observations": totalObs,
		"active":             active,
		"overflow":           overflow,
		"values":             pageEntries,
		"total_values":       total,
		"page":               page,
		"page_size":          pageSize,
	})
}

// clearAllData clears all data from the storage.
// POST /api/v1/admin/clear
func (s *Server) clearAllData(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := s.store.Clear(ctx); err != nil {
		s.respondError(w, http.StatusInternalServerError, "Failed to clear data")
		return
	}

	s.respondJSON(w, http.StatusOK, map[string]string{
		"message": "All data cleared successfully",
	})
}
