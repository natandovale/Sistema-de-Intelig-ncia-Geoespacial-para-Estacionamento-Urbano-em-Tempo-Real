# Stage 1: Build Go binary
FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /spotsense-api ./cmd/api

# Stage 2: Minimal runtime image
FROM alpine:3.20

WORKDIR /app
COPY --from=builder /spotsense-api /app/spotsense-api

EXPOSE 8080

CMD ["/app/spotsense-api"]
