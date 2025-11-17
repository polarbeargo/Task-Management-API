# Stage 1: Build
FROM golang:1.23-alpine AS builder

# Set the working directory
WORKDIR /app

# Copy go.mod and go.sum from backend directory
COPY backend/go.mod backend/go.sum ./

# Download dependencies
RUN go mod download

# Copy backend source code
COPY backend/ .

# Copy migration files from database-migrations directory
COPY database-migrations/migrations /app/migrations

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o main .

# Stage 2: Run
FROM alpine:latest

# Set the working directory
WORKDIR /app

# Create a new user and group
RUN addgroup -S nonrootgroup && adduser -S nonrootuser -G nonrootgroup

# Create migrations directory
RUN mkdir -p /app/migrations

# Copy the compiled binary from the previous stage
COPY --from=builder /app/main .

# Copy migration files from the builder stage
COPY --from=builder /app/migrations /app/migrations

# Change ownership of the working directory and binary to the new user
RUN chown -R nonrootuser:nonrootgroup /app

# Expose port
EXPOSE 8080

# Switch to the non-root user before running the application
USER nonrootuser:nonrootgroup

# Run the command to start the application
CMD ["./main"]
