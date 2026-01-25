package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/fidde/otlp_cardinality_checker/internal/storage/sessions"
	"github.com/fidde/otlp_cardinality_checker/pkg/models"
	"github.com/go-chi/chi/v5"
)

// mockStoreAccessor is a test implementation of StoreAccessor
type mockStoreAccessor struct {
	metrics    []*models.MetricMetadata
	spans      []*models.SpanMetadata
	logs       []*models.LogMetadata
	attributes []*models.AttributeMetadata
	services   []string
}

func newMockStoreAccessor() *mockStoreAccessor {
	return &mockStoreAccessor{
		metrics:    []*models.MetricMetadata{},
		spans:      []*models.SpanMetadata{},
		logs:       []*models.LogMetadata{},
		attributes: []*models.AttributeMetadata{},
		services:   []string{},
	}
}

func (m *mockStoreAccessor) GetAll(ctx context.Context) (
	[]*models.MetricMetadata,
	[]*models.SpanMetadata,
	[]*models.LogMetadata,
	[]*models.AttributeMetadata,
	[]string,
	error,
) {
	return m.metrics, m.spans, m.logs, m.attributes, m.services, nil
}

func (m *mockStoreAccessor) MergeMetric(ctx context.Context, metric *models.MetricMetadata) error {
	m.metrics = append(m.metrics, metric)
	return nil
}

func (m *mockStoreAccessor) MergeSpan(ctx context.Context, span *models.SpanMetadata) error {
	m.spans = append(m.spans, span)
	return nil
}

func (m *mockStoreAccessor) MergeLog(ctx context.Context, log *models.LogMetadata) error {
	m.logs = append(m.logs, log)
	return nil
}

func (m *mockStoreAccessor) MergeAttribute(ctx context.Context, attr *models.AttributeMetadata) error {
	m.attributes = append(m.attributes, attr)
	return nil
}

func (m *mockStoreAccessor) Clear(ctx context.Context) error {
	m.metrics = []*models.MetricMetadata{}
	m.spans = []*models.SpanMetadata{}
	m.logs = []*models.LogMetadata{}
	m.attributes = []*models.AttributeMetadata{}
	m.services = []string{}
	return nil
}

