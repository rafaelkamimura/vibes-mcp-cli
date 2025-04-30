SHELL := /bin/bash
.PHONY: all init build test lint docker-build docker-up release clean

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
	@LAST_TAG=$$(git describe --tags --abbrev=0 2>/dev/null || echo "0.0.0"); \
	LAST_TAG=$${LAST_TAG#v}; \
	MAJOR=$$(echo $$LAST_TAG | cut -d. -f1); \
	MINOR=$$(echo $$LAST_TAG | cut -d. -f2); \
	PATCH=$$(echo $$LAST_TAG | cut -d. -f3); \
	NEW_PATCH=$$((PATCH+1)); \
	VERSION=v$$MAJOR.$$MINOR.$$NEW_PATCH; \
	echo "Releasing $$VERSION"; \
	mkdir -p dist; \
	for target in $(RELEASE_OSARCH); do \
		OS=$${target%%/*}; ARCH=$${target##*/}; \
		EXT=""; if [ "$$OS" = "windows" ]; then EXT=".exe"; fi; \
		BIN=dist/openai-cli-$$VERSION-$$OS-$$ARCH$$EXT; \
		echo "> $$OS/$$ARCH -> $$BIN"; \
		GOOS=$${OS} GOARCH=$${ARCH} go build -ldflags='-s -w' -o $$BIN .; \
	done; \
	git tag -a $$VERSION -m "Release $$VERSION"; \
	git push origin $$VERSION

clean:
	rm -f openai-cli