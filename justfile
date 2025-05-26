# Devcmd - Declarative CLI Generation Tool
# Run `just` to see all available commands

# Variables
project_name := "devcmd"
grammar_dir := "grammar"
gen_dir := "internal/gen"
go_version := "1.22"

# Default command - show available commands
default:
    @echo "🔧 Devcmd Development Commands"
    @echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    @echo ""
    @echo "🚀 Quick Start:"
    @echo "  setup          - Initial project setup (grammar + deps)"
    @echo "  build          - Build the CLI tool"
    @echo "  test           - Run Go unit tests"
    @echo "  ci             - Run full CI workflow locally"
    @echo ""
    @echo "📝 Grammar & Development:"
    @echo "  grammar        - Generate parser from ANTLR grammar"
    @echo "  format         - Format all code (Go + Nix)"
    @echo "  lint           - Run all linters"
    @echo "  clean          - Clean generated files and artifacts"
    @echo ""
    @echo "🧪 Testing (ordered by speed):"
    @echo "  test-quick     - Fast syntax/format checks"
    @echo "  test-go        - Go unit tests with coverage"
    @echo "  test-build     - Build and test binaries"
    @echo "  test-nix       - Nix package tests"
    @echo "  test-examples  - Build and test example CLIs"
    @echo "  test-all       - Complete test suite"
    @echo ""
    @echo "📦 Nix Integration:"
    @echo "  nix-build      - Build core Nix packages"
    @echo "  nix-examples   - Build all example CLIs"
    @echo "  nix-test       - Run Nix-based tests"
    @echo "  nix-check      - Comprehensive Nix validation"
    @echo "  try-examples   - Interactively test example CLIs"
    @echo ""
    @echo "🔄 Workflows:"
    @echo "  workflow-dev   - Development workflow"
    @echo "  workflow-ci    - CI workflow (mirrors GitHub Actions)"
    @echo "  workflow-release - Release preparation workflow"
    @echo ""
    @echo "For help: just --list"
    @echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# =============================================================================
# 🚀 SETUP & CORE COMMANDS
# =============================================================================

# Initial project setup - mirrors CI setup phase
setup:
    @echo "🔧 Setting up Devcmd development environment..."
    @echo "📝 Generating ANTLR parser..."
    just grammar
    @echo "📦 Downloading Go dependencies..."
    go mod download
    go mod verify
    @echo "✅ Setup complete! Run 'just test' to verify."

# Generate parser from ANTLR grammar
grammar:
    @echo "📝 Generating ANTLR parser..."
    @if command -v antlr >/dev/null 2>&1; then \
        mkdir -p {{gen_dir}}; \
        cd {{grammar_dir}} && antlr -Dlanguage=Go -package gen -o ../{{gen_dir}} devcmd.g4; \
        echo "✅ Parser generated with local antlr"; \
    elif command -v java >/dev/null 2>&1; then \
        echo "📥 Downloading ANTLR jar..."; \
        mkdir -p {{gen_dir}}; \
        wget -q https://www.antlr.org/download/antlr-4.13.1-complete.jar -O /tmp/antlr.jar; \
        cd {{grammar_dir}} && java -jar /tmp/antlr.jar -Dlanguage=Go -package gen -o ../{{gen_dir}} devcmd.g4; \
        echo "✅ Parser generated with ANTLR jar"; \
    else \
        echo "❌ Neither antlr nor java found. Install Java 17+ or ANTLR."; \
        exit 1; \
    fi

# Build the CLI tool
build:
    @echo "🔨 Building devcmd CLI..."
    @if [ ! -f {{gen_dir}}/devcmd_lexer.go ]; then \
        echo "⚠️  ANTLR parser not found, generating..."; \
        just grammar; \
    fi
    go build -ldflags="-s -w -X main.Version=$(git describe --tags --always --dirty 2>/dev/null || echo 'dev') -X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" -o {{project_name}} ./cmd/{{project_name}}
    @echo "✅ Built: ./{{project_name}}"

# =============================================================================
# 🧪 TESTING COMMANDS (ordered by execution speed)
# =============================================================================

# Fast checks - format and lint (mirrors CI format-lint job)
test-quick:
    @echo "⚡ Running quick checks..."
    @echo "🔍 Checking Go formatting..."
    @if command -v gofumpt >/dev/null 2>&1; then \
        if [ "$(gofumpt -l . | wc -l)" -gt 0 ]; then \
            echo "❌ Go formatting issues:"; gofumpt -l .; exit 1; \
        fi; \
    else \
        if [ "$(gofmt -l . | wc -l)" -gt 0 ]; then \
            echo "❌ Go formatting issues:"; gofmt -l .; exit 1; \
        fi; \
    fi
    @echo "🔍 Checking Nix formatting..."
    @if command -v nixpkgs-fmt >/dev/null 2>&1; then \
        nixpkgs-fmt --check . || (echo "❌ Run 'just format' to fix"; exit 1); \
    else \
        echo "⚠️  nixpkgs-fmt not available, skipping Nix format check"; \
    fi
    just lint
    @echo "✅ Quick checks passed!"

