# Deploying Go-Proxy

This document provides instructions on how to deploy the Go-Proxy application.

## Building from Source

### Prerequisites

*   Go 1.21 or later
*   Redis

### Steps

1.  Clone the repository:
    ```bash
    git clone https://github.com/your-username/go-proxy.git
    cd go-proxy
    ```

2.  Build the proxy:
    ```bash
    go build ./cmd/proxy
    ```

3.  Run the proxy:
    ```bash
    ./proxy -http-port 8888 -https-port 8889 -redis-addr localhost:6379
    ```

## Docker Integration

The project includes a `Dockerfile` and a `docker-compose.yml` file to make it easy to run the proxy in a Docker container.

### Building the Docker Image

To build the Docker image, run the following command from the root of the project:

```bash
docker build -t go-proxy .
```

### Running with Docker Compose

The `docker-compose.yml` file starts the proxy and a Redis container.

To start the proxy using Docker Compose, run the following command:

```bash
docker-compose up
```

This will start the proxy on port 8888 (HTTP) and 8889 (HTTPS), and a Redis container.

### Configuration

You can configure the proxy by modifying the `docker-compose.yml` file. The `command` section of the `proxy` service can be used to pass command-line flags to the proxy.