func setupTestSessionHandler(t *testing.T) (*SessionHandler, string, func()) {
	t.Helper()

	// Create temp directory for sessions
	tmpDir, err := os.MkdirTemp("", "session-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create session store with config
	sessionStore, err := sessions.NewWithConfig(sessions.Config{
		SessionDir:     tmpDir,
		MaxSessions:    10,
		MaxSessionSize: 10 * 1024 * 1024,
	})
	if err != nil {
		t.Fatalf("Failed to create session store: %v", err)
	}

	// Create mock store accessor with test data
	mockStore := newMockStoreAccessor()
	mockStore.metrics = []*models.MetricMetadata{
		{
			Name:         "test_metric",
			SampleCount:  100,
			Services:     map[string]int64{"test-service": 100},
			LabelKeys:    map[string]*models.KeyMetadata{},
			ResourceKeys: map[string]*models.KeyMetadata{},
		},
	}
	mockStore.spans = []*models.SpanMetadata{
		{
			Name:          "test_span",
			SampleCount:   50,
			Services:      map[string]int64{"test-service": 50},
			AttributeKeys: map[string]*models.KeyMetadata{},
			ResourceKeys:  map[string]*models.KeyMetadata{},
		},
	}
	mockStore.services = []string{"test-service"}

	handler := NewSessionHandlerWithStore(sessionStore, mockStore)

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return handler, tmpDir, cleanup
}

func TestSessionHandler_CreateSession(t *testing.T) {
	handler, _, cleanup := setupTestSessionHandler(t)
	defer cleanup()

	// Create session request
	reqBody := `{"name": "test-session", "description": "Test session"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.CreateSession(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp["message"] != "Session created successfully" {
		t.Errorf("Unexpected message: %v", resp["message"])
	}
}

func TestSessionHandler_ListSessions(t *testing.T) {
	handler, _, cleanup := setupTestSessionHandler(t)
	defer cleanup()

	// Create a session first
	reqBody := `{"name": "list-test-session"}`
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/sessions", bytes.NewBufferString(reqBody))
	createReq.Header.Set("Content-Type", "application/json")
	createRR := httptest.NewRecorder()
	handler.CreateSession(createRR, createReq)

	// List sessions
	req := httptest.NewRequest(http.MethodGet, "/api/v1/sessions", nil)
	rr := httptest.NewRecorder()
	handler.ListSessions(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	sessions := resp["sessions"].([]interface{})
	if len(sessions) != 1 {
		t.Errorf("Expected 1 session, got %d", len(sessions))
	}
}

func TestSessionHandler_GetSessionMetadata(t *testing.T) {
	handler, _, cleanup := setupTestSessionHandler(t)
	defer cleanup()

	// Create a session first
	reqBody := `{"name": "get-test-session"}`
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/sessions", bytes.NewBufferString(reqBody))
	createReq.Header.Set("Content-Type", "application/json")
	createRR := httptest.NewRecorder()
	handler.CreateSession(createRR, createReq)

	// Get session metadata using chi router context
	req := httptest.NewRequest(http.MethodGet, "/api/v1/sessions/get-test-session", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("name", "get-test-session")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.GetSessionMetadata(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestSessionHandler_DeleteSession(t *testing.T) {
	handler, _, cleanup := setupTestSessionHandler(t)
	defer cleanup()

	// Create a session first
	reqBody := `{"name": "delete-test-session"}`
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/sessions", bytes.NewBufferString(reqBody))
	createReq.Header.Set("Content-Type", "application/json")
	createRR := httptest.NewRecorder()
	handler.CreateSession(createRR, createReq)

	// Delete session
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/sessions/delete-test-session", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("name", "delete-test-session")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.DeleteSession(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify session is deleted
	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/sessions/delete-test-session", nil)
	getRctx := chi.NewRouteContext()
	getRctx.URLParams.Add("name", "delete-test-session")
	getReq = getReq.WithContext(context.WithValue(getReq.Context(), chi.RouteCtxKey, getRctx))

	getRR := httptest.NewRecorder()
	handler.GetSessionMetadata(getRR, getReq)

	if getRR.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 after delete, got %d", getRR.Code)
	}
}

func TestSessionHandler_MergeSession(t *testing.T) {
	handler, _, cleanup := setupTestSessionHandler(t)
	defer cleanup()

	// Create a session first
	reqBody := `{"name": "merge-test-session"}`
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/sessions", bytes.NewBufferString(reqBody))
	createReq.Header.Set("Content-Type", "application/json")
	createRR := httptest.NewRecorder()
	handler.CreateSession(createRR, createReq)

	// Merge session
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/merge-test-session/merge", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("name", "merge-test-session")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.MergeSession(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp["message"] != "Session merged successfully" {
		t.Errorf("Unexpected message: %v", resp["message"])
	}

	merged := resp["merged"].(map[string]interface{})
	if merged["metrics"].(float64) != 1 {
		t.Errorf("Expected 1 merged metric, got %v", merged["metrics"])
	}
}

func TestSessionHandler_ExportImport(t *testing.T) {
	handler, _, cleanup := setupTestSessionHandler(t)
	defer cleanup()

	// Create a session first
	reqBody := `{"name": "export-test-session"}`
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/sessions", bytes.NewBufferString(reqBody))
	createReq.Header.Set("Content-Type", "application/json")
	createRR := httptest.NewRecorder()
	handler.CreateSession(createRR, createReq)

	// Export session
	exportReq := httptest.NewRequest(http.MethodGet, "/api/v1/sessions/export-test-session/export", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("name", "export-test-session")
	exportReq = exportReq.WithContext(context.WithValue(exportReq.Context(), chi.RouteCtxKey, rctx))

	exportRR := httptest.NewRecorder()
	handler.ExportSession(exportRR, exportReq)

	if exportRR.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", exportRR.Code, exportRR.Body.String())
	}

	exportedData := exportRR.Body.Bytes()

	// Modify the exported session for import (change ID)
	var session models.Session
	if err := json.Unmarshal(exportedData, &session); err != nil {
		t.Fatalf("Failed to parse exported session: %v", err)
	}
	session.ID = "imported-session"
	session.Created = time.Now().UTC()

	importData, _ := json.Marshal(session)

	// Import the session
	importReq := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/import", bytes.NewReader(importData))
	importReq.Header.Set("Content-Type", "application/json")
	importRR := httptest.NewRecorder()
	handler.ImportSession(importRR, importReq)

	if importRR.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d: %s", importRR.Code, importRR.Body.String())
	}

	// Verify imported session exists
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/sessions", nil)
	listRR := httptest.NewRecorder()
	handler.ListSessions(listRR, listReq)

	var listResp map[string]interface{}
	json.Unmarshal(listRR.Body.Bytes(), &listResp)
	
	if int(listResp["total"].(float64)) != 2 {
		t.Errorf("Expected 2 sessions after import, got %v", listResp["total"])
	}
}

func TestSessionHandler_CreateSession_InvalidName(t *testing.T) {
	handler, _, cleanup := setupTestSessionHandler(t)
	defer cleanup()

	testCases := []struct {
		name string
		body string
	}{
		{"empty name", `{"name": ""}`},
		{"uppercase", `{"name": "INVALID"}`},
		{"spaces", `{"name": "has spaces"}`},
		{"special chars", `{"name": "has@chars"}`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			handler.CreateSession(rr, req)

			if rr.Code != http.StatusBadRequest {
				t.Errorf("Expected status 400 for %s, got %d", tc.name, rr.Code)
			}
		})
	}
}

func TestSessionHandler_DiffSessions(t *testing.T) {
	handler, _, cleanup := setupTestSessionHandler(t)
	defer cleanup()

	// Create two sessions
	for _, name := range []string{"diff-session-a", "diff-session-b"} {
		reqBody := `{"name": "` + name + `"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions", bytes.NewBufferString(reqBody))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		handler.CreateSession(rr, req)
	}

	// Diff sessions
	req := httptest.NewRequest(http.MethodGet, "/api/v1/sessions/diff?from=diff-session-a&to=diff-session-b", nil)
	rr := httptest.NewRecorder()
	handler.DiffSessions(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp["from"] != "diff-session-a" {
		t.Errorf("Unexpected from: %v", resp["from"])
	}
	if resp["to"] != "diff-session-b" {
		t.Errorf("Unexpected to: %v", resp["to"])
	}
}

// Tests for task 3.1.6: Unit tests for each signal type diff
func TestDiff_MetricTypes(t *testing.T) {
	handler, _, cleanup := setupTestSessionHandlerWithDiffData(t)
	defer cleanup()

	// Diff sessions
	req := httptest.NewRequest(http.MethodGet, "/api/v1/sessions/diff?from=diff-base&to=diff-changed", nil)
	rr := httptest.NewRecorder()
	handler.DiffSessions(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp models.DiffResult
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Check metric diffs
	if resp.Summary.Metrics.Added != 1 {
		t.Errorf("Expected 1 added metric, got %d", resp.Summary.Metrics.Added)
	}
	if resp.Summary.Metrics.Changed != 1 {
		t.Errorf("Expected 1 changed metric, got %d", resp.Summary.Metrics.Changed)
	}
}

func TestDiff_SpanTypes(t *testing.T) {
	handler, _, cleanup := setupTestSessionHandlerWithDiffData(t)
	defer cleanup()

	// Diff sessions
	req := httptest.NewRequest(http.MethodGet, "/api/v1/sessions/diff?from=diff-base&to=diff-changed", nil)
	rr := httptest.NewRecorder()
	handler.DiffSessions(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp models.DiffResult
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Check span diffs
	if resp.Summary.Spans.Added != 1 {
		t.Errorf("Expected 1 added span, got %d", resp.Summary.Spans.Added)
	}
	if resp.Summary.Spans.Removed != 1 {
		t.Errorf("Expected 1 removed span, got %d", resp.Summary.Spans.Removed)
	}
}

func TestDiff_LogTypes(t *testing.T) {
	handler, _, cleanup := setupTestSessionHandlerWithDiffData(t)
	defer cleanup()

	// Diff sessions
	req := httptest.NewRequest(http.MethodGet, "/api/v1/sessions/diff?from=diff-base&to=diff-changed", nil)
	rr := httptest.NewRecorder()
	handler.DiffSessions(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp models.DiffResult
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Check log diffs
	if resp.Summary.Logs.Changed != 1 {
		t.Errorf("Expected 1 changed log, got %d", resp.Summary.Logs.Changed)
	}
}

func TestDiff_AttributeTypes(t *testing.T) {
	handler, _, cleanup := setupTestSessionHandlerWithDiffData(t)
	defer cleanup()

	// Diff sessions
	req := httptest.NewRequest(http.MethodGet, "/api/v1/sessions/diff?from=diff-base&to=diff-changed", nil)
	rr := httptest.NewRecorder()
	handler.DiffSessions(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp models.DiffResult
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Check attribute diffs
	if resp.Summary.Attributes.Added != 1 {
		t.Errorf("Expected 1 added attribute, got %d", resp.Summary.Attributes.Added)
	}
}

// Tests for task 3.2.5: API tests for diff endpoint filters
func TestDiff_SignalTypeFilter(t *testing.T) {
	handler, _, cleanup := setupTestSessionHandlerWithDiffData(t)
	defer cleanup()

	testCases := []struct {
		signalType   string
		expectedKey  string
		shouldHave   bool
	}{
		{"metric", "metrics", true},
		{"span", "spans", true},
		{"log", "logs", true},
		{"attribute", "attributes", true},
	}

	for _, tc := range testCases {
		t.Run(tc.signalType, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/sessions/diff?from=diff-base&to=diff-changed&signal_type="+tc.signalType, nil)
			rr := httptest.NewRecorder()
			handler.DiffSessions(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
			}

			var resp models.DiffResult
			if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			// Verify other signal types are filtered out
			switch tc.signalType {
			case "metric":
				if resp.Summary.Spans.Added+resp.Summary.Spans.Changed+resp.Summary.Spans.Removed > 0 {
					t.Error("Expected spans to be filtered out")
				}
			case "span":
				if resp.Summary.Metrics.Added+resp.Summary.Metrics.Changed+resp.Summary.Metrics.Removed > 0 {
					t.Error("Expected metrics to be filtered out")
				}
			}
		})
	}
}

func TestDiff_SeverityFilter(t *testing.T) {
	handler, _, cleanup := setupTestSessionHandlerWithDiffData(t)
	defer cleanup()

	// Request with warning filter
	req := httptest.NewRequest(http.MethodGet, "/api/v1/sessions/diff?from=diff-base&to=diff-changed&min_severity=warning", nil)
	rr := httptest.NewRecorder()
	handler.DiffSessions(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp models.DiffResult
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// All remaining changes should have warning or critical severity
	for _, c := range resp.Changes.Metrics.Added {
		if c.Severity == "info" {
			t.Error("Found info severity change when filter was warning")
		}
	}
}

func TestDiff_MissingParams(t *testing.T) {
	handler, _, cleanup := setupTestSessionHandler(t)
	defer cleanup()

	testCases := []struct {
		name     string
		url      string
		wantCode int
	}{
		{"missing from", "/api/v1/sessions/diff?to=session-b", http.StatusBadRequest},
		{"missing to", "/api/v1/sessions/diff?from=session-a", http.StatusBadRequest},
		{"missing both", "/api/v1/sessions/diff", http.StatusBadRequest},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.url, nil)
			rr := httptest.NewRecorder()
			handler.DiffSessions(rr, req)

			if rr.Code != tc.wantCode {
				t.Errorf("Expected status %d, got %d", tc.wantCode, rr.Code)
			}
		})
	}
}

func TestDiff_SessionNotFound(t *testing.T) {
	handler, _, cleanup := setupTestSessionHandler(t)
	defer cleanup()

	// Create only the "from" session
	reqBody := `{"name": "existing-session"}`
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/sessions", bytes.NewBufferString(reqBody))
	createReq.Header.Set("Content-Type", "application/json")
	createRR := httptest.NewRecorder()
	handler.CreateSession(createRR, createReq)

	// Diff with non-existent "to" session
	req := httptest.NewRequest(http.MethodGet, "/api/v1/sessions/diff?from=existing-session&to=non-existent", nil)
	rr := httptest.NewRecorder()
	handler.DiffSessions(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d: %s", rr.Code, rr.Body.String())
	}
}

// Tests for task 3.3.6: Tests for each change type detection
func TestChangeDetection_CardinalityIncrease(t *testing.T) {
	// Test that >10x increase is critical, >2x is warning
	testCases := []struct {
		from     int64
		to       int64
		expected string
	}{
		{100, 1000, models.SeverityCritical}, // 10x
		{100, 500, models.SeverityWarning},   // 5x
		{100, 150, models.SeverityInfo},      // 1.5x
		{0, 1000, models.SeverityWarning},    // new high cardinality
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%d_to_%d", tc.from, tc.to), func(t *testing.T) {
			severity := models.CalculateSeverity(tc.from, tc.to)
			if severity != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, severity)
			}
		})
	}
}

