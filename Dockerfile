# Multi-stage Dockerfile for Helios Docker Management Dashboard
# Builds React UI + Go backend, serves with NGINX + supervisord

# Stage 1: Build React UI
FROM node:20-alpine AS ui-builder

WORKDIR /app

# Copy package files
COPY helios-ui/package*.json ./

# Install dependencies
RUN npm ci

# Copy UI source
COPY helios-ui/ ./

# Remove .env files to ensure production uses relative paths
RUN rm -f .env .env.local .env.*.local

# Build for production
RUN npm run build

# Stage 2: Build Go backend
FROM golang:1.24-bookworm AS go-builder

# Install build dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc \
    libc6-dev \
    libsqlite3-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy go mod files
COPY helios-server/go.mod helios-server/go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY helios-server/ ./

# Build the application with SQLite support
RUN CGO_ENABLED=1 GOOS=linux go build -tags "sqlite_omit_load_extension" -o helios ./cmd

# Stage 3: Final runtime image with nginx and supervisord
FROM nginx:bookworm

# Install runtime dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    sqlite3 \
    libsqlite3-0 \
    supervisor \
    wget \
    && rm -rf /var/lib/apt/lists/*

# Copy Go binary from builder
COPY --from=go-builder /app/helios /usr/local/bin/helios

# Accept Docker GID as build argument
ARG DOCKER_GID=984

# Create docker group with host GID and helios user
RUN groupadd -g ${DOCKER_GID} docker || true && \
    groupadd -g 1001 helios && \
    useradd -r -u 1001 -g helios -G docker -s /bin/false helios && \
    mkdir -p /app/data && \
    chown -R helios:helios /app

# Copy built UI from ui-builder
COPY --from=ui-builder /app/dist /usr/share/nginx/html

# Copy nginx configuration
COPY config/nginx.conf /etc/nginx/conf.d/default.conf

# Copy supervisor configuration
COPY config/supervisord.conf /etc/supervisord.conf

# Expose port 5000
EXPOSE 5000

# Start supervisor to run both services
CMD ["/usr/bin/supervisord", "-c", "/etc/supervisord.conf"]
