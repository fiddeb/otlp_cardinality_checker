// Package api provides REST API handlers for querying metadata.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/fidde/otlp_cardinality_checker/internal/storage"
	"github.com/fidde/otlp_cardinality_checker/pkg/models"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Server is the REST API server.
type Server struct {
	store  storage.Storage
	router *chi.Mux
	server *http.Server
}

// PaginationParams contains pagination parameters from query string.
type PaginationParams struct {
	Limit  int
	Offset int
}

// PaginatedResponse wraps a paginated response with metadata.
type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Total      int         `json:"total"`
	Limit      int         `json:"limit"`
	Offset     int         `json:"offset"`
	HasMore    bool        `json:"has_more"`
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

	// API routes
	s.router.Route("/api/v1", func(r chi.Router) {
		// Health endpoint
		r.Get("/health", s.HandleHealth)

		// Metrics endpoints
		r.Get("/metrics", s.listMetrics)
		r.Get("/metrics/{name}", s.getMetric)

		// Spans endpoints
		r.Get("/spans", s.listSpans)
		r.Get("/spans/{name}", s.getSpan)

		// Logs endpoints
		r.Get("/logs", s.listLogs)
		r.Get("/logs/patterns", s.getLogPatterns)
		r.Get("/logs/{severity}", s.getLog)

		// Services endpoints
		r.Get("/services", s.listServices)
		r.Get("/services/{name}/overview", s.getServiceOverview)

		// Admin endpoints
		r.Post("/admin/clear", s.clearAllData)
	})

	// Serve static files from web/dist
	workDir, _ := os.Getwd()
	filesDir := http.Dir(filepath.Join(workDir, "web", "dist"))
	fileServer := http.FileServer(filesDir)
	
	// Serve static files, with SPA fallback to index.html
	s.router.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		// Try to serve the file
		path := filepath.Join(workDir, "web", "dist", r.URL.Path)
		if _, err := os.Stat(path); err == nil {
			fileServer.ServeHTTP(w, r)
			return
		}
		
		// If file doesn't exist, serve index.html for SPA routing
		http.ServeFile(w, r, filepath.Join(workDir, "web", "dist", "index.html"))
	})

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

	// Apply pagination
	_, response := paginateSlice(metrics, params)
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

	s.respondJSON(w, http.StatusOK, metric)
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
