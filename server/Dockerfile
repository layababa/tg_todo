# Builder stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the applications
RUN CGO_ENABLED=0 GOOS=linux go build -o api ./cmd/api && \
    CGO_ENABLED=0 GOOS=linux go build -o bot ./cmd/bot

# Runtime stage
FROM alpine:3.19

WORKDIR /app

# Install runtime dependencies (ca-certificates for HTTPS)
RUN apk add --no-cache ca-certificates tzdata

# Copy binaries from builder
COPY --from=builder /app/api ./api
COPY --from=builder /app/bot ./bot

# Copy config
COPY --from=builder /app/config ./config

# Expose port (for API)
EXPOSE 8080

# Default command (can be overridden in docker-compose)
CMD ["./api"]
