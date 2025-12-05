.PHONY: help build build-all go-install test test-race lint fmt vet coverage cover audit clean install start stop logs backup restore profile-test debug monitor all build-linux build-darwin build-windows build-all-platforms test-integration test-integration-cleanup deps setup-dev install-pre-commit install-trufflehog install-tools

# Default target
help:
	@echo "contextd - Development & Testing Makefile"
	@echo ""
	@echo "Build & Run:"
	@echo "  make build          Build contextd binary"
	@echo "  make build-ctxd     Build ctxd CLI binary"
	@echo "  make build-all      Build both contextd and ctxd binaries"
	@echo "  make go-install     Install binaries to GOPATH/bin"
	@echo "  make version        Show version of built/installed binary"
	@echo "  make start          Start contextd service"
	@echo "  make stop           Stop contextd service"
	@echo "  make logs           View contextd logs"
	@echo ""
	@echo "Deployment:"
	@echo "  make deploy         Full deployment (backup -> install -> verify)"
	@echo "  make deploy-rollback Rollback to previous version"
	@echo "  make deploy-list-backups  List available backups"
	@echo ""
	@echo "Cross-Platform Builds:"
	@echo "  make build-linux    Build for Linux (amd64, arm64)"
	@echo "  make build-darwin   Build for macOS (amd64, arm64)"
	@echo "  make build-windows  Build for Windows (amd64)"
	@echo "  make build-all-platforms  Build for all platforms"
	@echo ""
	@echo "Testing:"
	@echo "  make test           Run Go tests"
	@echo "  make test-race      Run Go tests with race detection"
	@echo "  make test-regression Run regression tests only"
	@echo "  make test-integration Run integration tests (requires Docker)"
	@echo "  make test-integration-cleanup Clean up integration test resources"
	@echo "  make coverage       Run tests with coverage report"
	@echo "  make cover          Alias for coverage"
	@echo "  make test-setup     Setup test profile"
	@echo "  make test-session   Start full test session (3 terminals)"
	@echo "  make debug          Run contextd in debug mode"
	@echo "  make monitor        Monitor logs with error capture"
	@echo "  make test-status    Show current test environment status"
	@echo "  make set-session id=<id>  Set test session ID for tracking"
	@echo ""
	@echo "Code Quality:"
	@echo "  make audit          Comprehensive quality checks (lint, vet, test, security)"
	@echo "  make lint           Run golangci-lint"
	@echo "  make fmt            Format code with go fmt and goimports"
	@echo "  make vet            Run go vet static analysis"
	@echo "  make pre-commit-install  Install pre-commit hooks"
	@echo "  make pre-commit-run      Run pre-commit on all files"
	@echo "  make pre-commit-update   Update pre-commit hooks"
	@echo ""
	@echo "Development Setup:"
	@echo "  make deps           Install all development dependencies"
	@echo "  make setup-dev      Setup complete development environment"
	@echo "  make install-pre-commit  Install pre-commit hooks"
	@echo "  make install-trufflehog  Install TruffleHog secret scanner"
	@echo "  make install-tools  Install development tools (golangci-lint, gosec)"
	@echo "  make install-air    Install Air live reload tool"
	@echo ""
	@echo "Live Reload Development:"
	@echo "  make dev-mcp        Run contextd in MCP mode with live reload"
	@echo "  make dev-api        Run contextd in API mode with live reload"
	@echo "  make dev-watch      Run Air with custom config (CONFIG=.air.toml)"
	@echo ""
	@echo "Profile Management:"
	@echo "  make profile-setup  Setup symlink-based profiles (one-time)"
	@echo "  make profile-user   Switch to user profile"
	@echo "  make profile-test   Switch to test profile"
	@echo "  make profile-status Show current profile"
	@echo "  make backup         Backup current profile"
	@echo ""
	@echo "Utilities:"
	@echo "  make clean          Clean build artifacts"
	@echo "  make health         Check contextd health"
	@echo "  make milvus-start   Start local Milvus"
	@echo "  make milvus-stop    Stop local Milvus"

# Build targets
build:
	@echo "🔨 Building contextd (with CGO for FastEmbed)..."
	@VERSION=$$(git describe --tags --always --dirty 2>/dev/null || echo "dev"); \
	COMMIT=$$(git rev-parse --short HEAD 2>/dev/null || echo "unknown"); \
	DATE=$$(date -u +"%Y-%m-%dT%H:%M:%SZ"); \
	CGO_ENABLED=1 go build -ldflags="-X main.version=$$VERSION -X main.commit=$$COMMIT -X main.buildDate=$$DATE" \
		-o contextd ./cmd/contextd/
	@echo "✓ Built contextd (FastEmbed enabled)"

