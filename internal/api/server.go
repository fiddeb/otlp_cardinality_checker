// Package api provides REST API handlers for querying metadata.
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/fidde/otlp_cardinality_checker/internal/storage/memory"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Server is the REST API server.
type Server struct {
	store  *memory.Store
	router *chi.Mux
	server *http.Server
}

// NewServer creates a new API server.
func NewServer(addr string, store *memory.Store) *Server {
	s := &Server{
		store:  store,
		router: chi.NewRouter(),
	}

	// Middleware
	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)
	s.router.Use(middleware.RequestID)
	s.router.Use(middleware.Timeout(60 * time.Second))

	// Routes
	s.router.Get("/api/v1/metrics", s.listMetrics)
	s.router.Get("/api/v1/metrics/{name}", s.getMetric)
	s.router.Get("/api/v1/spans", s.listSpans)
	s.router.Get("/api/v1/spans/{name}", s.getSpan)
	s.router.Get("/api/v1/logs", s.listLogs)
	s.router.Get("/api/v1/logs/{severity}", s.getLog)
	s.router.Get("/api/v1/services", s.listServices)
	s.router.Get("/api/v1/services/{name}/overview", s.getServiceOverview)
	s.router.Get("/health", s.health)

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
func (s *Server) listMetrics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	serviceName := r.URL.Query().Get("service")

	metrics, err := s.store.ListMetrics(ctx, serviceName)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.respondJSON(w, http.StatusOK, metrics)
}

// getMetric returns a specific metric by name.
func (s *Server) getMetric(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	name := chi.URLParam(r, "name")

	metric, err := s.store.GetMetric(ctx, name)
	if err != nil {
		if err == memory.ErrNotFound {
			s.respondError(w, http.StatusNotFound, "metric not found")
			return
		}
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.respondJSON(w, http.StatusOK, metric)
}

// listSpans returns all spans, optionally filtered by service.
func (s *Server) listSpans(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	serviceName := r.URL.Query().Get("service")

	spans, err := s.store.ListSpans(ctx, serviceName)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.respondJSON(w, http.StatusOK, spans)
}

// getSpan returns a specific span by name.
func (s *Server) getSpan(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	name := chi.URLParam(r, "name")

	span, err := s.store.GetSpan(ctx, name)
	if err != nil {
		if err == memory.ErrNotFound {
			s.respondError(w, http.StatusNotFound, "span not found")
			return
		}
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.respondJSON(w, http.StatusOK, span)
}

// listLogs returns all log metadata, optionally filtered by service.
func (s *Server) listLogs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	serviceName := r.URL.Query().Get("service")

	logs, err := s.store.ListLogs(ctx, serviceName)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.respondJSON(w, http.StatusOK, logs)
}

// getLog returns log metadata for a specific severity.
func (s *Server) getLog(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	severity := chi.URLParam(r, "severity")

	log, err := s.store.GetLog(ctx, severity)
	if err != nil {
		if err == memory.ErrNotFound {
			s.respondError(w, http.StatusNotFound, "log severity not found")
			return
		}
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.respondJSON(w, http.StatusOK, log)
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
		"services": services,
		"count":    len(services),
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
