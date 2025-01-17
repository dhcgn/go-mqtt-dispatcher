FROM golang:1.23-alpine AS builder
WORKDIR /app

# Define build arguments
ARG VERSION=v0.0.0
ARG COMMIT=unknown
ARG BUILDTIME=unknown

COPY . .

# Build the application with version info
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "\
    -X main.Version=${VERSION} \
    -X main.Commit=${COMMIT} \
    -X main.BuildTime=${BUILDTIME}" \
    -o go-mqtt-dispatcher main.go

FROM gcr.io/distroless/static-debian12
WORKDIR /app
COPY --from=builder /app/go-mqtt-dispatcher /app/
ENTRYPOINT ["/app/go-mqtt-dispatcher", "-config", "/app/config/config.yaml"]
