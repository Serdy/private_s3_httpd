# Step 1: Build the Go binary
FROM golang:1.22 AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy the go.mod and go.sum files
COPY src/go.mod src/go.sum ./

# Download the module dependencies
RUN go mod download

# Copy the source code
COPY src/ .

# Build the Go binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /app/private_s3_httpd main.go


# Step 2: Create the final image using distroless
FROM gcr.io/distroless/base-debian10

# Set the working directory inside the container
WORKDIR /app


# Copy the Go binary from the builder stage
COPY --from=builder /app/private_s3_httpd /app/private_s3_httpd


# Expose the port that your application runs on (if any)
# EXPOSE 8080

# Command to run the Go binary
ENTRYPOINT ["/app/private_s3_httpd"]