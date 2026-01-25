// Package api provides REST API handlers for querying metadata.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"

	"github.com/fidde/otlp_cardinality_checker/internal/storage/sessions"
	"github.com/fidde/otlp_cardinality_checker/pkg/models"
	"github.com/go-chi/chi/v5"
)

// StoreAccessor provides read/write access to the main telemetry store.
type StoreAccessor interface {
	// GetAll returns all metadata from the store
	GetAll(ctx context.Context) (
		metrics []*models.MetricMetadata,
		spans []*models.SpanMetadata,
		logs []*models.LogMetadata,
		attrs []*models.AttributeMetadata,
		services []string,
		err error,
	)
	// MergeMetric merges a metric into the store
	MergeMetric(ctx context.Context, metric *models.MetricMetadata) error
	// MergeSpan merges a span into the store
	MergeSpan(ctx context.Context, span *models.SpanMetadata) error
	// MergeLog merges a log into the store
	MergeLog(ctx context.Context, log *models.LogMetadata) error
	// MergeAttribute merges an attribute into the store
	MergeAttribute(ctx context.Context, attr *models.AttributeMetadata) error
	// Clear removes all data from the store
	Clear(ctx context.Context) error
}

// SessionHandler handles session-related API requests.
type SessionHandler struct {
	store        *sessions.Store
	serializer   *sessions.Serializer
	storeAccess  StoreAccessor
	// Legacy getter for backward compatibility
	mainStore func() (
		metrics []*models.MetricMetadata,
		spans []*models.SpanMetadata,
		logs []*models.LogMetadata,
		attrs []*models.AttributeMetadata,
		services []string,
		err error,
	)
}

// NewSessionHandler creates a new session handler.
func NewSessionHandler(store *sessions.Store, mainStoreGetter func() (
	[]*models.MetricMetadata,
	[]*models.SpanMetadata,
	[]*models.LogMetadata,
	[]*models.AttributeMetadata,
	[]string,
	error,
)) *SessionHandler {
	return &SessionHandler{
		store:      store,
		serializer: sessions.NewSerializer(),
		mainStore:  mainStoreGetter,
	}
}

// NewSessionHandlerWithStore creates a session handler with full store access.
func NewSessionHandlerWithStore(sessionStore *sessions.Store, storeAccess StoreAccessor) *SessionHandler {
	return &SessionHandler{
		store:       sessionStore,
		serializer:  sessions.NewSerializer(),
		storeAccess: storeAccess,
		mainStore: func() ([]*models.MetricMetadata, []*models.SpanMetadata, []*models.LogMetadata, []*models.AttributeMetadata, []string, error) {
			return storeAccess.GetAll(context.Background())
		},
	}
}

// ListSessions returns metadata for all saved sessions.
// GET /api/v1/sessions
func (h *SessionHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	sessionList, err := h.store.List(ctx)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list sessions: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"sessions": sessionList,
		"total":    len(sessionList),
	})
}

