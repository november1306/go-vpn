# Makefile for go-vpn project

.PHONY: build test test-unit test-integration test-docker test-all lint fmt clean clean-all download-wintun help

# Default target
build:
	go build -o bin/server$(shell go env GOEXE) ./cmd/server
	go build -o bin/vpn-cli$(shell go env GOEXE) ./cmd/vpn-cli

# Cross-platform builds for releases
build-all:
	@mkdir -p bin
	GOOS=windows GOARCH=amd64 go build -o bin/server-windows-amd64.exe ./cmd/server
	GOOS=windows GOARCH=amd64 go build -o bin/vpn-cli-windows-amd64.exe ./cmd/vpn-cli
	GOOS=linux GOARCH=amd64 go build -o bin/server-linux-amd64 ./cmd/server
	GOOS=linux GOARCH=amd64 go build -o bin/vpn-cli-linux-amd64 ./cmd/vpn-cli

# Test stages - aligned with CI pipeline
test: test-unit

# Stage 1: Fast unit tests (no external dependencies)
test-unit:
	@echo "Running unit tests..."
	go test -v ./internal/... ./cmd/...

# Stage 2: Integration tests (require Docker/networking)
test-integration: build docker-build
	@echo "Running integration tests..."
	@if [ -f scripts/test-container.sh ]; then \
		chmod +x scripts/test-container.sh && ./scripts/test-container.sh; \
	else \
		echo "Integration test script not found"; \
		exit 1; \
	fi

# Stage 3: Docker container tests
test-docker: docker-build
	@echo "Running Docker tests..."
	@if [ -f scripts/test-container.sh ]; then \
		chmod +x scripts/test-container.sh && ./scripts/test-container.sh go-vpn:latest; \
	else \
		echo "Docker test script not found"; \
		exit 1; \
	fi

# Run all test stages (CI pipeline)
test-all: test-unit test-integration

# Docker support for testing
docker-build:
	@echo "Building Docker image for testing..."
	docker build -t go-vpn:latest .

lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		go vet ./...; \
	fi

fmt:
	go fmt ./...

clean:
ifeq ($(OS),Windows_NT)
	@if exist bin rmdir /s /q bin 2>nul || echo.
	@if exist coverage.out del coverage.out 2>nul || echo.
	@if exist coverage.html del coverage.html 2>nul || echo.
else
	rm -rf bin/
	rm -f coverage.out coverage.html
endif

clean-all: clean
ifeq ($(OS),Windows_NT)
	@if exist lib rmdir /s /q lib 2>nul || echo.
else
	rm -rf lib/
endif

# Download WinTun DLLs for Windows support
download-wintun:
	@echo "Downloading WinTun DLLs..."
	@mkdir -p lib/amd64 lib/arm64 lib/arm lib/x86
	@if command -v curl >/dev/null 2>&1; then \
		curl -L https://www.wintun.net/builds/wintun-0.14.1.zip -o wintun.zip; \
	elif command -v wget >/dev/null 2>&1; then \
		wget -O wintun.zip https://www.wintun.net/builds/wintun-0.14.1.zip; \
	else \
		echo "Error: Neither curl nor wget found. Please install one to download WinTun."; \
		exit 1; \
	fi
	@if command -v unzip >/dev/null 2>&1; then \
		unzip -j wintun.zip "wintun/bin/amd64/wintun.dll" -d lib/amd64/; \
		unzip -j wintun.zip "wintun/bin/arm64/wintun.dll" -d lib/arm64/; \
		unzip -j wintun.zip "wintun/bin/arm/wintun.dll" -d lib/arm/; \
		unzip -j wintun.zip "wintun/bin/x86/wintun.dll" -d lib/x86/; \
	else \
		echo "Error: unzip not found. Please install unzip to extract WinTun DLLs."; \
		exit 1; \
	fi
	@rm -f wintun.zip
	@echo "WinTun DLLs downloaded successfully to lib/ directories"

help:
	@echo "Available targets:"
	@echo "  build             - Build server and CLI"
	@echo "  build-all         - Cross-platform builds"
	@echo ""
	@echo "Test stages (aligned with CI):"
	@echo "  test              - Run unit tests (default)"
	@echo "  test-unit         - Stage 1: Fast unit tests"
	@echo "  test-integration  - Stage 2: Integration tests (requires Docker)"
	@echo "  test-docker       - Stage 3: Docker container tests"
	@echo "  test-all          - Run all test stages"
	@echo ""
	@echo "Other:"
	@echo "  lint              - Run linter"
	@echo "  fmt               - Format code"
	@echo "  clean             - Clean build artifacts"
	@echo "  clean-all         - Clean build artifacts and downloaded libs"
	@echo "  download-wintun   - Download WinTun DLLs for Windows support"
	@echo "  help              - Show this help"