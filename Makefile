# Makefile for the Chess Engine Backend WebSocket Server

# Variables
APP_NAME      ?= eng-server
VERSION       ?= 0.1.0
BUILD_DIR     ?= build
BIN_DIR       ?= $(BUILD_DIR)/bin
DOCKER_IMAGE  ?= eng-server:$(VERSION)
SRC_DIR       ?= ./cmd/server

# Go commands and flags
GO            := go
GOBUILD       := $(GO) build -ldflags="-X main.version=$(VERSION)" -o $(BIN_DIR)/$(APP_NAME)
GOTEST        := $(GO) test -v -coverprofile=$(BUILD_DIR)/coverage.out
GOLINT        := golangci-lint run

.PHONY: all build run test lint clean docker-build docker-run coverage

# Default target builds the application.
all: build

# Build the server binary.
build:
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BIN_DIR)
	$(GOBUILD) $(SRC_DIR)

# Run the server binary.
run: build
	@echo "Running $(APP_NAME)..."
	$(BIN_DIR)/$(APP_NAME)

# Run all tests with coverage.
test:
	@echo "Running tests..."
	$(GOTEST) ./...

# Run linter (requires golangci-lint installed).
lint:
	@echo "Running linter..."
	$(GOLINT)

# Clean build artifacts.
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)

# Build a Docker image for the server.
docker-build:
	@echo "Building Docker image $(DOCKER_IMAGE)..."
	docker build -t $(DOCKER_IMAGE) .

# Run the Docker container.
docker-run: docker-build
	@echo "Running Docker container..."
	docker run --rm -p 8080:8080 $(DOCKER_IMAGE)

# Generate and display test coverage report.
coverage:
	@echo "Running tests and generating coverage report..."
	$(GOTEST) ./...
	@echo "Coverage report generated at $(BUILD_DIR)/coverage.out"