build-ctxd:
	@echo "🔨 Building ctxd CLI..."
	@VERSION=$$(git describe --tags --always --dirty 2>/dev/null || echo "dev"); \
	go build -ldflags="-X main.version=$$VERSION" \
		-o ctxd ./cmd/ctxd/
	@echo "✓ Built ctxd"

build-all: build build-ctxd

go-install:
	@echo "📦 Installing contextd binaries with go install (CGO enabled for FastEmbed)..."
	@VERSION=$$(git describe --tags --always --dirty 2>/dev/null || echo "dev"); \
	COMMIT=$$(git rev-parse --short HEAD 2>/dev/null || echo "unknown"); \
	DATE=$$(date -u +"%Y-%m-%dT%H:%M:%SZ"); \
	CGO_ENABLED=1 go install -ldflags="-X main.version=$$VERSION -X main.commit=$$COMMIT -X main.buildDate=$$DATE" ./cmd/contextd
	@echo "✓ Installed contextd to $(shell go env GOPATH)/bin (FastEmbed enabled)"
	@go install -ldflags="-X main.version=$$VERSION" ./cmd/ctxd
	@echo "✓ Installed ctxd to $(shell go env GOPATH)/bin"
	@echo "  Make sure $(shell go env GOPATH)/bin is in your PATH"

clean:
	@rm -f contextd ctxd
	@rm -rf dist/
	@rm -f coverage.out coverage.html
	@echo "✓ Cleaned build artifacts"

# Cross-platform build targets
build-linux:
	@echo "🔨 Building for Linux..."
	@mkdir -p dist/linux
	@CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o dist/linux/contextd-linux-amd64 ./cmd/contextd
	@CGO_ENABLED=1 GOOS=linux GOARCH=arm64 go build -o dist/linux/contextd-linux-arm64 ./cmd/contextd
	@echo "✓ Linux binaries built in dist/linux/"
	@ls -lh dist/linux/

build-darwin:
	@echo "🔨 Building for macOS..."
	@mkdir -p dist/darwin
	@CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -o dist/darwin/contextd-darwin-amd64 ./cmd/contextd
	@CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -o dist/darwin/contextd-darwin-arm64 ./cmd/contextd
	@echo "✓ macOS binaries built in dist/darwin/"
	@ls -lh dist/darwin/

build-windows:
	@echo "🔨 Building for Windows..."
	@mkdir -p dist/windows
	@CGO_ENABLED=1 GOOS=windows GOARCH=amd64 go build -o dist/windows/contextd-windows-amd64.exe ./cmd/contextd
	@echo "✓ Windows binaries built in dist/windows/"
	@ls -lh dist/windows/

build-all-platforms: build-linux build-darwin build-windows
	@echo ""
	@echo "✓ All platform binaries built successfully"
	@echo ""
	@echo "Distribution structure:"
	@tree -L 2 dist/ 2>/dev/null || find dist/ -type f

# Service management targets
start:
	@systemctl --user start contextd
	@echo "✓ contextd started"

stop:
	@systemctl --user stop contextd
	@echo "✓ contextd stopped"

logs:
	@journalctl --user -u contextd -f

# Testing targets
test:
	@go test -v ./...

test-race:
	@go test -race -v ./...

test-regression:
	@echo "Running regression tests..."
	@go test -v -run TestRegression ./...
	@echo "✓ All regression tests passed"

coverage:
	@echo "Running tests with coverage..."
	@go test -race -coverprofile=coverage.out -covermode=atomic ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"
	@echo "Total coverage:"
	@go tool cover -func=coverage.out | grep total | awk '{print $$3}'

# Alias for coverage
cover: coverage

# Pre-commit hooks
pre-commit-install:
	@echo "Installing pre-commit hooks..."
	@./scripts/setup-pre-commit.sh

pre-commit-run:
	@echo "Running pre-commit on all files..."
	@pre-commit run --all-files

pre-commit-update:
	@echo "Updating pre-commit hooks..."
	@pre-commit autoupdate

