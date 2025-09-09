# Use the official Go image as the base image
FROM golang:1.23.6-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Install git and ca-certificates (needed for Go modules)
RUN apk add --no-cache git ca-certificates

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o opensoho .

# Use a minimal Alpine image for the final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Create a non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Set the working directory
WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/opensoho .

# Copy embedded files and migrations
COPY --from=builder /app/pb_migrations ./pb_migrations
COPY --from=builder /app/favicon.png .
COPY --from=builder /app/logo.svg .

# Change ownership to the non-root user
RUN chown -R appuser:appgroup /app

# Switch to the non-root user
USER appuser

# Expose the port
EXPOSE 8090

# Run the application
CMD ["./opensoho", "serve", "--http", "0.0.0.0:8090"]

