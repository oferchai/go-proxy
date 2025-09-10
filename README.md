# Go-Proxy

Go-Proxy is a lightweight, high-performance HTTP/HTTPS proxy server written in Go. It provides a simple and effective way to monitor and control your network traffic. It includes features such as blacklisting, real-time statistics, and geolocation of requests.

## Features

*   **HTTP/HTTPS Proxying:** Proxies both HTTP and HTTPS traffic.
*   **Blacklisting:** Block requests to specific hosts using a blacklist of regular expressions.
*   **Real-time Statistics:** Collects and stores real-time statistics about network traffic, including the number of requests, bytes transferred, and blocked attempts.
*   **Geolocation:** Tracks the geographic location of requested hosts.
*   **Web-based API:** Exposes a simple REST API to retrieve statistics and geolocation data.
*   **Redis Integration:** Uses Redis to store statistics and geolocation data.
*   **Configurable:** Can be configured using command-line flags.

## API Documentation

The proxy exposes a REST API for retrieving statistics and geolocation data.

### Statistics API

#### Get Daily/Hourly Statistics

*   **Endpoint:** `/api/stats/daily`
*   **Method:** `GET` or `POST`
*   **Description:** Retrieves daily or hourly statistics for a given date range and optional host filter.
*   **`GET` Parameters:**
    *   `from_date`: The start date in `YYYY-MM-DD` format.
    *   `to_date`: The end date in `YYYY-MM-DD` format.
    *   `host_filter` (optional): A string to filter hosts by.
    *   `granularity` (optional): "day" or "hour". Defaults to "day".
*   **`POST` Body:** A JSON object with the same parameters as the `GET` request.
*   **Response:** A JSON object containing `keys` (a list of hosts) and `records` (a map of host stats).

#### Get Hourly Statistics

*   **Endpoint:** `/api/stats/hourly`
*   **Method:** `GET` or `POST`
*   **Description:** Retrieves hourly statistics for a given date and hour range.
*   **`GET` Parameters:**
    *   `date`: The date in `YYYY-MM-DD` format.
    *   `from_hour`: The start hour (0-23).
    *   `to_hour`: The end hour (0-23).
*   **`POST` Body:** A JSON object with the same parameters as the `GET` request.
*   **Response:** A JSON object containing `keys` (a list of hosts) and `records` (a map of host stats).

#### Get Metrics

*   **Endpoint:** `/api/metrics`
*   **Method:** `GET`
*   **Description:** Retrieves metrics for the last hour.
*   **Response:** A JSON object containing transformed host stats.

### Geolocation API

#### Get Geolocation Data

*   **Endpoint:** `/api/geo`
*   **Method:** `GET`
*   **Description:** Retrieves all geolocation data from Redis.
*   **Response:** A JSON object containing a `records` map, where the keys are hosts and the values are `GeoData` objects.

## Configuration

The proxy can be configured using the following command-line flags:

*   `-http-port`: The port for the HTTP proxy (default: 8888).
*   `-https-port`: The port for the HTTPS proxy (default: 8889).
*   `-log-file`: The path to the log file (default: "proxy.log").
*   `-redis-addr`: The address of the Redis server (default: "localhost:6379").
*   `-redis-password`: The password for the Redis server.
*   `-block-file`: The path to the blacklist file.
*   `-geo-enabled`: Enable or disable geolocation (default: false).
*   `-geo-cache-size`: The size of the geolocation in-memory cache (default: 1024).

## Installation and Usage

For detailed deployment instructions, please see the [Deployment Guide](DEPLOYMENT.md).


### Prerequisites

*   Go 1.21 or later
*   Redis

### Building from Source

1.  Clone the repository:
    ```bash
    git clone https://github.com/your-username/go-proxy.git
    cd go-proxy
    ```
2.  Build the proxy:
    ```bash
    go build ./cmd/proxy
    ```

### Running the Proxy

```bash
./proxy -http-port 8888 -https-port 8889 -redis-addr localhost:6379
```

## Future Features

*   **Authentication:** Add support for proxy authentication (e.g., basic auth, OAuth2).
*   **Caching:** Implement a caching mechanism to improve performance for frequently requested resources.
*   **Dashboard:** Create a web-based dashboard to visualize the statistics and geolocation data. The project already contains a `dashboard` directory with some initial work, which could be expanded.
*   **More flexible blacklisting:** Allow for more complex blacklisting rules, such as blocking based on IP address, content type, or request size.
*   **White-listing:** Implement a white-listing feature to only allow requests to specific hosts.
*   **Docker and Docker Compose:** The project already has a `Dockerfile` and `docker-compose.yml`, which could be improved and documented.