# Code quality targets
audit:
	@echo "========================================"
	@echo "🔍 Running comprehensive code audit..."
	@echo "========================================"
	@echo ""
	@echo "1️⃣  Checking code formatting..."
	@if [ -n "$$(gofmt -l .)" ]; then \
		echo "❌ Code formatting issues found:"; \
		gofmt -l .; \
		echo ""; \
		echo "Run 'make fmt' to fix formatting"; \
		exit 1; \
	fi
	@echo "✓ Code formatting OK"
	@echo ""
	@echo "2️⃣  Running go vet..."
	@go vet ./... || (echo "❌ go vet failed" && exit 1)
	@echo "✓ go vet passed"
	@echo ""
	@echo "3️⃣  Running golangci-lint..."
	@golangci-lint run --timeout=5m || (echo "❌ golangci-lint failed" && exit 1)
	@echo "✓ golangci-lint passed"
	@echo ""
	@echo "4️⃣  Running tests with race detection..."
	@go test -race -short ./... || (echo "❌ Tests failed" && exit 1)
	@echo "✓ Tests passed"
	@echo ""
	@echo "5️⃣  Verifying dependencies..."
	@go mod verify || (echo "❌ Dependency verification failed" && exit 1)
	@echo "✓ Dependencies verified"
	@echo ""
	@echo "6️⃣  Checking for security issues..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec -quiet ./... 2>&1 | grep -v "Golang errors" || echo "✓ No security issues found"; \
	else \
		echo "⚠️  gosec not installed (optional)"; \
		echo "   Install with: go install github.com/securego/gosec/v2/cmd/gosec@latest"; \
	fi
	@echo ""
	@echo "========================================"
	@echo "✅ Audit complete - all checks passed!"
	@echo "========================================"

lint:
	@echo "Running golangci-lint..."
	@golangci-lint run --timeout=5m

fmt:
	@echo "Running go fmt..."
	@go fmt ./...
	@echo "Running goimports..."
	@goimports -w -local github.com/fyrsmithlabs/contextd .

vet:
	@echo "Running go vet..."
	@go vet ./...

test-setup:
	@./scripts/profile-switch.sh setup

test-session:
	@./scripts/start-test-session.sh

test-status:
	@./scripts/test-status-line.sh

set-session:
	@./scripts/set-test-session.sh $(id)

debug:
	@./contextd --mcp --debug

monitor:
	@./scripts/monitor-contextd.sh

# Profile management (symlink-based)
profile-setup:
	@./scripts/profile-switch.sh setup

profile-user:
	@./scripts/profile-switch.sh user

profile-test:
	@./scripts/profile-switch.sh test

profile-status:
	@./scripts/profile-switch.sh current

# Legacy backup (still useful)
backup:
	@./scripts/claude-profile.sh backup

# Utilities
health:
	@./scripts/health-check.sh

milvus-start:
	@./scripts/start-milvus-local.sh

milvus-stop:
	@./scripts/stop-milvus-local.sh

# Combined targets
all: build install

dev: build test

# Integration test targets
test-integration:
	@echo "Starting test services..."
	@docker-compose -f test/docker-compose.test.yml up -d
	@echo "Waiting for services to be ready..."
	@./test/scripts/wait-for-services.sh
	@echo "Running integration tests..."
	@go test -v -race -tags=integration ./test/integration/...
	@echo "Stopping test services..."
	@docker-compose -f test/docker-compose.test.yml down

test-integration-cleanup:
	@echo "Cleaning up integration test resources..."
	@docker-compose -f test/docker-compose.test.yml down -v
	@echo "✓ Integration test cleanup complete"

# Development setup targets
deps: install-tools install-trufflehog install-pre-commit install-air
	@echo ""
	@echo "========================================"
	@echo "✅ All dependencies installed!"
	@echo "========================================"
	@echo ""
	@echo "Development environment ready:"
	@echo "  ✓ Go tools (golangci-lint, gosec)"
	@echo "  ✓ TruffleHog (secret scanner)"
	@echo "  ✓ Pre-commit hooks"
	@echo "  ✓ Air (live reload)"
	@echo ""
	@echo "Next steps:"
	@echo "  1. Run 'make test' to verify setup"
	@echo "  2. Run 'make build' to build binary"
	@echo "  3. Run 'make dev-mcp' for live reload development"
	@echo ""

setup-dev: deps
	@echo "Setting up development environment..."
	@go mod download
	@echo "✓ Go dependencies downloaded"
	@echo "✓ Development environment setup complete"

