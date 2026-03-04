# OTLP Cardinality Checker - Build Makefile
# Ensures consistent builds for both backend and frontend

.PHONY: all build dist clean dev run test help ui backend install-deps

# Version info — override via: make dist VERSION=v1.2.3
VERSION    ?= dev
COMMIT     ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
PKG        := github.com/fidde/otlp_cardinality_checker/internal/version
LDFLAGS    := -s -w \
              -X $(PKG).Version=$(VERSION) \
              -X $(PKG).Commit=$(COMMIT) \
              -X $(PKG).BuildDate=$(BUILD_DATE)

# Default target
all: build

# Help target - show available commands
help:
	@echo "Available targets:"
	@echo "  make build        - Build both backend and frontend (production)"
	@echo "  make dist         - Cross-compile release binaries for all platforms"
	@echo "  make backend      - Build only the Go backend"
	@echo "  make ui           - Build only the React frontend"
	@echo "  make dev          - Start development servers (backend + frontend)"
	@echo "  make run          - Run production build"
	@echo "  make test         - Run all tests"
	@echo "  make clean        - Remove build artifacts"
	@echo "  make install-deps - Install all dependencies"

# Install dependencies
install-deps:
	@echo "📦 Installing Go dependencies..."
	go mod download
	@echo "📦 Installing npm dependencies..."
	cd web && npm install
	@echo "✅ Dependencies installed"

# Build backend
backend:
	@echo "🔨 Building Go backend..."
	go build -ldflags="$(LDFLAGS)" -o bin/occ ./cmd/server
	@echo "✅ Backend built: bin/occ"

# Build frontend
ui:
	@echo "🔨 Building React frontend..."
	cd web && npm ci && npm run build
	@echo "✅ Frontend built: web/dist/"

# Build both (production) - UI must be built first so Go can embed it
build: ui backend
	@echo "✅ Full build complete!"
	@echo "   Binary with embedded UI: bin/occ"

# Development mode (runs both servers)
dev:
	@echo "🚀 Starting development servers..."
	@echo "   Backend will run on: http://localhost:8080"
	@echo "   Frontend will run on: http://localhost:5173"
	@echo ""
	@echo "Press Ctrl+C to stop both servers"
	@trap 'kill 0' EXIT; \
	(cd web && npm run dev) & \
	go run ./cmd/server

# Run production build
run: build
	@echo "🚀 Starting production server..."
	./bin/occ

# Run tests
test:
	@echo "🧪 Running Go tests..."
	go test ./... -v
	@echo "✅ All tests passed"

# Clean build artifacts
clean:
	@echo "🧹 Cleaning build artifacts..."
	rm -rf bin/
	rm -rf dist/
	rm -rf web/dist/
	rm -rf web/node_modules/.vite/
	@echo "✅ Clean complete"

# Docker build (optional)
docker-build:
	@echo "🐳 Building Docker image..."
	docker build -t occ:latest .
	@echo "✅ Docker image built: occ:latest"

# Cross-compile release binaries for all platforms
dist: ui
	@echo "🔨 Building release binaries..."
	@mkdir -p dist
	GOOS=linux   GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o dist/otlp_cardinality_checker-linux-amd64   ./cmd/server
	GOOS=linux   GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o dist/otlp_cardinality_checker-linux-arm64   ./cmd/server
	GOOS=darwin  GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o dist/otlp_cardinality_checker-darwin-amd64  ./cmd/server
	GOOS=darwin  GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o dist/otlp_cardinality_checker-darwin-arm64  ./cmd/server
	GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o dist/otlp_cardinality_checker-windows-amd64.exe ./cmd/server
	@echo "✅ Release binaries built: dist/"

# Quick rebuild (no clean)
rebuild: build

# Check if tools are installed
check-tools:
	@echo "🔍 Checking required tools..."
	@command -v go >/dev/null 2>&1 || { echo "❌ Go is not installed"; exit 1; }
	@command -v node >/dev/null 2>&1 || { echo "❌ Node.js is not installed"; exit 1; }
	@command -v npm >/dev/null 2>&1 || { echo "❌ npm is not installed"; exit 1; }
	@echo "✅ All required tools are installed"
	@echo "   Go version: $$(go version)"
	@echo "   Node version: $$(node --version)"
	@echo "   npm version: $$(npm --version)"
