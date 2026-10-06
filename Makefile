.PHONY: all build dev dev-backend dev-frontend serve start run test clean

VERSION ?= $(shell git describe --tags --always 2>/dev/null || echo "v1.2.0")
LDFLAGS := -X "local-finance/internal/updater.CurrentVersion=$(VERSION)"

# Build production frontend and single standalone Go binary
all: build

build:
	@echo "📦 Building React frontend..."
	@cd frontend && pnpm build
	@echo "🔨 Building Go binary with embedded frontend..."
	@go build -ldflags="$(LDFLAGS)" -o local-finance ./cmd/server/main.go
	@echo "✅ Single binary build complete: ./local-finance"

# Build frontend and start the Go server in one command
dev:
	@echo "📦 Building React frontend..."
	@cd frontend && pnpm build
	@echo "🚀 Starting Go server on http://127.0.0.1:8080..."
	@go run ./cmd/server/main.go -port 8080 -db ./local_finance.db

serve: dev
start: dev

# Run backend only (without rebuilding frontend)
dev-backend:
	@echo "🚀 Starting Go backend on http://127.0.0.1:8080..."
	@go run ./cmd/server/main.go -port 8080 -db ./local_finance.db

# Run frontend in dev mode with hot reload (listening on :5173 with proxy to :8080)
dev-frontend:
	@echo "⚡ Starting Vite dev server on http://localhost:5173..."
	@cd frontend && pnpm dev

# Run standalone single binary
run:
	@./local-finance -port 8080 -open

clean:
	@rm -f local-finance server local_finance.db*
	@rm -rf frontend/dist
	@echo "🧹 Cleaned up build artifacts and temporary databases."
