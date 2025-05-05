.PHONY: build run clean test lint

# Build settings
BINARY_NAME=cartographoor
BUILD_DIR=build
MAIN_PATH=./cmd/cartographoor

# Note: We've fully migrated to the cartographoor name
# The legacy network-status name has been deprecated

# Build the binary
build:
	@echo "Building cartographoor..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)

# Run the application
run: build
	@echo "Running cartographoor..."
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