// GetSessionMetadata returns metadata for a specific session.
// GET /api/v1/sessions/{name}
func (h *SessionHandler) GetSessionMetadata(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	name := chi.URLParam(r, "name")

	decodedName, err := url.QueryUnescape(name)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid session name encoding")
		return
	}

	meta, err := h.store.GetMetadata(ctx, decodedName)
	if err != nil {
		if errors.Is(err, models.ErrSessionNotFound) {
			respondError(w, http.StatusNotFound, "Session not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to get session: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, meta)
}

// CreateSession saves the current state as a new session.
// POST /api/v1/sessions
func (h *SessionHandler) CreateSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse request body
	var opts models.SessionSaveOptions
	if err := json.NewDecoder(r.Body).Decode(&opts); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	// Validate
	if err := opts.Validate(); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Check if session already exists (unless force=true)
	forceStr := r.URL.Query().Get("force")
	force := forceStr == "true"

	exists, _ := h.store.Exists(ctx, opts.Name)
	if exists && !force {
		respondError(w, http.StatusConflict, "Session already exists. Use ?force=true to overwrite.")
		return
	}

	// Get current store state
	metrics, spans, logs, attrs, services, err := h.mainStore()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get store data: "+err.Error())
		return
	}

	// Create session
	session, err := h.serializer.CreateSession(
		ctx,
		sessions.CreateSessionOptions{
			Name:        opts.Name,
			Description: opts.Description,
			Signals:     opts.Signals,
			Services:    opts.Services,
		},
		metrics, spans, logs, attrs, services,
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create session: "+err.Error())
		return
	}

	// Save to disk
	if err := h.store.Save(ctx, session); err != nil {
		if errors.Is(err, models.ErrTooManySessions) {
			respondError(w, http.StatusConflict, "Maximum number of sessions reached")
			return
		}
		if errors.Is(err, models.ErrSessionTooLarge) {
			respondError(w, http.StatusRequestEntityTooLarge, "Session data too large")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to save session: "+err.Error())
		return
	}

	// Get metadata for response
	meta, _ := h.store.GetMetadata(ctx, opts.Name)

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"message": "Session created successfully",
		"session": meta,
	})
}

// DeleteSession removes a session.
// DELETE /api/v1/sessions/{name}
func (h *SessionHandler) DeleteSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	name := chi.URLParam(r, "name")

	decodedName, err := url.QueryUnescape(name)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid session name encoding")
		return
	}

	if err := h.store.Delete(ctx, decodedName); err != nil {
		if errors.Is(err, models.ErrSessionNotFound) {
			respondError(w, http.StatusNotFound, "Session not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to delete session: "+err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// LoadSession loads a session into the current store (replacing current data).
// POST /api/v1/sessions/{name}/load
func (h *SessionHandler) LoadSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	name := chi.URLParam(r, "name")

	decodedName, err := url.QueryUnescape(name)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid session name encoding")
		return
	}

	// Load session from disk
	session, err := h.store.Load(ctx, decodedName)
	if err != nil {
		if errors.Is(err, models.ErrSessionNotFound) {
			respondError(w, http.StatusNotFound, "Session not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to load session: "+err.Error())
		return
	}

	// If we have store access, perform actual load (clear + merge)
	if h.storeAccess != nil {
		// Clear existing data
		if err := h.storeAccess.Clear(ctx); err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to clear store: "+err.Error())
			return
		}

		// Merge session data into store
		loadedCounts, err := h.performMerge(ctx, session)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to load session data: "+err.Error())
			return
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"message": "Session loaded successfully",
			"session": session.ID,
			"loaded":  loadedCounts,
		})
		return
	}

	// Legacy: Return session data for client-side loading
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Session loaded",
		"session": session.ID,
		"stats":   session.Stats,
		"data":    session.Data,
		"action":  "replace",
	})
}

// MergeSession merges a session into the current store (additive).
// POST /api/v1/sessions/{name}/merge
func (h *SessionHandler) MergeSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	name := chi.URLParam(r, "name")

	decodedName, err := url.QueryUnescape(name)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid session name encoding")
		return
	}

	// Load session from disk
	session, err := h.store.Load(ctx, decodedName)
	if err != nil {
		if errors.Is(err, models.ErrSessionNotFound) {
			respondError(w, http.StatusNotFound, "Session not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to load session: "+err.Error())
		return
	}

	// If we have store access, perform actual merge
	if h.storeAccess != nil {
		mergedCounts, err := h.performMerge(ctx, session)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to merge session: "+err.Error())
			return
		}
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"message": "Session merged successfully",
			"session": session.ID,
			"merged":  mergedCounts,
		})
		return
	}

	// Legacy: Return session data for client-side merging
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Session ready for merge",
		"session": session.ID,
		"stats":   session.Stats,
		"data":    session.Data,
		"action":  "merge",
	})
}

