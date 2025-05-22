FROM golang:1.24-alpine AS builder

# Set working directory
WORKDIR /app

# Install necessary build tools
RUN apk add --no-cache git

# Copy go.mod and go.sum files first for better caching
COPY go.mod go.sum* ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application with optimizations
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o cloud-bootstrap .

# Use a minimal alpine image for the final container
FROM alpine:3.20

# Install CA certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Set working directory
WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/cloud-bootstrap .

# Create a directory for configuration
RUN mkdir -p /app/config

# Set the entrypoint
ENTRYPOINT ["/app/cloud-bootstrap"]

# Default command (can be overridden)
# CMD ["--config", "/app/config/aws-resources.yaml"]
