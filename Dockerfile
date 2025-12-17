# Multi-stage build for MultiExit Proxy

# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git make gcc musl-dev

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build server and client
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o client ./cmd/client

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache \
    iptables \
    iproute2 \
    ca-certificates \
    tzdata

WORKDIR /app

# Copy binaries from builder
COPY --from=builder /build/server .
COPY --from=builder /build/client .

# Copy configs
COPY configs ./configs

# Create directories
RUN mkdir -p /etc/multiexit-proxy /var/log/multiexit-proxy

# Expose ports
EXPOSE 8443 8080

# Set capabilities for network operations (requires --cap-add NET_ADMIN when running)
# We'll handle this in docker-compose.yml

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/health || exit 1

# Default command
CMD ["./server", "-config", "/etc/multiexit-proxy/server.yaml"]

