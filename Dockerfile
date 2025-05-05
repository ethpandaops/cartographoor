FROM golang:1.24-alpine AS builder

# Install dependencies
RUN apk add --no-cache git make bash

# Set working directory
WORKDIR /app

# Copy Go module files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -o network-status ./cmd/network-status

# Use a minimal alpine image for the final image
FROM alpine:3.19

# Install ca-certificates for HTTPS
RUN apk add --no-cache ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/network-status /app/network-status

# Create a directory for config
RUN mkdir -p /app/config

# Expose the port the app runs on
EXPOSE 8080

# Command to run
ENTRYPOINT ["/app/network-status"]
CMD ["run", "--config=/app/config/config.yaml"]