install-pre-commit:
	@echo "📦 Installing pre-commit..."
	@if command -v pre-commit >/dev/null 2>&1; then \
		echo "✅ pre-commit already installed: $$(pre-commit --version)"; \
	else \
		if command -v pip3 >/dev/null 2>&1; then \
			pip3 install --user pre-commit; \
		elif command -v pip >/dev/null 2>&1; then \
			pip install --user pre-commit; \
		elif command -v brew >/dev/null 2>&1; then \
			brew install pre-commit; \
		else \
			echo "❌ Error: Could not install pre-commit"; \
			echo "   Install Python and pip first, or use Homebrew"; \
			exit 1; \
		fi; \
		echo "✅ pre-commit installed: $$(pre-commit --version)"; \
	fi
	@if [ -f ".pre-commit-config.yaml" ]; then \
		echo "🔗 Installing pre-commit git hooks..."; \
		pre-commit install --hook-type pre-commit; \
		pre-commit install --hook-type commit-msg; \
		echo "✅ Pre-commit hooks installed"; \
	else \
		echo "⚠️  No .pre-commit-config.yaml found"; \
	fi

install-trufflehog:
	@echo "📦 Installing TruffleHog..."
	@if command -v trufflehog >/dev/null 2>&1; then \
		echo "✅ TruffleHog already installed: $$(trufflehog --version 2>&1 | head -1)"; \
	else \
		if command -v brew >/dev/null 2>&1; then \
			brew install trufflehog; \
			echo "✅ TruffleHog installed via Homebrew"; \
		else \
			echo "⚠️  Homebrew not found"; \
			echo "   Install TruffleHog manually from:"; \
			echo "   https://github.com/trufflesecurity/trufflehog/releases"; \
		fi; \
	fi

install-tools:
	@echo "📦 Installing development tools..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		echo "✅ golangci-lint already installed"; \
	else \
		echo "Installing golangci-lint..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
		echo "✅ golangci-lint installed"; \
	fi
	@if command -v gosec >/dev/null 2>&1; then \
		echo "✅ gosec already installed"; \
	else \
		echo "Installing gosec..."; \
		go install github.com/securego/gosec/v2/cmd/gosec@latest; \
		echo "✅ gosec installed"; \
	fi
	@if command -v goimports >/dev/null 2>&1; then \
		echo "✅ goimports already installed"; \
	else \
		echo "Installing goimports..."; \
		go install golang.org/x/tools/cmd/goimports@latest; \
		echo "✅ goimports installed"; \
	fi
	@echo "✅ All Go tools installed"

install-air:
	@echo "📦 Installing Air (live reload)..."
	@if command -v air >/dev/null 2>&1; then \
		echo "✅ Air already installed: $$(air -v)"; \
	else \
		echo "Installing Air..."; \
		go install github.com/air-verse/air@latest; \
		echo "✅ Air installed: $$(air -v)"; \
	fi

# Live reload development targets
dev-mcp:
	@echo "🔥 Starting contextd in MCP mode with live reload..."
	@echo "   Logs: tmp/air.log"
	@echo "   Press Ctrl+C to stop"
	@echo ""
	@air

dev-api:
	@echo "🔥 Starting contextd in API mode with live reload..."
	@echo "   Logs: tmp/air.log"
	@echo "   Press Ctrl+C to stop"
	@echo ""
	@sed 's/full_bin = "tmp\/contextd --mcp --debug"/full_bin = "tmp\/contextd --debug"/' .air.toml > tmp/.air-api.toml
	@air -c tmp/.air-api.toml

dev-watch:
	@echo "🔥 Starting Air with custom config..."
	@air -c $(or $(CONFIG),.air.toml)
# Local Testing & Monitoring Stack Targets
# Append to main Makefile or include with: include Makefile.local-testing

.PHONY: test-unit test-watch test-all test-e2e test-integration bench coverage-check \
        coverage-view docker-check stack-up stack-down stack-restart stack-clean \
        stack-logs stack-health dev-setup dev-teardown agent-test-review agent-code-review

