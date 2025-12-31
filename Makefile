.PHONY: build test clean deploy validate lint deps

# Variables
BINARY_NAME := bootstrap
LAMBDA_DIR := cmd/lambda
BUILD_DIR := .aws-sam/build
GO_VERSION := 1.22
GOOS := linux
GOARCH := arm64

# Default target
all: deps lint test build

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

# Run linter
lint:
	@echo "Running linter..."
	@if command -v golangci-lint &> /dev/null; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed, skipping..."; \
	fi

# Run tests
test:
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

# Build Lambda binary
build:
	@echo "Building Lambda binary for $(GOOS)/$(GOARCH)..."
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=0 go build \
		-ldflags="-s -w" \
		-o $(LAMBDA_DIR)/$(BINARY_NAME) \
		./$(LAMBDA_DIR)/

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	rm -f $(LAMBDA_DIR)/$(BINARY_NAME)
	rm -f coverage.out

# Validate SAM template
validate:
	@echo "Validating SAM template..."
	sam validate --template infrastructure/template.yaml --lint

# Build with SAM
sam-build: build
	@echo "Building with SAM..."
	sam build --template infrastructure/template.yaml

# Deploy with SAM (local - for testing only)
sam-deploy: sam-build
	@echo "Deploying with SAM..."
	@echo "WARNING: Production deployments should go through GitHub Actions!"
	sam deploy \
		--template-file infrastructure/template.yaml \
		--stack-name headset-agent-dev \
		--capabilities CAPABILITY_IAM CAPABILITY_NAMED_IAM \
		--no-confirm-changeset

# Sync knowledge base documents
sync-kb:
	@echo "Syncing knowledge base documents..."
	./scripts/sync-knowledge-base.sh

# Create/update Bedrock agents
create-agents:
	@echo "Creating Bedrock agents..."
	python scripts/create-agents.py --environment $(ENV)

# Run locally with SAM
local:
	@echo "Starting local API..."
	sam local start-api --template infrastructure/template.yaml

# Format code
fmt:
	@echo "Formatting code..."
	gofmt -w .

# Check for security issues
security:
	@echo "Running security checks..."
	@if command -v gosec &> /dev/null; then \
		gosec ./...; \
	else \
		echo "gosec not installed, skipping..."; \
	fi

# Generate mock files for testing
mocks:
	@echo "Generating mocks..."
	@if command -v mockgen &> /dev/null; then \
		go generate ./...; \
	else \
		echo "mockgen not installed, skipping..."; \
	fi

# Help
help:
	@echo "Available targets:"
	@echo "  deps          - Install Go dependencies"
	@echo "  lint          - Run golangci-lint"
	@echo "  test          - Run unit tests"
	@echo "  build         - Build Lambda binary"
	@echo "  clean         - Clean build artifacts"
	@echo "  validate      - Validate SAM template"
	@echo "  sam-build     - Build with SAM"
	@echo "  sam-deploy    - Deploy with SAM (dev only)"
	@echo "  sync-kb       - Sync knowledge base to S3"
	@echo "  create-agents - Create Bedrock agents"
	@echo "  local         - Run locally with SAM"
	@echo "  fmt           - Format Go code"
	@echo "  security      - Run security checks"
	@echo "  mocks         - Generate test mocks"