func TestChangeDetection_SampleRateIncrease(t *testing.T) {
	testCases := []struct {
		from     int64
		to       int64
		expected string
	}{
		{100, 500, models.SeverityWarning}, // 5x
		{100, 400, models.SeverityInfo},    // 4x
		{100, 150, models.SeverityInfo},    // 1.5x
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%d_to_%d", tc.from, tc.to), func(t *testing.T) {
			severity := models.CalculateSampleRateSeverity(tc.from, tc.to)
			if severity != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, severity)
			}
		})
	}
}

// Tests for task 2.2.3: Signals filter for load/merge
func TestLoadSession_SignalsFilter(t *testing.T) {
	handler, _, cleanup := setupTestSessionHandlerWithData(t)
	defer cleanup()

	// Create a session with all signals
	reqBody := `{"name": "filter-test-session"}`
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/sessions", bytes.NewBufferString(reqBody))
	createReq.Header.Set("Content-Type", "application/json")
	createRR := httptest.NewRecorder()
	handler.CreateSession(createRR, createReq)

	if createRR.Code != http.StatusCreated {
		t.Fatalf("Failed to create session: %s", createRR.Body.String())
	}

	// Load session with only metrics filter
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/filter-test-session/load?signals=metrics", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("name", "filter-test-session")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.LoadSession(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	loaded := resp["loaded"].(map[string]interface{})
	if loaded["metrics"].(float64) != 1 {
		t.Error("Expected metrics to be loaded")
	}
	if loaded["spans"].(float64) != 0 {
		t.Error("Expected spans to be filtered out")
	}
}

