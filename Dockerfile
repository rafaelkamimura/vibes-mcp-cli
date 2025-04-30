# Stage 1: Build the Go binary
FROM golang:1.20-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -ldflags="-s -w" -o openai-cli .

# Stage 2: Create a minimal runtime image
FROM alpine:latest
RUN apk add --no-cache ca-certificates
WORKDIR /root/
COPY --from=builder /app/openai-cli .
COPY .env_example .env
ENTRYPOINT ["./openai-cli"]