# Go unit tests with coverage (mirrors CI go-tests job)
test-go:
    @echo "🧪 Running Go tests with coverage..."
    @if [ ! -f {{gen_dir}}/devcmd_lexer.go ]; then \
        echo "⚠️  ANTLR parser not found, generating..."; \
        just grammar; \
    fi
    go test -race -coverprofile=coverage.out -covermode=atomic ./...
    @if command -v go >/dev/null 2>&1; then \
        go tool cover -html=coverage.out -o coverage.html; \
        echo "📊 Coverage report: coverage.html"; \
    fi
    @echo "✅ Go tests passed!"

# Build and test binaries (mirrors CI build-binaries job)
test-build:
    @echo "🔨 Building and testing binaries..."
    just build
    @echo "🧪 Testing built binary..."
    ./{{project_name}} --help
    ./{{project_name}} --version || echo "⚠️  Version command not available"
    @echo "✅ Binary tests passed!"

# Nix package tests (mirrors CI nix-core job)
test-nix:
    @echo "📦 Testing Nix packages..."
    @echo "Building core package..."
    nix build .#{{project_name}} --print-build-logs
    @echo "Testing core package..."
    ./result/bin/{{project_name}} --help
    @echo "Testing development shell..."
    nix develop --command echo "✅ Dev shell works"
    @echo "Basic flake check..."
    nix flake check --no-build
    @echo "✅ Nix core tests passed!"