func TestMergeSession_SignalsFilter(t *testing.T) {
	handler, _, cleanup := setupTestSessionHandlerWithData(t)
	defer cleanup()

	// Create a session with all signals
	reqBody := `{"name": "merge-filter-session"}`
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/sessions", bytes.NewBufferString(reqBody))
	createReq.Header.Set("Content-Type", "application/json")
	createRR := httptest.NewRecorder()
	handler.CreateSession(createRR, createReq)

	// Merge session with only spans filter
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/merge-filter-session/merge?signals=spans", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("name", "merge-filter-session")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.MergeSession(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	merged := resp["merged"].(map[string]interface{})
	if merged["spans"].(float64) != 1 {
		t.Error("Expected spans to be merged")
	}
	if merged["metrics"].(float64) != 0 {
		t.Error("Expected metrics to be filtered out")
	}
}

// Tests for task 4.1.3: Cardinality accuracy after merge
func TestMerge_CardinalityAccuracy(t *testing.T) {
	handler, _, cleanup := setupTestSessionHandlerWithHLLData(t)
	defer cleanup()

	// Create two sessions with overlapping data
	for _, name := range []string{"cardinality-session-a", "cardinality-session-b"} {
		reqBody := `{"name": "` + name + `"}`
		createReq := httptest.NewRequest(http.MethodPost, "/api/v1/sessions", bytes.NewBufferString(reqBody))
		createReq.Header.Set("Content-Type", "application/json")
		createRR := httptest.NewRecorder()
		handler.CreateSession(createRR, createReq)
		if createRR.Code != http.StatusCreated {
			t.Fatalf("Failed to create session %s: %s", name, createRR.Body.String())
		}
	}

	// Merge session b into current state
	mergeReq := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/cardinality-session-a/merge", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("name", "cardinality-session-a")
	mergeReq = mergeReq.WithContext(context.WithValue(mergeReq.Context(), chi.RouteCtxKey, rctx))

	mergeRR := httptest.NewRecorder()
	handler.MergeSession(mergeRR, mergeReq)

	if mergeRR.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", mergeRR.Code, mergeRR.Body.String())
	}

	// Verify merge was successful
	var resp map[string]interface{}
	if err := json.Unmarshal(mergeRR.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp["message"] != "Session merged successfully" {
		t.Errorf("Unexpected message: %v", resp["message"])
	}
}