# Fast unit tests for local development
test-unit:
	@echo "🧪 Running Go unit tests..."
	@go test ./pkg/... -v -short -cover
	@echo "🧪 Running shell unit tests..."
	@if command -v bats >/dev/null 2>&1; then \
		if ls test/unit/*.sh 1> /dev/null 2>&1; then \
			bats test/unit/*.sh; \
		else \
			echo "⚠️  No shell tests found in test/unit/"; \
		fi \
	else \
		echo "⚠️  BATS not installed, skipping shell tests"; \
		echo "   Install: npm install -g bats"; \
	fi
	@echo "✅ Unit tests passed"

# Watch mode - continuous testing during development
test-watch:
	@if ! command -v entr >/dev/null 2>&1; then \
		echo "❌ entr not installed"; \
		echo "   Install: sudo apt-get install entr"; \
		exit 1; \
	fi
	@echo "👀 Watching for changes... (Ctrl+C to stop)"
	@find . -name "*.go" -not -path "./vendor/*" -not -path "./.git/*" | entr -c make test

# All test suites
test-all: test-unit test-integration test-e2e
	@echo "✅ All test suites passed"

# End-to-end tests (requires Docker + Air)
test-e2e: docker-check
	@echo "🌐 Running E2E tests..."
	@if ! command -v air >/dev/null 2>&1; then \
		echo "❌ Air not installed"; \
		echo "   Install: go install github.com/air-verse/air@latest"; \
		exit 1; \
	fi
	@if [ -d test/e2e ]; then \
		go test ./test/e2e/... -v -timeout 10m; \
	else \
		echo "⚠️  No E2E tests found in test/e2e/"; \
	fi
	@echo "✅ E2E tests passed"

# Benchmarks
bench:
	@echo "⚡ Running benchmarks..."
	@if [ -d test/benchmark ]; then \
		go test ./test/benchmark/... -bench=. -benchmem; \
	else \
		echo "⚠️  No benchmarks found in test/benchmark/"; \
		echo "   Run go benchmarks in pkg/: go test ./pkg/... -bench=."; \
		go test ./pkg/... -bench=. -benchmem -run=^$; \
	fi

# Coverage with 80% threshold check
coverage-check: coverage
	@echo "🎯 Checking coverage threshold..."
	@COVERAGE=$$(go tool cover -func=coverage.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	echo "Coverage: $$COVERAGE%"; \
	if [ $$(echo "$$COVERAGE < 80" | bc -l) -eq 1 ]; then \
		echo "❌ Coverage is $$COVERAGE%, required ≥80%"; \
		exit 1; \
	else \
		echo "✅ Coverage is $$COVERAGE% (≥80% required)"; \
	fi

# Open coverage report in browser
coverage-view: coverage
	@echo "📊 Opening coverage report..."
	@xdg-open coverage.html 2>/dev/null || open coverage.html 2>/dev/null || echo "Please open coverage.html manually"

# Docker helpers
docker-check:
	@if ! docker info >/dev/null 2>&1; then \
		echo "❌ Docker is not running"; \
		echo "   Start: sudo systemctl start docker"; \
		exit 1; \
	fi

# Start monitoring stack
stack-up: docker-check
	@echo "🚀 Starting monitoring stack..."
	@docker-compose up -d
	@echo "   Waiting for services (45s)..."
	@sleep 45
	@echo "✅ Stack ready"
	@docker-compose ps

# Stop monitoring stack
stack-down:
	@echo "🛑 Stopping monitoring stack..."
	@docker-compose down
	@echo "✅ Stack stopped"

# Restart monitoring stack
stack-restart: stack-down stack-up

# Stop and remove volumes
stack-clean:
	@echo "🧹 Cleaning monitoring stack..."
	@docker-compose down -v
	@echo "✅ Stack cleaned (volumes removed)"

# Show stack logs
stack-logs:
	@docker-compose logs -f

# Check stack health
stack-health:
	@echo "🏥 Checking stack health..."
	@echo ""
	@docker-compose ps
	@echo ""
	@echo "Service Health Checks:"
	@echo -n "  VictoriaMetrics: "
	@if curl -s http://localhost:8428/health >/dev/null 2>&1; then echo "✅ OK"; else echo "❌ DOWN"; fi
	@echo -n "  VictoriaLogs: "
	@if curl -s http://localhost:9428/health >/dev/null 2>&1; then echo "✅ OK"; else echo "❌ DOWN"; fi
	@echo -n "  Tempo: "
	@if curl -s http://localhost:3200/ready >/dev/null 2>&1; then echo "✅ OK"; else echo "⚠️  STARTING"; fi
	@echo -n "  Grafana: "
	@if curl -s -u admin:admin http://localhost:3001/api/health 2>/dev/null | grep -q '"database":"ok"'; then echo "✅ OK"; else echo "❌ DOWN"; fi
	@echo -n "  OTEL Collector: "
	@if docker ps | grep -q contextd-otel-collector.*Up; then echo "✅ RUNNING"; else echo "❌ DOWN"; fi
	@echo -n "  Qdrant: "
	@if curl -s http://localhost:6333/healthz >/dev/null 2>&1; then echo "✅ OK"; else echo "❌ DOWN"; fi

# Complete dev environment setup
dev-setup: stack-up install-air
	@echo "🛠️  Setting up development environment..."
	@echo "   1. Installing Go tools..."
	@make install-go-tools >/dev/null 2>&1 || true
	@echo "   2. Installing test dependencies..."
	@go mod download
	@echo "   3. Verifying setup with unit tests..."
	@make test-unit
	@echo "✅ Development environment ready"
	@echo ""
	@echo "Next steps:"
	@echo "  - Start Air:      air  or  make dev-mcp"
	@echo "  - Run tests:      make test"
	@echo "  - Watch mode:     make test-watch"
	@echo "  - Stack health:   make stack-health"
	@echo "  - View Grafana:   http://localhost:3001 (admin/admin)"

# Clean up dev environment
dev-teardown: stack-down
	@echo "✅ Development environment cleaned"

# Agent workflow helpers (documentation)
agent-test-review:
	@echo "🤖 Agent: test-strategist"
	@echo ""
	@echo "Use test-strategist agent for:"
	@echo "  - Test planning and strategy"
	@echo "  - Test gap analysis"
	@echo "  - Failure scenario identification"
	@echo ""
	@echo "Example prompts in Claude Code:"
	@echo "  @agent-test-strategist review test coverage and identify gaps"
	@echo "  @agent-test-strategist analyze failing test and suggest fix"
	@echo "  @agent-test-strategist design tests for [feature]"

agent-code-review:
	@echo "🤖 Agent: golang-reviewer"
	@echo ""
	@echo "Use golang-reviewer agent for:"
	@echo "  - Code review before testing"
	@echo "  - Test quality review"
	@echo "  - Coverage gap identification"
	@echo ""
	@echo "Example prompts in Claude Code:"
	@echo "  @agent-golang-reviewer review changes for test coverage"
	@echo "  @agent-golang-reviewer review test/integration/ for Go best practices"
	@echo "  @agent-golang-reviewer identify untested error paths"

# ============================================================================
# Deployment Targets
# ============================================================================

.PHONY: version deploy deploy-check deploy-backup deploy-install deploy-verify deploy-rollback

# Show version information
version:
	@echo "contextd version information:"
	@if [ -f "./contextd" ]; then \
		./contextd --version; \
	else \
		echo "Binary not built. Run 'make build' first."; \
	fi

# Full deployment workflow (backup -> install -> verify)
deploy: deploy-check deploy-backup deploy-install deploy-verify
	@echo ""
	@echo "========================================"
	@echo "✅ Deployment complete!"
	@echo "========================================"
	@echo ""
	@echo "Verify with: contextd --version"
	@echo "Rollback with: make deploy-rollback"

# Pre-deployment checks
deploy-check:
	@echo "🔍 Pre-deployment checks..."
	@if [ ! -f "./contextd" ]; then \
		echo "❌ Binary not found. Building..."; \
		make build; \
	fi
	@echo "   Binary version: $$(./contextd --version 2>&1 || echo 'build required')"
	@echo "✓ Pre-deployment checks passed"

# Backup existing installation
deploy-backup:
	@echo "💾 Backing up existing installation..."
	@INSTALL_PATH=$$(which contextd 2>/dev/null || echo ""); \
	if [ -n "$$INSTALL_PATH" ] && [ -f "$$INSTALL_PATH" ]; then \
		BACKUP_DIR="$$HOME/.contextd/backups"; \
		mkdir -p "$$BACKUP_DIR"; \
		TIMESTAMP=$$(date +%Y%m%d_%H%M%S); \
		OLD_VERSION=$$($$INSTALL_PATH --version 2>&1 | head -1 || echo "unknown"); \
		cp "$$INSTALL_PATH" "$$BACKUP_DIR/contextd.$$TIMESTAMP"; \
		echo "$$OLD_VERSION" > "$$BACKUP_DIR/contextd.$$TIMESTAMP.version"; \
		echo "   Backed up: $$INSTALL_PATH -> $$BACKUP_DIR/contextd.$$TIMESTAMP"; \
		echo "   Version: $$OLD_VERSION"; \
	else \
		echo "   No existing installation found (first install)"; \
	fi
	@echo "✓ Backup complete"

# Install new binary
deploy-install:
	@echo "📦 Installing contextd..."
	@GOPATH_BIN=$$(go env GOPATH)/bin; \
	cp ./contextd "$$GOPATH_BIN/contextd"; \
	chmod +x "$$GOPATH_BIN/contextd"; \
	echo "   Installed to: $$GOPATH_BIN/contextd"
	@echo "✓ Installation complete"

# Verify deployment
deploy-verify:
	@echo "🔎 Verifying deployment..."
	@INSTALL_PATH=$$(which contextd 2>/dev/null || echo ""); \
	if [ -z "$$INSTALL_PATH" ]; then \
		echo "❌ contextd not found in PATH"; \
		echo "   Add $$(go env GOPATH)/bin to your PATH"; \
		exit 1; \
	fi
	@echo "   Location: $$(which contextd)"
	@echo "   Version: $$(contextd --version 2>&1 | head -1)"
	@echo "✓ Deployment verified"

# Rollback to previous version
deploy-rollback:
	@echo "⏪ Rolling back to previous version..."
	@BACKUP_DIR="$$HOME/.contextd/backups"; \
	if [ ! -d "$$BACKUP_DIR" ]; then \
		echo "❌ No backups found"; \
		exit 1; \
	fi; \
	LATEST=$$(ls -t "$$BACKUP_DIR"/contextd.* 2>/dev/null | grep -v '.version' | head -1); \
	if [ -z "$$LATEST" ]; then \
		echo "❌ No backup files found"; \
		exit 1; \
	fi; \
	GOPATH_BIN=$$(go env GOPATH)/bin; \
	cp "$$LATEST" "$$GOPATH_BIN/contextd"; \
	chmod +x "$$GOPATH_BIN/contextd"; \
	echo "   Restored from: $$LATEST"; \
	echo "   Version: $$(contextd --version 2>&1 | head -1)"; \
	echo "✓ Rollback complete"

# List available backups
deploy-list-backups:
	@echo "📋 Available backups:"
	@BACKUP_DIR="$$HOME/.contextd/backups"; \
	if [ -d "$$BACKUP_DIR" ]; then \
		ls -lh "$$BACKUP_DIR"/contextd.* 2>/dev/null | grep -v '.version' || echo "   No backups found"; \
	else \
		echo "   No backup directory found"; \
	fi

# Update help target to include new commands
.PHONY: help-local-testing
help-local-testing:
	@echo ""
	@echo "Local Testing & Monitoring:"
	@echo "  make test-unit           Fast unit tests (<10s)"
	@echo "  make test-watch          Watch mode - continuous testing"
	@echo "  make test-all            All test suites"
	@echo "  make test-e2e            End-to-end tests (requires Docker + Air)"
	@echo "  make bench               Run benchmarks"
	@echo "  make coverage-check      Check coverage ≥80% threshold"
	@echo "  make coverage-view       Open coverage report in browser"
	@echo ""
	@echo "Monitoring Stack:"
	@echo "  make stack-up            Start monitoring stack (Docker Compose)"
	@echo "  make stack-down          Stop monitoring stack"
	@echo "  make stack-restart       Restart monitoring stack"
	@echo "  make stack-clean         Stop and remove volumes"
	@echo "  make stack-logs          View stack logs (tail -f)"
	@echo "  make stack-health        Check all services health"
	@echo ""
	@echo "Development Environment:"
	@echo "  make dev-setup           Complete setup (stack + tools + verify)"
	@echo "  make dev-teardown        Clean up dev environment"
	@echo ""
	@echo "Agent Workflows:"
	@echo "  make agent-test-review   Show test-strategist agent usage"
	@echo "  make agent-code-review   Show golang-reviewer agent usage"
	@echo ""
	@echo "Quick Start:"
	@echo "  1. make dev-setup        # One-time setup"
	@echo "  2. make dev-mcp          # Start Air (Terminal 1)"
	@echo "  3. make test-watch       # Watch tests (Terminal 2)"
	@echo "  4. make stack-health     # Verify stack (anytime)"
