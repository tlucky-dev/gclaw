# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o gclaw ./cmd/main.go

# Final stage
FROM alpine:latest

WORKDIR /app

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user for security
RUN addgroup -g 1000 gclaw && \
    adduser -D -u 1000 -G gclaw gclaw

# Copy binary from builder
COPY --from=builder /app/gclaw .
COPY --from=builder /app/config.example.json config.json

# Create directories for data and logs
RUN mkdir -p /app/data /app/logs /app/cache && \
    chown -R gclaw:gclaw /app

# Switch to non-root user
USER gclaw

# Expose default port for Feishu adapter
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Set environment variables
ENV GCLAW_CONFIG=/app/config.json
ENV GCLAW_DATA_DIR=/app/data
ENV GCLAW_CACHE_DIR=/app/cache
ENV GCLAW_LOG_DIR=/app/logs

# Run the application
ENTRYPOINT ["./gclaw"]
CMD ["-config", "/app/config.json"]