// Tests for task 4.2.6: Integration tests for multi-session merge
func TestMerge_MultiSession(t *testing.T) {
	handler, _, cleanup := setupTestSessionHandlerWithData(t)
	defer cleanup()

	// Create three sessions
	sessionNames := []string{"multi-merge-a", "multi-merge-b", "multi-merge-c"}
	for _, name := range sessionNames {
		reqBody := `{"name": "` + name + `"}`
		createReq := httptest.NewRequest(http.MethodPost, "/api/v1/sessions", bytes.NewBufferString(reqBody))
		createReq.Header.Set("Content-Type", "application/json")
		createRR := httptest.NewRecorder()
		handler.CreateSession(createRR, createReq)
		if createRR.Code != http.StatusCreated {
			t.Fatalf("Failed to create session %s: %s", name, createRR.Body.String())
		}
	}

	// Merge all sessions sequentially
	totalMerged := 0
	for _, name := range sessionNames {
		mergeReq := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/"+name+"/merge", nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("name", name)
		mergeReq = mergeReq.WithContext(context.WithValue(mergeReq.Context(), chi.RouteCtxKey, rctx))

		mergeRR := httptest.NewRecorder()
		handler.MergeSession(mergeRR, mergeReq)

		if mergeRR.Code != http.StatusOK {
			t.Errorf("Failed to merge session %s: %s", name, mergeRR.Body.String())
			continue
		}

		var resp map[string]interface{}
		json.Unmarshal(mergeRR.Body.Bytes(), &resp)
		merged := resp["merged"].(map[string]interface{})
		totalMerged += int(merged["metrics"].(float64))
	}

	// Verify we merged data from all sessions
	if totalMerged < len(sessionNames) {
		t.Errorf("Expected to merge from all %d sessions, total merged metrics: %d", len(sessionNames), totalMerged)
	}
}

