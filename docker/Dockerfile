# Build stage
FROM golang:1.24.3-alpine AS builder

# Install build dependencies (git for Go, node/npm for Tailwind CSS)
RUN apk add --no-cache git nodejs npm

# Set working directory
WORKDIR /build

# Copy go mod files and vendor directory
COPY go.mod go.sum ./
COPY vendor/ ./vendor/

# Copy source code
COPY . .

# Build static assets for production (Tailwind CSS + htmx, no CDNs)
RUN npm install && \
    mkdir -p web/static/css web/static/js && \
    npx tailwindcss -i ./web/css/input.css -o ./web/static/css/styles.css --minify && \
    cp node_modules/htmx.org/dist/htmx.min.js web/static/js/htmx.min.js

# Build the application using vendor
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -a -installsuffix cgo -o server ./cmd/server

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy binary and web assets (templates + static including built CSS)
COPY --from=builder /build/server .
COPY --from=builder /build/web ./web

# Create non-root user
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser && \
    chown -R appuser:appuser /app

USER appuser

# Expose port
EXPOSE 3000

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:3000/health || exit 1

# Run the application
CMD ["./server"]