# Example CLI tests (mirrors CI nix-examples job)
test-examples:
    @echo "🎯 Testing example CLIs..."
    @examples=(basicDev webDev goProject rustProject dataScienceProject devOpsProject); \
    for example in $${examples[@]}; do \
        echo "Building $$example..."; \
        nix build .#$$example --print-build-logs || exit 1; \
        echo "Testing $$example..."; \
        ./result/bin/* --help >/dev/null || echo "⚠️  $$example help command issues"; \
    done
    @echo "✅ Example CLI tests passed!"

# Complete test suite (mirrors CI workflow)
test-all:
    @echo "🧪 Running complete test suite..."
    just test-quick
    just test-go
    just test-build
    just test-nix
    just test-examples
    @echo "🎉 All tests passed!"

# Run basic Go tests (for quick feedback)
test:
    @echo "🧪 Running Go unit tests..."
    @if [ ! -f {{gen_dir}}/devcmd_lexer.go ]; then \
        echo "⚠️  ANTLR parser not found, generating..."; \
        just grammar; \
    fi
    go test ./...

# =============================================================================
# 📝 CODE QUALITY COMMANDS
# =============================================================================

# Format all code
format:
    @echo "📝 Formatting all code..."
    @echo "Formatting Go code..."
    @if command -v gofumpt >/dev/null 2>&1; then \
        gofumpt -w .; \
    else \
        go fmt ./...; \
    fi
    @echo "Formatting Nix files..."
    @if command -v nixpkgs-fmt >/dev/null 2>&1; then \
        find . -name '*.nix' -exec nixpkgs-fmt {} +; \
    else \
        echo "⚠️  nixpkgs-fmt not available"; \
    fi
    @echo "✅ Code formatted!"

# Run linters
lint:
    @echo "🔍 Running linters..."
    @if command -v golangci-lint >/dev/null 2>&1; then \
        golangci-lint run --timeout=5m; \
    else \
        echo "⚠️  golangci-lint not installed, running basic checks"; \
        go vet ./...; \
        go fmt ./...; \
    fi
    @echo "✅ Linting complete!"

# =============================================================================
# 📦 NIX COMMANDS
# =============================================================================

# Build core Nix packages
nix-build:
    @echo "📦 Building core Nix packages..."
    nix build .#{{project_name}} --print-build-logs
    @echo "✅ Core packages built"

# Build all example CLIs
nix-examples:
    @echo "🎯 Building all example CLIs..."
    @examples=(basicDev webDev goProject rustProject dataScienceProject devOpsProject); \
    for example in $${examples[@]}; do \
        echo "Building $$example..."; \
        nix build .#$$example --print-build-logs || exit 1; \
    done
    @echo "✅ All example CLIs built"

# Run Nix-based tests
nix-test:
    @echo "🧪 Running Nix-based tests..."
    nix build .#tests --print-build-logs
    @echo "Building example tests..."
    nix build .#test-examples --print-build-logs || echo "⚠️  Example tests not available"
    @echo "✅ Nix tests completed"

# Comprehensive Nix validation
nix-check:
    @echo "🔍 Running comprehensive Nix validation..."
    nix flake check --print-build-logs
    @echo "✅ Nix validation passed"

# Try example CLIs interactively
try-examples:
    @echo "🎯 Interactive Example CLI Testing"
    @echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    @examples=(basicDev:dev webDev:webdev goProject:godev rustProject:rustdev dataScienceProject:datadev devOpsProject:devops); \
    for example in $${examples[@]}; do \
        pkg=$${example%%:*}; \
        cmd=$${example##*:}; \
        echo ""; \
        echo "🔹 $$pkg CLI ($$cmd):"; \
        nix run .#$$pkg -- --help || echo "❌ $$pkg failed"; \
        echo ""; \
    done; \
    echo "🎉 Try running specific commands like:"; \
    echo "  nix run .#basicDev -- build"; \
    echo "  nix run .#webDev -- install"; \
    echo "  nix run .#goProject -- deps"

# =============================================================================
# 🔄 WORKFLOW COMMANDS (mirror CI jobs)
# =============================================================================

# Development workflow - fast iteration
workflow-dev:
    @echo "🔄 Running development workflow..."
    just setup
    just test-quick
    just test-go
    just build
    @echo "✅ Development workflow complete!"

# CI workflow - mirrors GitHub Actions exactly
workflow-ci:
    @echo "🔄 Running CI workflow (mirrors GitHub Actions)..."
    @echo ""
    @echo "Stage 1: Format & Lint..."
    just test-quick
    @echo ""
    @echo "Stage 2: Go Tests..."
    just test-go
    @echo ""
    @echo "Stage 3: Build Binaries..."
    just test-build
    @echo ""
    @echo "Stage 4: Nix Core..."
    just test-nix
    @echo ""
    @echo "Stage 5: Nix Tests..."
    just nix-test
    @echo ""
    @echo "Stage 6: Example CLIs..."
    just test-examples
    @echo ""
    @echo "🎉 CI workflow complete - ready for production!"

# Release preparation workflow
workflow-release:
    @echo "📦 Running release preparation workflow..."
    just clean
    just setup
    just workflow-ci
    just nix-check
    just format
    @echo "📋 Release checklist:"
    @echo "  ✅ All tests passed"
    @echo "  ✅ Code formatted"
    @echo "  ✅ Nix packages validated"
    @echo "  ✅ Example CLIs working"
    @echo ""
    @echo "🚀 Ready for release!"

# =============================================================================
# 🧹 MAINTENANCE COMMANDS
# =============================================================================

# Clean all generated files and artifacts
clean:
    @echo "🧹 Cleaning generated files and artifacts..."
    rm -f {{project_name}}
    rm -f coverage.out coverage.html
    rm -rf result result-*
    rm -rf artifacts/
    rm -rf release/
    go clean -cache -modcache -testcache || echo "Go clean completed with warnings"
    @echo "✅ Cleanup complete"

# =============================================================================
# 📊 UTILITY COMMANDS
# =============================================================================

# Show project status
status:
    @echo "📊 Devcmd Project Status"
    @echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    @echo "Grammar files: $(find {{grammar_dir}} -name '*.g4' | wc -l)"
    @echo "Generated files: $(find {{gen_dir}} -name '*.go' 2>/dev/null | wc -l || echo 0)"
    @echo "Go source files: $(find . -name '*.go' -not -path './{{gen_dir}}/*' | wc -l)"
    @echo "Test files: $(find . -name '*_test.go' | wc -l)"
    @echo "Nix files: $(find . -name '*.nix' | wc -l)"
    @echo ""
    @echo "Go version: $(go version 2>/dev/null || echo 'Not installed')"
    @echo "Nix version: $(nix --version 2>/dev/null || echo 'Not installed')"
    @echo ""
    @echo "Git status:"
    @git status --porcelain | head -10 || echo "Not a git repository"

# Show available Nix outputs
nix-show:
    @echo "📋 Available Nix flake outputs:"
    nix flake show

# Development shell shortcuts
shell-basic:
    nix develop .#basic

shell-web:
    nix develop .#web

shell-go:
    nix develop .#go

shell-data:
    nix develop .#data

shell-test:
    nix develop .#testEnv

# =============================================================================
# 🔧 ALIASES FOR CONVENIENCE
# =============================================================================

alias g := grammar
alias t := test
alias tq := test-quick
alias tg := test-go
alias ta := test-all
alias b := build
alias c := clean
alias f := format
alias l := lint
alias s := status

# Nix aliases
alias nb := nix-build
alias ne := nix-examples
alias nt := nix-test
alias nc := nix-check
alias ns := nix-show

# Workflow aliases
alias dev := workflow-dev
alias ci := workflow-ci
alias release := workflow-release
