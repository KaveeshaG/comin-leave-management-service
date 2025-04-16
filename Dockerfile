# Use an official Golang image to build our application
FROM golang:1.16 AS builder

# Set the GOPATH to /app
ENV GOPATH=/app

# Move to /app
WORKDIR /app

# Copy go.mod and go.sum
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN go build -o main ./cmd/server

# Use an official Alpine Linux image for the production environment
FROM alpine:latest

# Move to /app
WORKDIR /app

# Copy the binary from the builder
COPY --from=builder /app/main .

# Expose the port
EXPOSE 8082

CMD ["./main"]
