# Build Stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

# Final Stage
FROM alpine:latest

WORKDIR /root/
RUN apk --no-cache add ca-certificates

# Copy binary from builder
COPY --from=builder /app/main .

EXPOSE 8080

HEALTHCHECK --interval=10s --timeout=3s --retries=5 --start-period=15s \
  CMD wget -qO- http://127.0.0.1:8080/health || exit 1

CMD ["./main"]
