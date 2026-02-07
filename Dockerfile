# Build stage
FROM golang:1.24.3-alpine AS builder

# Install build dependencies (git for Go, node/npm for Tailwind CSS)
RUN apk add --no-cache git nodejs npm

# Set working directory
WORKDIR /build

# Copy full app (build context = app root: directory with go.mod, cmd/, internal/, web/)
COPY . .

# Ensure .env exists so final stage copy never fails (empty file if not in context, e.g. CI)
RUN [ -f .env ] || touch .env

# Populate vendor inside the image (no host vendor needed)
RUN go mod tidy && go mod vendor

# Build static assets for production (Tailwind CSS + htmx, no CDNs)
RUN npm install && \
    mkdir -p web/static/css web/static/js && \
    npx tailwindcss -i ./web/css/input.css -o ./web/static/css/styles.css --minify && \
    cp node_modules/htmx.org/dist/htmx.min.js web/static/js/htmx.min.js

# Build the application using vendor
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -a -installsuffix cgo -o server ./cmd/server

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS and wget (healthcheck + dbmate download)
RUN apk --no-cache add ca-certificates wget

# Install dbmate for running migrations on startup
ENV DBMATE_VERSION=2.28.0
RUN wget -q -O /usr/local/bin/dbmate "https://github.com/amacneil/dbmate/releases/download/v${DBMATE_VERSION}/dbmate-linux-amd64" && \
    chmod +x /usr/local/bin/dbmate

WORKDIR /app

# Copy binary, web assets, db migrations, and root .env (override via Railway dashboard in production)
COPY --from=builder /build/server .
COPY --from=builder /build/web ./web
COPY --from=builder /build/db ./db
COPY --from=builder /build/.env .

# Create non-root user
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser && \
    chown -R appuser:appuser /app

USER appuser

# dbmate config (DATABASE_URL set at runtime, e.g. Railway)
ENV DBMATE_MIGRATIONS_DIR=/app/db/migrations
ENV DBMATE_NO_DUMP_SCHEMA=true
ENV DBMATE_WAIT=true
ENV DBMATE_WAIT_TIMEOUT=60s

# Expose port
EXPOSE 3000

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:3000/health || exit 1

# Run migrations then start the server (DATABASE_URL must be set at runtime)
CMD ["sh", "-c", "dbmate up && exec ./server"]
