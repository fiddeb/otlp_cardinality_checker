// Package web provides embedded static files for the React UI.
package web

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed dist
var distFS embed.FS

// DistFS returns a filesystem rooted at the dist directory.
func DistFS() (http.FileSystem, error) {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		return nil, err
	}
	return http.FS(sub), nil
}

// StaticFileSystem implements a serve file system that checks for file existence.
type StaticFileSystem struct {
	fs http.FileSystem
}

// NewStaticFileSystem creates a new static file system from the embedded dist.
func NewStaticFileSystem() (*StaticFileSystem, error) {
	distFS, err := DistFS()
	if err != nil {
		return nil, err
	}
	return &StaticFileSystem{fs: distFS}, nil
}

// Open opens a file from the embedded filesystem.
func (s *StaticFileSystem) Open(name string) (http.File, error) {
	return s.fs.Open(name)
}

// Exists checks if a file exists in the embedded filesystem.
func (s *StaticFileSystem) Exists(prefix string, path string) bool {
	f, err := s.fs.Open(path)
	if err != nil {
		return false
	}
	_ = f.Close()
	return true
}
