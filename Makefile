# OTLP Cardinality Checker - Build Makefile
# Ensures consistent builds for both backend and frontend

.PHONY: all build clean dev run test help ui backend install-deps

# Default target
all: build

# Help target - show available commands
help:
	@echo "Available targets:"
	@echo "  make build        - Build both backend and frontend (production)"
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
	go build -o bin/otlp-cardinality-checker ./cmd/server
	@echo "✅ Backend built: bin/otlp-cardinality-checker"

# Build frontend
ui:
	@echo "🔨 Building React frontend..."
	cd web && npm run build
	@echo "✅ Frontend built: web/dist/"

# Build both (production)
build: backend ui
	@echo "✅ Full build complete!"
	@echo "   Backend: bin/otlp-cardinality-checker"
	@echo "   Frontend: web/dist/"

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
	./bin/otlp-cardinality-checker

# Run tests
test:
	@echo "🧪 Running Go tests..."
	go test ./... -v
	@echo "🧪 Running frontend tests..."
	cd web && npm test -- --run
	@echo "✅ All tests passed"

# Clean build artifacts
clean:
	@echo "🧹 Cleaning build artifacts..."
	rm -rf bin/
	rm -rf web/dist/
	rm -rf web/node_modules/.vite/
	@echo "✅ Clean complete"

# Docker build (optional)
docker-build:
	@echo "🐳 Building Docker image..."
	docker build -t otlp-cardinality-checker:latest .
	@echo "✅ Docker image built: otlp-cardinality-checker:latest"

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
