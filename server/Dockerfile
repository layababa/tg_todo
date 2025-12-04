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

# Build the application
# Build the application
# CGO_ENABLED=0 for static binary
RUN CGO_ENABLED=0 GOOS=linux go build -o api ./cmd/api

# Runtime stage
FROM alpine:3.19

WORKDIR /app

# Install runtime dependencies (ca-certificates for HTTPS)
RUN apk add --no-cache ca-certificates tzdata

# Copy binary from builder
COPY --from=builder /app/api .
# Copy config (optional, if you want to bake it in, but usually mounted)
COPY --from=builder /app/config ./config
# Copy migrations (if needed at runtime, though they are embedded)

# Expose port
EXPOSE 8080

# Run the application
CMD ["./api"]