// performMerge merges session data into the main store.
func (h *SessionHandler) performMerge(ctx context.Context, session *models.Session) (map[string]int, error) {
	counts := map[string]int{
		"metrics":    0,
		"spans":      0,
		"logs":       0,
		"attributes": 0,
	}

	// Merge metrics
	if len(session.Data.Metrics) > 0 {
		metrics, err := h.serializer.UnmarshalMetrics(session.Data.Metrics)
		if err != nil {
			return nil, err
		}
		for _, m := range metrics {
			if err := h.storeAccess.MergeMetric(ctx, m); err != nil {
				return nil, err
			}
			counts["metrics"]++
		}
	}

	// Merge spans
	if len(session.Data.Spans) > 0 {
		spans, err := h.serializer.UnmarshalSpans(session.Data.Spans)
		if err != nil {
			return nil, err
		}
		for _, s := range spans {
			if err := h.storeAccess.MergeSpan(ctx, s); err != nil {
				return nil, err
			}
			counts["spans"]++
		}
	}

	// Merge logs
	if len(session.Data.Logs) > 0 {
		logs, err := h.serializer.UnmarshalLogs(session.Data.Logs)
		if err != nil {
			return nil, err
		}
		for _, l := range logs {
			if err := h.storeAccess.MergeLog(ctx, l); err != nil {
				return nil, err
			}
			counts["logs"]++
		}
	}

	// Merge attributes
	if len(session.Data.Attributes) > 0 {
		attrs, err := h.serializer.UnmarshalAttributes(session.Data.Attributes)
		if err != nil {
			return nil, err
		}
		for _, a := range attrs {
			if err := h.storeAccess.MergeAttribute(ctx, a); err != nil {
				return nil, err
			}
			counts["attributes"]++
		}
	}

	return counts, nil
}

// ExportSession downloads a session as JSON.
// GET /api/v1/sessions/{name}/export
func (h *SessionHandler) ExportSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	name := chi.URLParam(r, "name")

	decodedName, err := url.QueryUnescape(name)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid session name encoding")
		return
	}

	session, err := h.store.Load(ctx, decodedName)
	if err != nil {
		if errors.Is(err, models.ErrSessionNotFound) {
			respondError(w, http.StatusNotFound, "Session not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to load session: "+err.Error())
		return
	}

	// Set headers for download
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+decodedName+".json\"")

	json.NewEncoder(w).Encode(session)
}

// ImportSession uploads a session from JSON.
// POST /api/v1/sessions/import
func (h *SessionHandler) ImportSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse session from body
	var session models.Session
	if err := json.NewDecoder(r.Body).Decode(&session); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid session JSON: "+err.Error())
		return
	}

	// Validate session ID
	if err := models.ValidateSessionName(session.ID); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid session name: "+err.Error())
		return
	}

	// Check if session exists (unless force=true)
	forceStr := r.URL.Query().Get("force")
	force := forceStr == "true"

	exists, _ := h.store.Exists(ctx, session.ID)
	if exists && !force {
		respondError(w, http.StatusConflict, "Session already exists. Use ?force=true to overwrite.")
		return
	}

	// Save session
	if err := h.store.Save(ctx, &session); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to save session: "+err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"message": "Session imported successfully",
		"session": session.ID,
	})
}

