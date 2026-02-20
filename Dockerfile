# syntax=docker/dockerfile:1

# --- Build stage ---
FROM golang:1.25.6-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /app ./cmd/server

# --- Runtime stage ---
FROM gcr.io/distroless/static-debian12:nonroot

LABEL org.opencontainers.image.title="go-service-template-v2" \
      org.opencontainers.image.source="https://github.com/jsamuelsen11/go-service-template-v2"

COPY --from=builder /app /app
COPY --chown=nonroot:nonroot configs/ /configs/

WORKDIR /
EXPOSE 8080

ENTRYPOINT ["/app"]