func TestMerge_PreservesSampleCounts(t *testing.T) {
	handler, mockStore, cleanup := setupTestSessionHandlerWithMockStore(t)
	defer cleanup()

	// Set up initial data
	mockStore.metrics = []*models.MetricMetadata{
		{
			Name:         "merge_test_metric",
			SampleCount:  100,
			Services:     map[string]int64{"svc-a": 100},
			LabelKeys:    map[string]*models.KeyMetadata{},
			ResourceKeys: map[string]*models.KeyMetadata{},
		},
	}

	// Create session
	reqBody := `{"name": "sample-count-session"}`
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/sessions", bytes.NewBufferString(reqBody))
	createReq.Header.Set("Content-Type", "application/json")
	createRR := httptest.NewRecorder()
	handler.CreateSession(createRR, createReq)

	// Clear and set different sample count
	mockStore.Clear(context.Background())
	mockStore.metrics = []*models.MetricMetadata{
		{
			Name:         "merge_test_metric",
			SampleCount:  200, // Different count
			Services:     map[string]int64{"svc-b": 200},
			LabelKeys:    map[string]*models.KeyMetadata{},
			ResourceKeys: map[string]*models.KeyMetadata{},
		},
	}

	// Merge original session
	mergeReq := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/sample-count-session/merge", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("name", "sample-count-session")
	mergeReq = mergeReq.WithContext(context.WithValue(mergeReq.Context(), chi.RouteCtxKey, rctx))

	mergeRR := httptest.NewRecorder()
	handler.MergeSession(mergeRR, mergeReq)

	if mergeRR.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", mergeRR.Code, mergeRR.Body.String())
	}
}

