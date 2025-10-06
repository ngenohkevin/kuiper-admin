# Ganymede Admin Dashboard Makefile

# Go parameters
BINARY_NAME=ganymede-admin
MAIN_PACKAGE=./cmd
GOBUILD=go build
GORUN=go run
GOCLEAN=go clean
GOTEST=go test
GOGET=go get
GOMOD=go mod
TEMPL=templ

# Docker parameters
DOCKER_IMAGE=kuiper_admin
DOCKER_CONTAINER=kuiper_admin_container
DOCKER_BUILD=docker build
DOCKER_RUN=docker run
DOCKER_STOP=docker stop
DOCKER_RM=docker rm
DOCKER_COMPOSE=docker compose

# Build flags
BUILD_FLAGS=-v

# Environment
ENV_FILE=.env

.PHONY: all build run clean deps test help templ setup_dev docker-build docker-run docker-stop docker-clean docker-compose-up docker-compose-down

# Default target when just running 'make'
all: templ build

# Generate templ files
templ:
	@echo "Generating templ files..."
	$(TEMPL) generate

# Build the binary
build: templ
	@echo "Building $(BINARY_NAME)..."
	$(GOBUILD) $(BUILD_FLAGS) -o $(BINARY_NAME) $(MAIN_PACKAGE)

# Run the application
run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BINARY_NAME)

# Clean build files
clean:
	@echo "Cleaning build files..."
	$(GOCLEAN)
	rm -f $(BINARY_NAME)

# Install all dependencies
deps:
	@echo "Installing dependencies..."
	$(GOGET) github.com/a-h/templ/cmd/templ@latest
	$(GOMOD) tidy

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

# Set up development environment
setup_dev: deps
	@echo "Setting up development environment..."
	@if [ ! -f $(ENV_FILE) ]; then \
		echo "Creating $(ENV_FILE) file..."; \
		echo "DB_NAME=ganymede" > $(ENV_FILE); \
		echo "DB_USER=postgres.uevejvahmnfmfcsmdqdx" >> $(ENV_FILE); \
		echo "DB_PASSWORD=rcdsjfx5wWpcWV7m" >> $(ENV_FILE); \
		echo "DB_HOST=aws-0-eu-central-1.pooler.supabase.com" >> $(ENV_FILE); \
		echo "DB_PORT=5432" >> $(ENV_FILE); \
		echo "" >> $(ENV_FILE); \
		echo "# Database connection string (used when MOCK_DB=false)" >> $(ENV_FILE); \
		echo "DATABASE_URL=postgresql://postgres.uevejvahmnfmfcsmdqdx:rcdsjfx5wWpcWV7m@aws-0-eu-central-1.pooler.supabase.com:5432/postgres" >> $(ENV_FILE); \
		echo "" >> $(ENV_FILE); \
		echo "PORT=8090" >> $(ENV_FILE); \
	fi
	@mkdir -p web/static/css
	@mkdir -p web/static/js
	@mkdir -p migrations

# Run the application and open in browser
serve: build
	@echo "Running the application and opening in browser..."
	./run.sh

# Build Docker image
docker-build:
	@echo "Building Docker image $(DOCKER_IMAGE)..."
	$(DOCKER_BUILD) -t $(DOCKER_IMAGE) .

# Run Docker container
docker-run: docker-build
	@echo "Running Docker container $(DOCKER_CONTAINER)..."
	$(DOCKER_RUN) -d -p 8090:8090 --name $(DOCKER_CONTAINER) $(DOCKER_IMAGE)
	@echo "Container started. Access the application at http://localhost:8090"

# Stop Docker container
docker-stop:
	@echo "Stopping Docker container $(DOCKER_CONTAINER)..."
	-$(DOCKER_STOP) $(DOCKER_CONTAINER) 2>/dev/null || true

# Remove Docker container
docker-clean: docker-stop
	@echo "Removing Docker container $(DOCKER_CONTAINER)..."
	-$(DOCKER_RM) $(DOCKER_CONTAINER) 2>/dev/null || true

# Start application with Docker Compose
docker-compose-up:
	@echo "Starting application with Docker Compose..."
	$(DOCKER_COMPOSE) up -d

# Stop application with Docker Compose
docker-compose-down:
	@echo "Stopping application with Docker Compose..."
	$(DOCKER_COMPOSE) down

# Display help information
help:
	@echo "Available targets:"
	@echo "  all                - Generate templ files and build the application (default)"
	@echo "  templ              - Generate templ files from templates"
	@echo "  build              - Build the application"
	@echo "  run                - Run the application"
	@echo "  serve              - Run the application and open in browser"
	@echo "  clean              - Remove build artifacts"
	@echo "  deps               - Install dependencies"
	@echo "  test               - Run tests"
	@echo "  setup_dev          - Set up development environment"
	@echo "  docker-build       - Build Docker image"
	@echo "  docker-run         - Build and run Docker container"
	@echo "  docker-stop        - Stop Docker container"
	@echo "  docker-clean       - Stop and remove Docker container"
	@echo "  docker-compose-up  - Start application with Docker Compose"
	@echo "  docker-compose-down- Stop application with Docker Compose"
	@echo "  help               - Display this help message"
