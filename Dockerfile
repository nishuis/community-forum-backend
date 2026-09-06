# Build stage
FROM golang:1.26.5-alpine AS builder

# Go module 代理：国内网络直连 proxy.golang.org 不通，默认走 goproxy.cn
# 海外构建可用 --build-arg GOPROXY=... 覆盖
ARG GOPROXY=https://goproxy.cn,direct
ENV GOPROXY=${GOPROXY}

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files and download dependencies (layer caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o app main.go

# Runtime stage
FROM alpine:3.20

WORKDIR /app

# Install runtime dependencies (ca-certificates for HTTPS)
RUN apk add --no-cache ca-certificates

# Copy binary from builder
COPY --from=builder /build/app .

# Copy config directory
COPY configs ./configs

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --quiet --tries=1 --spider http://localhost:8080/api/v1/posts/search || exit 1

# Run application
CMD ["./app"]
