VERSION ?= $(shell git describe --tags --always --dirty)

.PHONY: test test-unit test-integration run clean

# Default target
all: test run

# Build Docker image
docker-build:
	docker build -t cloud-bootstrap .

docker-push:
	docker buildx build \
		--platform linux/amd64,linux/arm64 \
		--push \
		--tag wang/cloud-bootstrap:$(VERSION) \
		--tag wang/cloud-bootstrap:latest \
		.


# Run the application
run:
	go run main.go

# Run all tests
test: test-unit test-integration

# Run unit tests only
test-unit:
	go test -v ./...

# Run integration tests (requires AWS credentials)
test-integration:
	TEST_INTEGRATION=1 TEST_RUN_ID=$$(date +%s) go test -v ./test -run TestS3BucketIntegration

# Run with a dry-run flag (doesn't modify AWS resources)
dry-run:
	go run main.go --dry-run

# Run with a specific config file
run-with-config:
	go run main.go --config aws-resources-test.yaml

# Clean up generated files
clean:
	rm -f coverage.out

# Generate test coverage report
coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

# Install dependencies
deps:
	go mod tidy

# Lint the code
lint:
	go vet ./...
	@if command -v golint > /dev/null; then \
		golint ./...; \
	else \
		echo "golint not installed. Run: go install golang.org/x/lint/golint@latest"; \
	fi

# Help target
help:
	@echo "Available targets:"
	@echo "  run              - Run the application"
	@echo "  test             - Run all tests"
	@echo "  test-unit        - Run unit tests only"
	@echo "  test-integration - Run integration tests (requires AWS credentials)"
	@echo "  dry-run          - Run with dry-run flag (doesn't modify AWS resources)"
	@echo "  run-with-config  - Run with a specific config file"
	@echo "  clean            - Clean up generated files"
	@echo "  coverage         - Generate test coverage report"
	@echo "  deps             - Install dependencies"
	@echo "  lint             - Lint the code"
	@echo "  help             - Show this help message"
