# Stage 1: Build
FROM golang:alpine AS builder

# Install git + SSL ca certificates + WebP dependencies
# Git is required for fetching Go dependencies.
# Ca-certificates is required to call HTTPS endpoints.
# libwebp-dev, gcc, musl-dev are required for github.com/chai2010/webp (CGO)
RUN apk update && apk add --no-cache git ca-certificates tzdata libwebp-dev gcc musl-dev && update-ca-certificates

WORKDIR /app

# Copy go.mod and go.sum first to leverage Docker cache
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build the binary
# CGO_ENABLED=1: Required for WebP library
# -ldflags="-w -s": Strip debug information to reduce binary size
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-w -s" -o main ./cmd/api/main.go

# Stage 2: Production Runtime
FROM alpine:latest

WORKDIR /root/

# Install CA certificates, timezone data, and libwebp (for runtime CGO linking)
RUN apk --no-cache add ca-certificates tzdata libwebp

# Copy binary from builder
COPY --from=builder /app/main .

# Expose port
EXPOSE 8080

CMD ["./main"]
