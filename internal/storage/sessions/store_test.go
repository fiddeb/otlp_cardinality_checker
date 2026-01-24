package sessions

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fidde/otlp_cardinality_checker/pkg/models"
)

func TestStore_SaveAndLoad(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()

	config := Config{
		SessionDir:     tempDir,
		MaxSessionSize: 10 * 1024 * 1024,
		MaxSessions:    10,
	}

	store, err := NewWithConfig(config)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	ctx := context.Background()

	// Create test session
	session := &models.Session{
		ID:          "test-session",
		Description: "Test session for unit tests",
		Signals:     []string{"metrics", "spans"},
		Data: models.SessionData{
			Metrics: []*models.SerializedMetric{
				{
					Name:        "http_requests_total",
					Type:        "counter",
					LabelKeys:   map[string]*models.SerializedKey{},
					ResourceKeys: map[string]*models.SerializedKey{},
					SampleCount: 1000,
					Services:    map[string]int64{"web": 500, "api": 500},
				},
			},
		},
		Stats: models.SessionStats{
			MetricsCount: 1,
			Services:     []string{"web", "api"},
		},
	}

	// Save session
	if err := store.Save(ctx, session); err != nil {
		t.Fatalf("Failed to save session: %v", err)
	}

	// Verify file exists
	filePath := filepath.Join(tempDir, "test-session.json.gz")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("Session file was not created")
	}

	// Load session
	loaded, err := store.Load(ctx, "test-session")
	if err != nil {
		t.Fatalf("Failed to load session: %v", err)
	}

	// Verify data
	if loaded.ID != session.ID {
		t.Errorf("ID mismatch: got %s, want %s", loaded.ID, session.ID)
	}
	if loaded.Description != session.Description {
		t.Errorf("Description mismatch: got %s, want %s", loaded.Description, session.Description)
	}
	if len(loaded.Data.Metrics) != 1 {
		t.Errorf("Metrics count mismatch: got %d, want 1", len(loaded.Data.Metrics))
	}
	if loaded.Data.Metrics[0].Name != "http_requests_total" {
		t.Errorf("Metric name mismatch: got %s, want http_requests_total", loaded.Data.Metrics[0].Name)
	}
}

func TestStore_Delete(t *testing.T) {
	tempDir := t.TempDir()
	store, _ := NewWithConfig(Config{SessionDir: tempDir, MaxSessionSize: 10 * 1024 * 1024, MaxSessions: 10})
	ctx := context.Background()

	// Create and save session
	session := &models.Session{ID: "to-delete", Signals: []string{}}
	store.Save(ctx, session)

	// Delete session
	if err := store.Delete(ctx, "to-delete"); err != nil {
		t.Fatalf("Failed to delete session: %v", err)
	}

	// Verify it's gone
	_, err := store.Load(ctx, "to-delete")
	if err != models.ErrSessionNotFound {
		t.Errorf("Expected ErrSessionNotFound, got %v", err)
	}

	// Delete non-existent session
	err = store.Delete(ctx, "non-existent")
	if err != models.ErrSessionNotFound {
		t.Errorf("Expected ErrSessionNotFound for non-existent, got %v", err)
	}
}

func TestStore_List(t *testing.T) {
	tempDir := t.TempDir()
	store, _ := NewWithConfig(Config{SessionDir: tempDir, MaxSessionSize: 10 * 1024 * 1024, MaxSessions: 10})
	ctx := context.Background()

	// Create multiple sessions with different timestamps
	sessions := []*models.Session{
		{ID: "session-a", Description: "First", Signals: []string{"metrics"}, Created: time.Now().Add(-2 * time.Hour)},
		{ID: "session-b", Description: "Second", Signals: []string{"spans"}, Created: time.Now().Add(-1 * time.Hour)},
		{ID: "session-c", Description: "Third", Signals: []string{"logs"}, Created: time.Now()},
	}

	for _, s := range sessions {
		store.Save(ctx, s)
	}

	// List sessions
	listed, err := store.List(ctx)
	if err != nil {
		t.Fatalf("Failed to list sessions: %v", err)
	}

	if len(listed) != 3 {
		t.Errorf("Expected 3 sessions, got %d", len(listed))
	}

	// Verify sorted by created time (newest first)
	if listed[0].ID != "session-c" {
		t.Errorf("Expected session-c first (newest), got %s", listed[0].ID)
	}
	if listed[2].ID != "session-a" {
		t.Errorf("Expected session-a last (oldest), got %s", listed[2].ID)
	}
}

