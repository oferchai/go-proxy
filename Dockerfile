FROM golang:1.21-alpine

WORKDIR /app

# Install required system packages
RUN apk add --no-cache ca-certificates

# Copy go.mod and go.sum first to leverage Docker cache
COPY go.mod ./
COPY go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN go build -o proxy ./proxy2.go

# Expose the proxy ports
EXPOSE 3000 3443

# Run the proxy
CMD ["./proxy"] 