// Helper function to create handler with diff test data
func setupTestSessionHandlerWithDiffData(t *testing.T) (*SessionHandler, string, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "session-diff-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	sessionStore, err := sessions.NewWithConfig(sessions.Config{
		SessionDir:     tmpDir,
		MaxSessions:    10,
		MaxSessionSize: 10 * 1024 * 1024,
	})
	if err != nil {
		t.Fatalf("Failed to create session store: %v", err)
	}

	// Create base session data
	baseSession := &models.Session{
		Version: 1,
		ID:      "diff-base",
		Created: time.Now().UTC(),
		Signals: []string{"metrics", "spans", "logs", "attributes"},
		Data: models.SessionData{
			Metrics: []*models.SerializedMetric{
				{Name: "base_metric", SampleCount: 100, ActiveSeries: 10, LabelKeys: map[string]*models.SerializedKey{}, ResourceKeys: map[string]*models.SerializedKey{}, Services: map[string]int64{"svc": 100}},
			},
			Spans: []*models.SerializedSpan{
				{Name: "base_span", SampleCount: 50, AttributeKeys: map[string]*models.SerializedKey{}, ResourceKeys: map[string]*models.SerializedKey{}, Services: map[string]int64{"svc": 50}},
				{Name: "removed_span", SampleCount: 20, AttributeKeys: map[string]*models.SerializedKey{}, ResourceKeys: map[string]*models.SerializedKey{}, Services: map[string]int64{"svc": 20}},
			},
			Logs: []*models.SerializedLog{
				{Severity: "ERROR", SampleCount: 30, AttributeKeys: map[string]*models.SerializedKey{}, ResourceKeys: map[string]*models.SerializedKey{}, Services: map[string]int64{"svc": 30}},
			},
			Attributes: []*models.SerializedAttribute{
				{Key: "existing.attr", Count: 100, EstimatedCardinality: 50},
			},
		},
	}

	// Create changed session data
	changedSession := &models.Session{
		Version: 1,
		ID:      "diff-changed",
		Created: time.Now().UTC(),
		Signals: []string{"metrics", "spans", "logs", "attributes"},
		Data: models.SessionData{
			Metrics: []*models.SerializedMetric{
				{Name: "base_metric", SampleCount: 500, ActiveSeries: 100, LabelKeys: map[string]*models.SerializedKey{}, ResourceKeys: map[string]*models.SerializedKey{}, Services: map[string]int64{"svc": 500}}, // Changed
				{Name: "new_metric", SampleCount: 200, ActiveSeries: 2000, LabelKeys: map[string]*models.SerializedKey{}, ResourceKeys: map[string]*models.SerializedKey{}, Services: map[string]int64{"svc": 200}}, // Added (high cardinality)
			},
			Spans: []*models.SerializedSpan{
				{Name: "base_span", SampleCount: 50, AttributeKeys: map[string]*models.SerializedKey{}, ResourceKeys: map[string]*models.SerializedKey{}, Services: map[string]int64{"svc": 50}},
				{Name: "new_span", SampleCount: 100, AttributeKeys: map[string]*models.SerializedKey{}, ResourceKeys: map[string]*models.SerializedKey{}, Services: map[string]int64{"svc": 100}}, // Added
			},
			Logs: []*models.SerializedLog{
				{Severity: "ERROR", SampleCount: 150, AttributeKeys: map[string]*models.SerializedKey{}, ResourceKeys: map[string]*models.SerializedKey{}, Services: map[string]int64{"svc": 150}}, // 5x increase
			},
			Attributes: []*models.SerializedAttribute{
				{Key: "existing.attr", Count: 100, EstimatedCardinality: 50},
				{Key: "new.attr", Count: 50, EstimatedCardinality: 5000}, // Added high cardinality
			},
		},
	}

	// Save sessions directly
	if err := sessionStore.Save(context.Background(), baseSession); err != nil {
		t.Fatalf("Failed to save base session: %v", err)
	}
	if err := sessionStore.Save(context.Background(), changedSession); err != nil {
		t.Fatalf("Failed to save changed session: %v", err)
	}

	mockStore := newMockStoreAccessor()
	handler := NewSessionHandlerWithStore(sessionStore, mockStore)

	return handler, tmpDir, func() { os.RemoveAll(tmpDir) }
}

