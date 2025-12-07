# Build stage
FROM golang:1.21 AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o /mini-asynq cmd/server/main.go

# Final stage
FROM alpine:latest

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /mini-asynq .

# Copy configuration files if needed
# COPY config.yaml .

# Expose ports
EXPOSE 8080 9090

# Command to run the application
CMD ["./mini-asynq"]