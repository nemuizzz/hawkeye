VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LD_FLAGS = -X github.com/nemuizzz/hawkeye/pkg/version.Version=$(VERSION) \
		   -X github.com/nemuizzz/hawkeye/pkg/version.GitCommit=$(COMMIT) \
		   -X github.com/nemuizzz/hawkeye/pkg/version.BuildDate=$(BUILD_DATE)

.PHONY: build
build:
	go build -ldflags "$(LD_FLAGS)" -o bin/hawkeye ./cmd/hawkeye

.PHONY: install
install:
	go install -ldflags "$(LD_FLAGS)" ./cmd/hawkeye

.PHONY: test
test:
	go test -v ./...

.PHONY: clean
clean:
	rm -rf bin/

.PHONY: lint
lint:
	golangci-lint run

.PHONY: release
release:
	@if [ -z "$(TAG)" ]; then \
		echo "Please specify a tag. Example: make release TAG=v0.1.0"; \
		exit 1; \
	fi
	git tag -a $(TAG) -m "Release $(TAG)"
	git push origin $(TAG)

.DEFAULT_GOAL := build 