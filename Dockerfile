FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o go-mqtt-dispatcher main.go

FROM gcr.io/distroless/static-debian12
WORKDIR /app
COPY --from=builder /app/go-mqtt-dispatcher /app/
ENTRYPOINT ["/app/go-mqtt-dispatcher", "-config", "/app/config/config.yaml"]
