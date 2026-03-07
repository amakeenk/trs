# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "v0.1.0")
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get

# Binary names
BINARY_NAME=trs
BINARY_UNIX=$(BINARY_NAME)_unix

# Build flags
LDFLAGS=-ldflags "-X altlinux.space/amakeenk/trs/internal/version.Version=$(VERSION) \
	-X altlinux.space/amakeenk/trs/internal/version.GitCommit=$(GIT_COMMIT) \
	-X altlinux.space/amakeenk/trs/internal/version.BuildDate=$(BUILD_DATE)"

.PHONY: all build clean test coverage install uninstall version

all: clean build

build:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) .

test:
	$(GOTEST) ./... -v

coverage:
	$(GOTEST) ./... -coverprofile=coverage.out -covermode=atomic
	go tool cover -func=coverage.out

coverage-html:
	$(GOTEST) ./... -coverprofile=coverage.out -covermode=atomic
	go tool cover -html=coverage.out

clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f coverage.out

install:
	$(GOBUILD) $(LDFLAGS) -o $(GOPATH)/bin/$(BINARY_NAME) .

uninstall:
	rm -f $(GOPATH)/bin/$(BINARY_NAME)

# Build for multiple platforms
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-amd64 .

build-darwin:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-amd64 .
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-arm64 .

build-windows:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_NAME)-windows-amd64.exe .

build-all: build-linux build-darwin build-windows

# Show version
version:
	@echo "Version: $(VERSION)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Build Date: $(BUILD_DATE)"
