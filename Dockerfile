# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies (gcc needed for CGO/mattn/go-sqlite3)
RUN apk add --no-cache git gcc musl-dev ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build static binary (CGO_ENABLED=1 needed for go-sqlite3)
ARG TARGETOS TARGETARCH
RUN CGO_ENABLED=1 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -ldflags="-w -s" \
    -o /app/mdict-server \
    ./cmd/server

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata wget

# Create non-root user
RUN addgroup -S mdict && adduser -S mdict -G mdict

# Create directories
RUN mkdir -p /dicts /data && \
    chown -R mdict:mdict /dicts /data

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/mdict-server .

# Copy templates
COPY --from=builder /app/templates ./templates

# Set ownership
RUN chown -R mdict:mdict /app

# Switch to non-root user
USER mdict

# Expose port
EXPOSE 8080

# Volumes
VOLUME ["/dicts", "/data"]

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/v1/health || exit 1

# Run
ENTRYPOINT ["./mdict-server"]