// DiffSessions compares two sessions.
// GET /api/v1/sessions/diff?from=A&to=B
func (h *SessionHandler) DiffSessions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	fromName := r.URL.Query().Get("from")
	toName := r.URL.Query().Get("to")

	if fromName == "" || toName == "" {
		respondError(w, http.StatusBadRequest, "Both 'from' and 'to' query parameters are required")
		return
	}

	// Load both sessions
	fromSession, err := h.store.Load(ctx, fromName)
	if err != nil {
		if errors.Is(err, models.ErrSessionNotFound) {
			respondError(w, http.StatusNotFound, "Source session '"+fromName+"' not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to load source session: "+err.Error())
		return
	}

	toSession, err := h.store.Load(ctx, toName)
	if err != nil {
		if errors.Is(err, models.ErrSessionNotFound) {
			respondError(w, http.StatusNotFound, "Target session '"+toName+"' not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to load target session: "+err.Error())
		return
	}

	// Compute diff
	diff := h.computeDiff(fromSession, toSession)

	// Apply severity filter if specified
	minSeverity := r.URL.Query().Get("min_severity")
	if minSeverity != "" {
		diff.Changes.Metrics.Added = filterChangesBySeverity(diff.Changes.Metrics.Added, minSeverity)
		diff.Changes.Metrics.Removed = filterChangesBySeverity(diff.Changes.Metrics.Removed, minSeverity)
		diff.Changes.Metrics.Changed = filterChangesBySeverity(diff.Changes.Metrics.Changed, minSeverity)
		diff.Changes.Spans.Added = filterChangesBySeverity(diff.Changes.Spans.Added, minSeverity)
		diff.Changes.Spans.Removed = filterChangesBySeverity(diff.Changes.Spans.Removed, minSeverity)
		diff.Changes.Spans.Changed = filterChangesBySeverity(diff.Changes.Spans.Changed, minSeverity)
		diff.Changes.Logs.Added = filterChangesBySeverity(diff.Changes.Logs.Added, minSeverity)
		diff.Changes.Logs.Removed = filterChangesBySeverity(diff.Changes.Logs.Removed, minSeverity)
		diff.Changes.Logs.Changed = filterChangesBySeverity(diff.Changes.Logs.Changed, minSeverity)
	}

	respondJSON(w, http.StatusOK, diff)
}

// computeDiff calculates the differences between two sessions.
func (h *SessionHandler) computeDiff(from, to *models.Session) *models.DiffResult {
	diff := models.NewDiffResult(from.ID, to.ID)

	// Build maps for quick lookup
	fromMetrics := make(map[string]*models.SerializedMetric)
	for _, m := range from.Data.Metrics {
		fromMetrics[m.Name] = m
	}

	toMetrics := make(map[string]*models.SerializedMetric)
	for _, m := range to.Data.Metrics {
		toMetrics[m.Name] = m
	}

	// Find added and changed metrics
	for name, toMetric := range toMetrics {
		fromMetric, exists := fromMetrics[name]
		if !exists {
			// New metric
			severity := models.SeverityInfo
			if toMetric.ActiveSeries > 1000 {
				severity = models.SeverityWarning
			}
			diff.AddChange(models.Change{
				Type:       models.ChangeTypeAdded,
				SignalType: models.SignalTypeMetric,
				Name:       name,
				Severity:   severity,
				Metadata: map[string]interface{}{
					"type":          toMetric.Type,
					"sample_count":  toMetric.SampleCount,
					"active_series": toMetric.ActiveSeries,
					"label_count":   len(toMetric.LabelKeys),
				},
			})
		} else {
			// Check for changes
			changes := h.compareMetrics(fromMetric, toMetric)
			if len(changes) > 0 {
				maxSeverity := models.SeverityInfo
				for _, c := range changes {
					maxSeverity = models.MaxSeverity(maxSeverity, c.Severity)
				}
				diff.AddChange(models.Change{
					Type:       models.ChangeTypeChanged,
					SignalType: models.SignalTypeMetric,
					Name:       name,
					Severity:   maxSeverity,
					Details:    changes,
				})
			}
		}
	}

	// Find removed metrics
	for name := range fromMetrics {
		if _, exists := toMetrics[name]; !exists {
			diff.AddChange(models.Change{
				Type:       models.ChangeTypeRemoved,
				SignalType: models.SignalTypeMetric,
				Name:       name,
				Severity:   models.SeverityInfo,
			})
		}
	}

	// Similar logic for spans
	h.diffSpans(from, to, diff)

	// Similar logic for logs
	h.diffLogs(from, to, diff)

	return diff
}

// compareMetrics compares two metrics and returns field changes.
func (h *SessionHandler) compareMetrics(from, to *models.SerializedMetric) []models.FieldChange {
	var changes []models.FieldChange

	// Compare sample count
	if to.SampleCount != from.SampleCount {
		pct := 0.0
		if from.SampleCount > 0 {
			pct = float64(to.SampleCount-from.SampleCount) / float64(from.SampleCount) * 100
		}
		changes = append(changes, models.FieldChange{
			Field:     "sample_count",
			From:      from.SampleCount,
			To:        to.SampleCount,
			ChangePct: pct,
			Severity:  models.CalculateSampleRateSeverity(from.SampleCount, to.SampleCount),
		})
	}

	// Compare active series
	if to.ActiveSeries != from.ActiveSeries {
		pct := 0.0
		if from.ActiveSeries > 0 {
			pct = float64(to.ActiveSeries-from.ActiveSeries) / float64(from.ActiveSeries) * 100
		}
		changes = append(changes, models.FieldChange{
			Field:     "active_series",
			From:      from.ActiveSeries,
			To:        to.ActiveSeries,
			ChangePct: pct,
			Severity:  models.CalculateSeverity(from.ActiveSeries, to.ActiveSeries),
		})
	}

	// Compare label keys (new keys, removed keys, cardinality changes)
	for keyName, toKey := range to.LabelKeys {
		fromKey, exists := from.LabelKeys[keyName]
		if !exists {
			// New label key
			severity := models.SeverityInfo
			if toKey.EstimatedCardinality > 1000 {
				severity = models.SeverityWarning
			}
			changes = append(changes, models.FieldChange{
				Field:    "labels." + keyName,
				From:     nil,
				To:       toKey.EstimatedCardinality,
				Severity: severity,
				Message:  "New label key added",
			})
		} else if toKey.EstimatedCardinality != fromKey.EstimatedCardinality {
			// Cardinality changed
			pct := 0.0
			if fromKey.EstimatedCardinality > 0 {
				pct = float64(toKey.EstimatedCardinality-fromKey.EstimatedCardinality) / float64(fromKey.EstimatedCardinality) * 100
			}
			changes = append(changes, models.FieldChange{
				Field:     "labels." + keyName + ".cardinality",
				From:      fromKey.EstimatedCardinality,
				To:        toKey.EstimatedCardinality,
				ChangePct: pct,
				Severity:  models.CalculateSeverity(fromKey.EstimatedCardinality, toKey.EstimatedCardinality),
			})
		}
	}

	// Find removed label keys
	for keyName := range from.LabelKeys {
		if _, exists := to.LabelKeys[keyName]; !exists {
			changes = append(changes, models.FieldChange{
				Field:    "labels." + keyName,
				From:     from.LabelKeys[keyName].EstimatedCardinality,
				To:       nil,
				Severity: models.SeverityInfo,
				Message:  "Label key removed",
			})
		}
	}

	return changes
}

// diffSpans compares spans between sessions.
func (h *SessionHandler) diffSpans(from, to *models.Session, diff *models.DiffResult) {
	fromSpans := make(map[string]*models.SerializedSpan)
	for _, s := range from.Data.Spans {
		fromSpans[s.Name] = s
	}

	toSpans := make(map[string]*models.SerializedSpan)
	for _, s := range to.Data.Spans {
		toSpans[s.Name] = s
	}

	// Find added spans
	for name, toSpan := range toSpans {
		if _, exists := fromSpans[name]; !exists {
			diff.AddChange(models.Change{
				Type:       models.ChangeTypeAdded,
				SignalType: models.SignalTypeSpan,
				Name:       name,
				Severity:   models.SeverityInfo,
				Metadata: map[string]interface{}{
					"sample_count":    toSpan.SampleCount,
					"attribute_count": len(toSpan.AttributeKeys),
				},
			})
		}
	}

	// Find removed spans
	for name := range fromSpans {
		if _, exists := toSpans[name]; !exists {
			diff.AddChange(models.Change{
				Type:       models.ChangeTypeRemoved,
				SignalType: models.SignalTypeSpan,
				Name:       name,
				Severity:   models.SeverityInfo,
			})
		}
	}

	// Find changed spans (sample count, attributes)
	for name, toSpan := range toSpans {
		fromSpan, exists := fromSpans[name]
		if !exists {
			continue
		}

		var changes []models.FieldChange

		if toSpan.SampleCount != fromSpan.SampleCount {
			pct := 0.0
			if fromSpan.SampleCount > 0 {
				pct = float64(toSpan.SampleCount-fromSpan.SampleCount) / float64(fromSpan.SampleCount) * 100
			}
			changes = append(changes, models.FieldChange{
				Field:     "sample_count",
				From:      fromSpan.SampleCount,
				To:        toSpan.SampleCount,
				ChangePct: pct,
				Severity:  models.CalculateSampleRateSeverity(fromSpan.SampleCount, toSpan.SampleCount),
			})
		}

		if len(changes) > 0 {
			maxSeverity := models.SeverityInfo
			for _, c := range changes {
				maxSeverity = models.MaxSeverity(maxSeverity, c.Severity)
			}
			diff.AddChange(models.Change{
				Type:       models.ChangeTypeChanged,
				SignalType: models.SignalTypeSpan,
				Name:       name,
				Severity:   maxSeverity,
				Details:    changes,
			})
		}
	}
}

// diffLogs compares logs between sessions.
func (h *SessionHandler) diffLogs(from, to *models.Session, diff *models.DiffResult) {
	fromLogs := make(map[string]*models.SerializedLog)
	for _, l := range from.Data.Logs {
		fromLogs[l.Severity] = l
	}

	toLogs := make(map[string]*models.SerializedLog)
	for _, l := range to.Data.Logs {
		toLogs[l.Severity] = l
	}

	// Find added logs
	for severity, toLog := range toLogs {
		if _, exists := fromLogs[severity]; !exists {
			diff.AddChange(models.Change{
				Type:       models.ChangeTypeAdded,
				SignalType: models.SignalTypeLog,
				Name:       severity,
				Severity:   models.SeverityInfo,
				Metadata: map[string]interface{}{
					"sample_count":   toLog.SampleCount,
					"template_count": len(toLog.BodyTemplates),
				},
			})
		}
	}

	// Find removed logs
	for severity := range fromLogs {
		if _, exists := toLogs[severity]; !exists {
			diff.AddChange(models.Change{
				Type:       models.ChangeTypeRemoved,
				SignalType: models.SignalTypeLog,
				Name:       severity,
				Severity:   models.SeverityInfo,
			})
		}
	}

	// Find changed logs
	for severity, toLog := range toLogs {
		fromLog, exists := fromLogs[severity]
		if !exists {
			continue
		}

		var changes []models.FieldChange

		if toLog.SampleCount != fromLog.SampleCount {
			pct := 0.0
			if fromLog.SampleCount > 0 {
				pct = float64(toLog.SampleCount-fromLog.SampleCount) / float64(fromLog.SampleCount) * 100
			}
			changes = append(changes, models.FieldChange{
				Field:     "sample_count",
				From:      fromLog.SampleCount,
				To:        toLog.SampleCount,
				ChangePct: pct,
				Severity:  models.CalculateSampleRateSeverity(fromLog.SampleCount, toLog.SampleCount),
			})
		}

		if len(changes) > 0 {
			maxSeverity := models.SeverityInfo
			for _, c := range changes {
				maxSeverity = models.MaxSeverity(maxSeverity, c.Severity)
			}
			diff.AddChange(models.Change{
				Type:       models.ChangeTypeChanged,
				SignalType: models.SignalTypeLog,
				Name:       severity,
				Severity:   maxSeverity,
				Details:    changes,
			})
		}
	}
}

// filterChangesBySeverity filters changes by minimum severity.
func filterChangesBySeverity(changes []models.Change, minSeverity string) []models.Change {
	return models.FilterBySeverity(changes, minSeverity)
}

// Helper functions for JSON responses (package-level to avoid circular deps)

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{
		"error": message,
	})
}
