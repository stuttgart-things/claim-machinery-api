# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o claim-machinery-api .

# Runtime stage with KCL
FROM ubuntu:24.04

# Install KCL
RUN apt-get update && \
    apt-get install -y wget ca-certificates && \
    wget -q https://kcl-lang.io/script/install-cli.sh -O - | bash && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# Copy binary from builder
COPY --from=builder /app/claim-machinery-api /usr/local/bin/claim-machinery-api

# Create app directory for config
RUN mkdir -p /app/config

# Set working directory
WORKDIR /app

# Expose port
EXPOSE 8080

# Run the application
ENTRYPOINT ["/usr/local/bin/claim-machinery-api"]
