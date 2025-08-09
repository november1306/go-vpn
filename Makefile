:; # Makefile for Git Bash / WSL

.PHONY: build test lint fmt

build:
go build -o bin/server.exe ./cmd/server
go build -o bin/vpn-cli.exe ./cmd/vpn-cli

test:
go test ./...

lint:
@echo "no linter configured yet"

fmt:
go fmt ./...
