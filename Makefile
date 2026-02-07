# Polis Build Makefile
#
# Three distribution targets:
#   - polis:        CLI-only (~8-9 MB)
#   - polis-server: Webapp-only (~11 MB)
#   - polis-full:   Bundled CLI + serve command (~11-12 MB)
#
# For GitHub releases, use: goreleaser release --clean

.PHONY: all cli webapp bundled clean test

# Default: build all targets
all: cli webapp bundled

# Version file (all Go binaries share the same version)
CLI_VERSION := $(shell cat cli-go/version.txt)

# Output directory
DIST := dist

# CLI-only build
cli:
	@mkdir -p $(DIST)
	cd cli-go && go build -ldflags "-X main.Version=$(CLI_VERSION)" -o ../$(DIST)/polis ./cmd/polis
	@echo "Built $(DIST)/polis (version $(CLI_VERSION))"

# Webapp-only build
webapp:
	@mkdir -p $(DIST)
	cd webapp/localhost && go build -ldflags "-X main.Version=$(CLI_VERSION)" -o ../../$(DIST)/polis-server ./cmd/server
	@echo "Built $(DIST)/polis-server (version $(CLI_VERSION))"

# Bundled build (uses CLI version since it's the CLI with serve added)
bundled:
	@mkdir -p $(DIST)
	cd webapp/localhost && go build -ldflags "-X main.Version=$(CLI_VERSION)" -o ../../$(DIST)/polis-full ./cmd/polis-full
	@echo "Built $(DIST)/polis-full (version $(CLI_VERSION))"

# Run all tests
test:
	cd cli-go && go test ./...
	cd webapp/localhost && go test ./...

# Clean build artifacts
clean:
	rm -rf $(DIST)
	rm -f cli-go/polis
	rm -f webapp/localhost/server
	rm -f webapp/localhost/polis-full

# Cross-compilation targets for releases
.PHONY: release-cli release-webapp release-bundled release

release-cli:
	@mkdir -p $(DIST)
	cd cli-go && GOOS=linux GOARCH=amd64 go build -ldflags "-X main.Version=$(CLI_VERSION)" -o ../$(DIST)/polis-linux-amd64 ./cmd/polis
	cd cli-go && GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.Version=$(CLI_VERSION)" -o ../$(DIST)/polis-darwin-amd64 ./cmd/polis
	cd cli-go && GOOS=darwin GOARCH=arm64 go build -ldflags "-X main.Version=$(CLI_VERSION)" -o ../$(DIST)/polis-darwin-arm64 ./cmd/polis
	cd cli-go && GOOS=windows GOARCH=amd64 go build -ldflags "-X main.Version=$(CLI_VERSION)" -o ../$(DIST)/polis-windows-amd64.exe ./cmd/polis
	@echo "Built CLI release binaries"

release-webapp:
	@mkdir -p $(DIST)
	cd webapp/localhost && GOOS=linux GOARCH=amd64 go build -ldflags "-X main.Version=$(CLI_VERSION)" -o ../../$(DIST)/polis-server-linux-amd64 ./cmd/server
	cd webapp/localhost && GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.Version=$(CLI_VERSION)" -o ../../$(DIST)/polis-server-darwin-amd64 ./cmd/server
	cd webapp/localhost && GOOS=darwin GOARCH=arm64 go build -ldflags "-X main.Version=$(CLI_VERSION)" -o ../../$(DIST)/polis-server-darwin-arm64 ./cmd/server
	cd webapp/localhost && GOOS=windows GOARCH=amd64 go build -ldflags "-X main.Version=$(CLI_VERSION)" -o ../../$(DIST)/polis-server-windows-amd64.exe ./cmd/server
	@echo "Built webapp release binaries"

release-bundled:
	@mkdir -p $(DIST)
	cd webapp/localhost && GOOS=linux GOARCH=amd64 go build -ldflags "-X main.Version=$(CLI_VERSION)" -o ../../$(DIST)/polis-full-linux-amd64 ./cmd/polis-full
	cd webapp/localhost && GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.Version=$(CLI_VERSION)" -o ../../$(DIST)/polis-full-darwin-amd64 ./cmd/polis-full
	cd webapp/localhost && GOOS=darwin GOARCH=arm64 go build -ldflags "-X main.Version=$(CLI_VERSION)" -o ../../$(DIST)/polis-full-darwin-arm64 ./cmd/polis-full
	cd webapp/localhost && GOOS=windows GOARCH=amd64 go build -ldflags "-X main.Version=$(CLI_VERSION)" -o ../../$(DIST)/polis-full-windows-amd64.exe ./cmd/polis-full
	@echo "Built bundled release binaries"

release: release-cli release-webapp release-bundled
