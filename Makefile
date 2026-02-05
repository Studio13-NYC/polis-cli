# Polis CLI - Development Makefile
# For releases, use: goreleaser release --clean

VERSION := $(shell cat cli-go/version.txt 2>/dev/null || echo "dev")

.PHONY: build test clean

build:
	cd cli-go && go build -ldflags "-X main.Version=$(VERSION)" -o polis ./cmd/polis
	@echo "Built cli-go/polis ($(VERSION))"

test:
	cd cli-go && go test ./...

clean:
	rm -f cli-go/polis
	rm -rf dist/
