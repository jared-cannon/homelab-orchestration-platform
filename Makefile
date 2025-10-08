.PHONY: help dev build test test-coverage-html clean frontend-build frontend-dev backend-dev backend-test types docker install-deps

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

dev: ## Run development servers (backend + frontend)
	@echo "🚀 Starting development servers..."
	@command -v goreman >/dev/null 2>&1 || { echo "Installing goreman..."; go install github.com/mattn/goreman@latest; }
	@$(shell go env GOPATH)/bin/goreman start

frontend-dev: ## Run frontend dev server only
	@echo "🎨 Starting frontend dev server..."
	@cd frontend && npm run dev

backend-dev: ## Run backend dev server only
	@echo "⚙️  Starting backend dev server..."
	@cd backend && go run cmd/server/main.go

backend-test: ## Run backend tests
	@echo "🧪 Running backend tests..."
	@cd backend && go test -v ./...

test: ## Run all tests with coverage
	@echo "🧪 Running tests with coverage..."
	@cd backend && go test ./... -coverprofile=coverage.out -covermode=atomic
	@cd backend && go tool cover -func=coverage.out | tail -n 1
	@echo "✅ Tests completed"
	@echo "📊 Coverage report: backend/coverage.out"
	@echo "   View HTML: make test-coverage-html"

test-coverage-html: ## Generate HTML coverage report
	@echo "📊 Generating HTML coverage report..."
	@cd backend && go tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report: backend/coverage.html"
	@open backend/coverage.html || xdg-open backend/coverage.html 2>/dev/null || echo "Open backend/coverage.html in your browser"

frontend-build: ## Build frontend for production
	@echo "📦 Building frontend..."
	@cd frontend && npm run build
	@echo "✅ Frontend built to frontend/dist/"

backend-build: frontend-build ## Build backend binary with embedded frontend
	@echo "🔨 Building Go binary with embedded frontend..."
	@rm -rf backend/web
	@mkdir -p backend/web
	@cp -r frontend/dist/* backend/web/
	@cd backend && go build -o ../bin/homelab cmd/server/main.go
	@echo "✅ Binary built to bin/homelab"

build: backend-build ## Build complete application (frontend + backend)
	@echo "✅ Build complete!"
	@echo "   Run: ./bin/homelab"

types: ## Generate TypeScript types from Go structs
	@echo "🔄 Generating TypeScript types from Go models..."
	@command -v $(shell go env GOPATH)/bin/tygo >/dev/null 2>&1 || { echo "Installing tygo..."; go install github.com/gzuidhof/tygo@latest; }
	@cd backend && $(shell go env GOPATH)/bin/tygo generate
	@echo "✅ Types generated at frontend/src/api/generated-types.ts"

clean: ## Clean build artifacts
	@echo "🧹 Cleaning build artifacts..."
	@rm -rf frontend/dist
	@rm -rf backend/web
	@rm -rf bin/homelab
	@rm -rf frontend/node_modules
	@rm -f backend/coverage.out backend/coverage.html
	@cd backend && go clean
	@echo "✅ Clean complete"

docker: ## Build Docker image
	@echo "🐳 Building Docker image..."
	@docker build -f deployments/docker/Dockerfile -t homelab:latest .
	@echo "✅ Docker image built: homelab:latest"

install-deps: ## Install all dependencies
	@echo "📥 Installing dependencies..."
	@cd frontend && npm install
	@cd backend && go mod download
	@echo "✅ Dependencies installed"
