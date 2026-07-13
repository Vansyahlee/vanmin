# Build stage
FROM golang:alpine AS builder

WORKDIR /app

# Copy module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the application
# Using CGO_ENABLED=0 for a static binary which is ideal for Alpine
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

# Run stage
FROM alpine:latest

WORKDIR /app

# Copy the built binary from the builder stage
COPY --from=builder /app/main .

# Expose standard backend port
EXPOSE 8080

# Command to run the executable
CMD ["./main"]
