FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

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
# Copy .env.example as default .env
COPY --from=builder /app/.env.example .env
# Copy init.sql for database initialization
COPY --from=builder /app/init.sql .

EXPOSE 8080

CMD ["./main"]
