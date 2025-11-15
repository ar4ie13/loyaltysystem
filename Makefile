# Define variables
BINARY_NAME := gophermart
SRC_DIR := ./cmd/gophermart
BUILD_DIR := bin

# Phony targets to prevent conflicts with files of the same name
.PHONY: all build run clean test cover coverrep mockery clean-mocks clean-bin pg-start pg-stop

# Default target
all: build

# Build the Go application
build:
	@echo "Building $(BINARY_NAME)..."
	go build -o $(BUILD_DIR)/$(BINARY_NAME) $(SRC_DIR)/main.go

# Run the Go application
run: build
	@echo "Running $(BINARY_NAME)..."
	$(BUILD_DIR)/$(BINARY_NAME) -l=debug

# Run the Go application with -d
run-d: build
	@echo "Running $(BINARY_NAME)..."
	$(BUILD_DIR)/$(BINARY_NAME) -l=debug -r "localhost:8081" -d="postgres://gophermart:gophermart@localhost:5432/gophermart?sslmode=disable"

# Clean built artifacts
clean: clean-mocks
	@echo "Cleaning build artifacts..."
	rm -f $(BUILD_DIR)/$(BINARY_NAME)

# Run tests
test:
	@echo "Running tests..."
	go test ./... -v

# Run autotest defined
# ./bin/shortenertest -test.v -test.run=^TestIteration5$ -binary-path=./bin/shortener -server-port=8080

# Simple check test coverage
cover:
	 go test ./... -cover

# Check test coverage with html report
coverrep:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out

# Generate mock by using .mockery.yaml in project root with mockery
mockery: clean-mocks
	go tool mockery

# Clean mocks
clean-mocks:
	rm -rf internal/service/internal/mockery

# Starts postgres docker container
pg-start:
	docker start pg

# Stops postgres docker container
pg-stop:
	docker stop pg

