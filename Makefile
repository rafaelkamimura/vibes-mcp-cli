SHELL := /bin/bash
PHONY: all init build test lint docker-build docker-up release clean

all: build

init:
	@if [ ! -f ".env" ]; then cp .env_example .env && echo ".env created from .env_example"; else echo ".env already exists"; fi
	go mod tidy

build:
	go build -o openai-cli

test:
	go test ./internal/client ./internal/service ./cmd

lint:
	go fmt ./...
	go vet ./...

docker-build:
	docker build -t openai-cli:latest .

docker-up:
	@docker-compose up --build

RELEASE_OSARCH ?= linux/amd64 darwin/amd64 windows/amd64
release:
	@VERSION="$$(git describe --tags --always 2>/dev/null || git rev-parse --short HEAD)"; \
	echo "Building release $$VERSION"; \
	mkdir -p dist; \
	for target in $$(echo "$(RELEASE_OSARCH)"); do \
		OS=$${target%%/*}; ARCH=$${target##*/}; \
		BIN_NAME=dist/openai-cli-$$VERSION-$$OS-$$ARCH; \
		if [ "$$OS" = "windows" ]; then BIN_NAME=$${BIN_NAME}.exe; fi; \
		echo "> $$OS/$$ARCH -> $$BIN_NAME"; \
		GOOS=$${OS} GOARCH=$${ARCH} go build -ldflags='-s -w' -o $$BIN_NAME .; \
	done

clean:
	rm -f openai-cli