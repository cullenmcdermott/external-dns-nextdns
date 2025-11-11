# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Copy go mod files
COPY go.mod go.sum* ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
    -ldflags="-s -w -X main.Version=${VERSION:-dev}" \
    -o webhook ./cmd/webhook

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1000 webhook && \
    adduser -D -u 1000 -G webhook webhook

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/webhook .

# Change ownership
RUN chown -R webhook:webhook /app

# Switch to non-root user
USER webhook

# Expose ports
# 8888 is the webhook API port (localhost only in production)
# 8080 is the health check port
EXPOSE 8080 8888

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/healthz || exit 1

# Run the webhook
ENTRYPOINT ["/app/webhook"]
