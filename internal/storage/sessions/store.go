// Package sessions provides file-based session storage for saving and loading
// telemetry metadata snapshots.
package sessions

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/fidde/otlp_cardinality_checker/pkg/models"
)

// Default configuration values
const (
	DefaultSessionDir     = "./data/sessions"
	DefaultMaxSessionSize = 100 * 1024 * 1024 // 100MB
	DefaultMaxSessions    = 50
	SessionFileExtension  = ".json.gz"
	CurrentVersion        = 1
)

// Config contains session storage configuration.
type Config struct {
	// SessionDir is the directory where sessions are stored
	SessionDir string

	// MaxSessionSize is the maximum size of a single session in bytes
	MaxSessionSize int64

	// MaxSessions is the maximum number of sessions to keep
	MaxSessions int
}

// DefaultConfig returns the default session storage configuration.
func DefaultConfig() Config {
	return Config{
		SessionDir:     getEnvOrDefault("OCC_SESSION_DIR", DefaultSessionDir),
		MaxSessionSize: getEnvInt64OrDefault("OCC_MAX_SESSION_SIZE", DefaultMaxSessionSize),
		MaxSessions:    getEnvIntOrDefault("OCC_MAX_SESSIONS", DefaultMaxSessions),
	}
}

// Store is a file-based session storage.
type Store struct {
	config Config
	mu     sync.RWMutex
}

// New creates a new session store with default configuration.
func New() (*Store, error) {
	return NewWithConfig(DefaultConfig())
}

// NewWithConfig creates a new session store with the given configuration.
func NewWithConfig(config Config) (*Store, error) {
	// Ensure session directory exists
	if err := os.MkdirAll(config.SessionDir, 0755); err != nil {
		return nil, fmt.Errorf("creating session directory: %w", err)
	}

	return &Store{
		config: config,
	}, nil
}

// Save saves a session to disk.
func (s *Store) Save(ctx context.Context, session *models.Session) error {
	if session == nil {
		return errors.New("session cannot be nil")
	}

	if err := models.ValidateSessionName(session.ID); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if we've reached the session limit
	sessions, err := s.listMetadataLocked()
	if err != nil {
		return fmt.Errorf("listing sessions: %w", err)
	}

	// Check for existing session with same name
	exists := false
	for _, meta := range sessions {
		if meta.ID == session.ID {
			exists = true
			break
		}
	}

	if !exists && len(sessions) >= s.config.MaxSessions {
		return models.ErrTooManySessions
	}

	// Set version and timestamp
	session.Version = CurrentVersion
	if session.Created.IsZero() {
		session.Created = time.Now().UTC()
	}

	// Serialize to JSON
	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("marshaling session: %w", err)
	}

	// Check size limit
	if int64(len(data)) > s.config.MaxSessionSize {
		return models.ErrSessionTooLarge
	}

	// Write to gzip file
	filePath := s.sessionPath(session.ID)
	if err := s.writeGzip(filePath, data); err != nil {
		return fmt.Errorf("writing session file: %w", err)
	}

	return nil
}

// Load loads a session from disk.
func (s *Store) Load(ctx context.Context, name string) (*models.Session, error) {
	if err := models.ValidateSessionName(name); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	filePath := s.sessionPath(name)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, models.ErrSessionNotFound
	}

	// Read gzip file
	data, err := s.readGzip(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading session file: %w", err)
	}

	// Deserialize
	var session models.Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("unmarshaling session: %w", err)
	}

	return &session, nil
}

// Delete removes a session from disk.
func (s *Store) Delete(ctx context.Context, name string) error {
	if err := models.ValidateSessionName(name); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	filePath := s.sessionPath(name)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return models.ErrSessionNotFound
	}

	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("removing session file: %w", err)
	}

	return nil
}

// List returns metadata for all saved sessions.
func (s *Store) List(ctx context.Context) ([]*models.SessionMetadata, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.listMetadataLocked()
}

// GetMetadata returns metadata for a specific session without loading full data.
func (s *Store) GetMetadata(ctx context.Context, name string) (*models.SessionMetadata, error) {
	if err := models.ValidateSessionName(name); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	filePath := s.sessionPath(name)

	// Get file info
	info, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return nil, models.ErrSessionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("stat session file: %w", err)
	}

	// Load full session to get metadata
	// TODO: Optimize by storing metadata separately
	session, err := s.Load(ctx, name)
	if err != nil {
		return nil, err
	}

	return &models.SessionMetadata{
		ID:          session.ID,
		Description: session.Description,
		Created:     session.Created,
		Signals:     session.Signals,
		SizeBytes:   info.Size(),
		Stats:       session.Stats,
	}, nil
}

// Exists checks if a session exists.
func (s *Store) Exists(ctx context.Context, name string) (bool, error) {
	if err := models.ValidateSessionName(name); err != nil {
		return false, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	filePath := s.sessionPath(name)
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// sessionPath returns the file path for a session.
func (s *Store) sessionPath(name string) string {
	return filepath.Join(s.config.SessionDir, name+SessionFileExtension)
}

// listMetadataLocked lists all session metadata (must hold lock).
func (s *Store) listMetadataLocked() ([]*models.SessionMetadata, error) {
	entries, err := os.ReadDir(s.config.SessionDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading session directory: %w", err)
	}

	var sessions []*models.SessionMetadata

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, SessionFileExtension) {
			continue
		}

		// Extract session ID from filename
		sessionID := strings.TrimSuffix(name, SessionFileExtension)

		info, err := entry.Info()
		if err != nil {
			continue // Skip files we can't stat
		}

		// Load session to get full metadata
		filePath := filepath.Join(s.config.SessionDir, name)
		data, err := s.readGzip(filePath)
		if err != nil {
			continue // Skip corrupted files
		}

		var session models.Session
		if err := json.Unmarshal(data, &session); err != nil {
			continue // Skip corrupted files
		}

		sessions = append(sessions, &models.SessionMetadata{
			ID:          sessionID,
			Description: session.Description,
			Created:     session.Created,
			Signals:     session.Signals,
			SizeBytes:   info.Size(),
			Stats:       session.Stats,
		})
	}

	// Sort by created time, newest first
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].Created.After(sessions[j].Created)
	})

	return sessions, nil
}

// writeGzip writes data to a gzip-compressed file.
func (s *Store) writeGzip(path string, data []byte) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	gw := gzip.NewWriter(file)
	defer gw.Close()

	if _, err := gw.Write(data); err != nil {
		return err
	}

	return gw.Close()
}

// readGzip reads data from a gzip-compressed file.
func (s *Store) readGzip(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	gr, err := gzip.NewReader(file)
	if err != nil {
		return nil, err
	}
	defer gr.Close()

	return io.ReadAll(gr)
}

// Helper functions for environment variable configuration

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var i int
		if _, err := fmt.Sscanf(value, "%d", &i); err == nil {
			return i
		}
	}
	return defaultValue
}

func getEnvInt64OrDefault(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		var i int64
		if _, err := fmt.Sscanf(value, "%d", &i); err == nil {
			return i
		}
	}
	return defaultValue
}
