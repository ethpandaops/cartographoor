.PHONY: build run clean test lint

# Build settings
BINARY_NAME=network-status
BUILD_DIR=build
MAIN_PATH=./cmd/network-status

# Build the binary
build:
	@echo "Building network-status..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)

# Run the application
run: build
	@echo "Running network-status..."
	@$(BUILD_DIR)/$(BINARY_NAME) run --config=config.example.yaml

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Run linting
lint:
	@echo "Running linter..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed"; \
		exit 1; \
	fi