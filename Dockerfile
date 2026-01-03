# Build stage
FROM golang:1.22-alpine AS builder

ARG VERSION=dev
ARG TARGETOS
ARG TARGETARCH

RUN apk add --no-cache gcc musl-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags="-s -w -X main.version=${VERSION}" -o apigate ./cmd/apigate

# Runtime stage
FROM alpine:3.23

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/apigate .
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /app/configs/apigate.example.yaml ./apigate.example.yaml

# Create data directory
RUN mkdir -p /app/data

# Default ports
EXPOSE 8080 8081 8082

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

ENTRYPOINT ["./apigate"]
CMD ["-config", "/app/apigate.yaml"]
