# Build stage
FROM golang:1.21-alpine AS builder

# Change apk mirror to aliyun
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories

# Install build dependencies for SQLite
RUN apk add --no-cache gcc musl-dev

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application with CGO enabled for SQLite
RUN CGO_ENABLED=1 GOOS=linux go build -o server ./cmd/server

# Runtime stage
FROM alpine:3.19

# Change apk mirror to aliyun
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/server .

# Copy templates
COPY --from=builder /app/templates ./templates

# Create data directory for SQLite database
RUN mkdir -p /data

# Set default environment variables
ENV TIMELOG_DB_PATH=/data/timelog.db
ENV TIMELOG_TZ=UTC
ENV TIMELOG_PORT=8000
ENV TIMELOG_RATE_LIMIT=100

# Expose port
EXPOSE 8000

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8000/healthz || exit 1

# Run the server
CMD ["./server"]