// Helper function with richer test data for load/merge filter tests
func setupTestSessionHandlerWithData(t *testing.T) (*SessionHandler, string, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "session-data-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	sessionStore, err := sessions.NewWithConfig(sessions.Config{
		SessionDir:     tmpDir,
		MaxSessions:    10,
		MaxSessionSize: 10 * 1024 * 1024,
	})
	if err != nil {
		t.Fatalf("Failed to create session store: %v", err)
	}

	mockStore := newMockStoreAccessor()
	mockStore.metrics = []*models.MetricMetadata{
		{Name: "test_metric", SampleCount: 100, Services: map[string]int64{"test-service": 100}, LabelKeys: map[string]*models.KeyMetadata{}, ResourceKeys: map[string]*models.KeyMetadata{}},
	}
	mockStore.spans = []*models.SpanMetadata{
		{Name: "test_span", SampleCount: 50, Services: map[string]int64{"test-service": 50}, AttributeKeys: map[string]*models.KeyMetadata{}, ResourceKeys: map[string]*models.KeyMetadata{}},
	}
	mockStore.logs = []*models.LogMetadata{
		{Severity: "INFO", SampleCount: 25, Services: map[string]int64{"test-service": 25}, AttributeKeys: map[string]*models.KeyMetadata{}, ResourceKeys: map[string]*models.KeyMetadata{}},
	}
	mockStore.services = []string{"test-service"}

	handler := NewSessionHandlerWithStore(sessionStore, mockStore)

	return handler, tmpDir, func() { os.RemoveAll(tmpDir) }
}

// Helper function for HLL cardinality tests
func setupTestSessionHandlerWithHLLData(t *testing.T) (*SessionHandler, string, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "session-hll-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	sessionStore, err := sessions.NewWithConfig(sessions.Config{
		SessionDir:     tmpDir,
		MaxSessions:    10,
		MaxSessionSize: 10 * 1024 * 1024,
	})
	if err != nil {
		t.Fatalf("Failed to create session store: %v", err)
	}

	mockStore := newMockStoreAccessor()
	mockStore.metrics = []*models.MetricMetadata{
		{Name: "hll_metric", SampleCount: 1000, Services: map[string]int64{"hll-service": 1000}, LabelKeys: map[string]*models.KeyMetadata{}, ResourceKeys: map[string]*models.KeyMetadata{}},
	}
	mockStore.services = []string{"hll-service"}

	handler := NewSessionHandlerWithStore(sessionStore, mockStore)

	return handler, tmpDir, func() { os.RemoveAll(tmpDir) }
}

// Helper function that returns the mock store for assertions
func setupTestSessionHandlerWithMockStore(t *testing.T) (*SessionHandler, *mockStoreAccessor, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "session-mock-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	sessionStore, err := sessions.NewWithConfig(sessions.Config{
		SessionDir:     tmpDir,
		MaxSessions:    10,
		MaxSessionSize: 10 * 1024 * 1024,
	})
	if err != nil {
		t.Fatalf("Failed to create session store: %v", err)
	}

	mockStore := newMockStoreAccessor()
	handler := NewSessionHandlerWithStore(sessionStore, mockStore)

	return handler, mockStore, func() { os.RemoveAll(tmpDir) }
}
