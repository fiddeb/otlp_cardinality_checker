package api

import (
	"bytes"
	"context"
	"encoding/json"
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
