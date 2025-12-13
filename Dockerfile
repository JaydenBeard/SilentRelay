# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git ca-certificates

# Copy go mod files and download dependencies (cached layer)
COPY go.mod go.sum* ./
RUN go mod download

# Force cache invalidation for source code
ARG BUILD_TIME
# Copy source code
COPY . .

# Build (removed -a flag to enable build cache)
RUN CGO_ENABLED=0 GOOS=linux go build -installsuffix cgo -o chatserver ./cmd/chatserver

# Runtime stage
FROM alpine:3.19

WORKDIR /app

# Install certificates and wget for healthcheck
RUN apk --no-cache add ca-certificates tzdata wget

# Copy binary from builder
COPY --from=builder /app/chatserver .

# Copy secrets entrypoint script
COPY scripts/docker-entrypoint.sh /docker-entrypoint.sh
RUN chmod +x /docker-entrypoint.sh

# Run as non-root user
RUN adduser -D -g '' appuser
USER appuser

EXPOSE 8080

ENTRYPOINT ["/docker-entrypoint.sh"]
CMD ["./chatserver"]

