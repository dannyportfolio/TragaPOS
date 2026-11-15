# Multi-stage Distroless Zero-CGO Build for POS & Commerce Engine
FROM golang:1.24-alpine AS builder

WORKDIR /src

# Install git and ca-certificates
RUN apk add --no-cache git ca-certificates tzdata

# Cache go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Compile static zero-CGO binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -extldflags '-static'" \
    -tags timetzdata \
    -o /bin/pos-engine \
    cmd/server/main.go && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -extldflags '-static'" \
    -o /bin/healthcheck \
    cmd/healthcheck/main.go && \
    mkdir -p -m 0775 /data && chown -R 65532:65532 /data

# Production Distroless Image
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

# Copy timezone data & certificates
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /bin/pos-engine /app/pos-engine
COPY --from=builder /bin/healthcheck /app/healthcheck
COPY --from=builder --chown=nonroot:nonroot /data /data

# Default non-root user (65532)
USER nonroot:nonroot

VOLUME ["/data"]

EXPOSE 8080 8081

ENTRYPOINT ["/app/pos-engine"]