func TestStore_MaxSessions(t *testing.T) {
	tempDir := t.TempDir()
	store, _ := NewWithConfig(Config{SessionDir: tempDir, MaxSessionSize: 10 * 1024 * 1024, MaxSessions: 2})
	ctx := context.Background()

	// Create max sessions
	store.Save(ctx, &models.Session{ID: "a", Signals: []string{}})
	store.Save(ctx, &models.Session{ID: "b", Signals: []string{}})

	// Try to create one more
	err := store.Save(ctx, &models.Session{ID: "c", Signals: []string{}})
	if err != models.ErrTooManySessions {
		t.Errorf("Expected ErrTooManySessions, got %v", err)
	}

	// Updating existing session should work
	err = store.Save(ctx, &models.Session{ID: "a", Description: "updated", Signals: []string{}})
	if err != nil {
		t.Errorf("Failed to update existing session: %v", err)
	}
}

func TestStore_InvalidSessionName(t *testing.T) {
	tempDir := t.TempDir()
	store, _ := NewWithConfig(Config{SessionDir: tempDir, MaxSessionSize: 10 * 1024 * 1024, MaxSessions: 10})
	ctx := context.Background()

	tests := []string{
		"",
		"UPPERCASE",
		"with spaces",
		"special@chars",
		"-starts-with-hyphen",
		"ends-with-hyphen-",
	}

	for _, name := range tests {
		t.Run(name, func(t *testing.T) {
			err := store.Save(ctx, &models.Session{ID: name, Signals: []string{}})
			if err != models.ErrInvalidSessionName {
				t.Errorf("Expected ErrInvalidSessionName for %q, got %v", name, err)
			}
		})
	}
}

func TestStore_Exists(t *testing.T) {
	tempDir := t.TempDir()
	store, _ := NewWithConfig(Config{SessionDir: tempDir, MaxSessionSize: 10 * 1024 * 1024, MaxSessions: 10})
	ctx := context.Background()

	// Check non-existent
	exists, err := store.Exists(ctx, "non-existent")
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if exists {
		t.Error("Expected non-existent session to not exist")
	}

	// Create session
	store.Save(ctx, &models.Session{ID: "exists", Signals: []string{}})

	// Check exists
	exists, err = store.Exists(ctx, "exists")
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if !exists {
		t.Error("Expected existing session to exist")
	}
}

func TestStore_GetMetadata(t *testing.T) {
	tempDir := t.TempDir()
	store, _ := NewWithConfig(Config{SessionDir: tempDir, MaxSessionSize: 10 * 1024 * 1024, MaxSessions: 10})
	ctx := context.Background()

	session := &models.Session{
		ID:          "meta-test",
		Description: "Metadata test",
		Signals:     []string{"metrics", "spans", "logs"},
		Stats: models.SessionStats{
			MetricsCount:    10,
			SpansCount:      20,
			LogsCount:       30,
			AttributesCount: 5,
			Services:        []string{"svc1", "svc2"},
		},
	}

	store.Save(ctx, session)

	meta, err := store.GetMetadata(ctx, "meta-test")
	if err != nil {
		t.Fatalf("GetMetadata failed: %v", err)
	}

	if meta.ID != "meta-test" {
		t.Errorf("ID mismatch: got %s", meta.ID)
	}
	if meta.Description != "Metadata test" {
		t.Errorf("Description mismatch: got %s", meta.Description)
	}
	if len(meta.Signals) != 3 {
		t.Errorf("Signals count mismatch: got %d", len(meta.Signals))
	}
	if meta.Stats.MetricsCount != 10 {
		t.Errorf("MetricsCount mismatch: got %d", meta.Stats.MetricsCount)
	}
	if meta.SizeBytes <= 0 {
		t.Error("SizeBytes should be positive")
	}
}

func TestStore_GzipCompression(t *testing.T) {
	tempDir := t.TempDir()
	store, _ := NewWithConfig(Config{SessionDir: tempDir, MaxSessionSize: 10 * 1024 * 1024, MaxSessions: 10})
	ctx := context.Background()

	// Create session with substantial data
	session := &models.Session{
		ID:      "compression-test",
		Signals: []string{"metrics"},
		Data: models.SessionData{
			Metrics: make([]*models.SerializedMetric, 100),
		},
	}

	for i := 0; i < 100; i++ {
		session.Data.Metrics[i] = &models.SerializedMetric{
			Name:         "metric_" + string(rune('a'+i%26)),
			Type:         "counter",
			LabelKeys:    map[string]*models.SerializedKey{},
			ResourceKeys: map[string]*models.SerializedKey{},
			SampleCount:  int64(i * 1000),
			Services:     map[string]int64{"service": 1},
		}
	}

	store.Save(ctx, session)

	// Load and verify
	loaded, err := store.Load(ctx, "compression-test")
	if err != nil {
		t.Fatalf("Failed to load compressed session: %v", err)
	}

	if len(loaded.Data.Metrics) != 100 {
		t.Errorf("Expected 100 metrics, got %d", len(loaded.Data.Metrics))
	}
}
