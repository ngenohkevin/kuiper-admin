# Build stage
FROM golang:1.23-alpine AS builder

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Install templ for template generation
RUN go install github.com/a-h/templ/cmd/templ@latest

# Copy the source code
COPY . .

# Generate templates
RUN templ generate

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o kuiper_admin ./cmd/main.go

# Run stage
FROM alpine:latest

# Set working directory
WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/kuiper_admin .

# Copy the web directory for static files
COPY --from=builder /app/web ./web

# Copy the migrations directory
COPY --from=builder /app/migrations ./migrations


# Expose port
EXPOSE 8090

# Run the app
CMD ["./kuiper_admin"]
