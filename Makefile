.PHONY: build test test-v fmt vet lint clean help

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOFMT=gofmt
GOVET=$(GOCMD) vet
GOLINT=golangci-lint

# Package name
PACKAGE=go-jinja

# Build directory
BUILD_DIR=./bin

all: test build

## build: Build the package
build:
	$(GOBUILD) ./...

## test: Run tests
test:
	$(GOTEST) -v ./... -count=1

## test-v: Run tests with verbose output
test-v:
	$(GOTEST) -v ./... -count=1

## test-cover: Run tests with coverage
test-cover:
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

## fmt: Format code
fmt:
	$(GOFMT) -w -s .

## vet: Run go vet
vet:
	$(GOVET) ./...

## lint: Run golangci-lint
lint:
	$(GOLINT) run ./...

## clean: Clean build artifacts
clean:
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

## tidy: Tidy go modules
tidy:
	$(GOCMD) mod tidy

## deps: Download dependencies
deps:
	$(GOCMD) mod download

## help: Show this help
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'