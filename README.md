# Go API Proxy

A lightweight HTTP reverse proxy server written in Go that forwards requests to a configurable backend API with special filtering capabilities for token endpoints.

## Features

- **Reverse Proxy**: Forwards HTTP requests to a configurable backend API
- **Token Filtering**: Filters `/api/v2/tokens` responses based on a whitelist of approved token addresses
- **Configurable**: Environment variable-based configuration
- **Logging**: Structured logging with request tracing
- **Health Checks**: Built-in health check endpoint
- **Graceful Shutdown**: Proper signal handling and graceful server shutdown

## Quick Start

### Prerequisites

- Go 1.24.3 or later
- Access to the target backend API (default: https://exp.co2e.cc)

### Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd go-api-proxy
```

2. Build the application:
```bash
go build -o go-api-proxy
```

3. Run the proxy server:
```bash
./go-api-proxy
```

The server will start on port 80 by default and proxy requests to `https://exp.co2e.cc/api/v2`.

### Docker Usage

#### Using Docker Compose (Recommended)

1. **Production deployment:**
```bash
# Start the service
docker-compose up -d

# View logs
docker-compose logs -f

# Stop the service
docker-compose down
```

2. **Development deployment:**
```bash
# Start development environment (port 8080)
docker-compose -f docker-compose.dev.yml up -d

# View logs
docker-compose -f docker-compose.dev.yml logs -f
```

3. **Using Makefile (if available):**
```bash
# Production
make up
make logs
make down

# Development
make dev
make dev-logs
make dev-down
```

#### Using Docker directly

Build and run with Docker:
```bash
# Build the Docker image
docker build -t go-api-proxy .

# Run the container
docker run -p 80:80 -v $(pwd)/whitelist.json:/app/whitelist.json go-api-proxy
```

#### Environment Configuration with Docker Compose

Create a `.env` file from the template:
```bash
cp .env.example .env
# Edit .env file with your configuration
```

## Configuration

The application is configured using environment variables:

### Environment Variables

| Variable | Description | Default Value | Required |
|----------|-------------|---------------|----------|
| `BACKEND_HOST` | Backend API host URL | `https://exp.co2e.cc` | No |
| `PORT` | Port for the proxy server to listen on | `80` | No |
| `WHITELIST_FILE` | Path to the token whitelist JSON file | `whitelist.json` | No |
| `HTTP_TIMEOUT` | HTTP client timeout in seconds | `30` | No |

### Example Configuration

```bash
export BACKEND_HOST="https://api.example.com"
export PORT="8080"
export WHITELIST_FILE="/path/to/custom-whitelist.json"
export HTTP_TIMEOUT="60"
./go-api-proxy
```

## Whitelist Configuration

The token whitelist is configured via a JSON file that specifies which token addresses should be included in `/api/v2/tokens` responses.

### Whitelist File Format

```json
{
  "_comment": "Whitelist of approved token addresses for the Go API Proxy",
  "_description": "Only tokens with addresses listed here will be returned by the /api/v2/tokens endpoint",
  "addresses": [
    "0x5db2B3f16E1a28ad4fe1229a2dc01f264a3f0614",
    "0x7254B7303A9d5d0A2F232eB62B0B27a06E068Ac7",
    "0xdAC17F958D2ee523a2206206994597C13D831ec7"
  ]
}
```

### Whitelist Behavior

- **Token Filtering**: Only tokens with addresses matching entries in the whitelist are returned
- **Dynamic Loading**: The whitelist is reloaded on each request (no server restart required)
- **Error Handling**: If the whitelist file is missing or invalid, all tokens are returned with a warning logged
- **Case Sensitivity**: Token address matching is case-sensitive

## API Usage

### Health Check

Check if the proxy server is running:

```bash
curl http://localhost/health
```

**Response:**
```json
{
  "status": "healthy",
  "service": "go-api-proxy"
}
```

### Token Endpoint (Filtered)

Get filtered token data:

```bash
curl http://localhost/api/v2/tokens
```

**Response Example:**
```json
{
  "items": [
    {
      "address": "0x5db2B3f16E1a28ad4fe1229a2dc01f264a3f0614",
      "address_hash": "0x5db2B3f16E1a28ad4fe1229a2dc01f264a3f0614",
      "circulating_market_cap": "1000000.50",
      "decimals": "18",
      "exchange_rate": "1.25",
      "holders": "1500",
      "holders_count": "1500",
      "icon_url": "https://example.com/icon.png",
      "name": "Example Token",
      "symbol": "EXT",
      "total_supply": "1000000000000000000000000",
      "type": "ERC-20",
      "volume_24h": "50000.75"
    }
  ]
}
```

### Other Endpoints (Pass-through)

All other endpoints are proxied directly to the backend:

```bash
# Example: Get account information
curl http://localhost/api/v2/accounts/0x123...

# Example: Get transaction data
curl http://localhost/api/v2/transactions/0xabc...
```

## Logging

The application provides structured logging with different log levels:

- **INFO**: General application information
- **DEBUG**: Detailed debugging information
- **ERROR**: Error conditions
- **WARN**: Warning conditions

### Log Format

```
2024-01-15T10:30:45Z [INFO] [main] Starting Go API Proxy server port=80 backend_api=https://exp.co2e.cc/api/v2
2024-01-15T10:30:46Z [INFO] [main] Server started successfully, waiting for shutdown signal
2024-01-15T10:31:00Z [INFO] [main] [req:1705315860123-12345] Incoming request method=GET path=/api/v2/tokens
```

## Error Handling

The proxy handles various error conditions gracefully:

### Backend Unreachable
- **HTTP Status**: 502 Bad Gateway
- **Response**: JSON error message
- **Logging**: Error logged with details

### Invalid Whitelist File
- **Behavior**: Continue operation without filtering
- **Logging**: Warning logged
- **Response**: All tokens returned

### Invalid JSON from Backend
- **HTTP Status**: 502 Bad Gateway
- **Response**: JSON error message
- **Logging**: Error logged with details

## Development

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run integration tests
go test -tags=integration ./...
```

### Building

```bash
# Build for current platform
go build -o go-api-proxy

# Build for Linux
GOOS=linux GOARCH=amd64 go build -o go-api-proxy-linux

# Build with optimizations
go build -ldflags="-s -w" -o go-api-proxy
```

## Troubleshooting

### Common Issues

1. **Port Already in Use**
   ```
   Error: listen tcp :80: bind: address already in use
   ```
   Solution: Change the port using the `PORT` environment variable or stop the conflicting service.

2. **Permission Denied on Port 80**
   ```
   Error: listen tcp :80: bind: permission denied
   ```
   Solution: Run with sudo or use a port > 1024.

3. **Backend Connection Failed**
   ```
   Error: dial tcp: lookup exp.co2e.cc: no such host
   ```
   Solution: Check network connectivity and DNS resolution.

4. **Whitelist File Not Found**
   ```
   WARN: Failed to load whitelist, continuing with empty whitelist
   ```
   Solution: Ensure the whitelist file exists at the specified path.

### Debug Mode

Enable debug logging by setting the log level (implementation-specific).

## License

[Add your license information here]

## Contributing

[Add contribution guidelines here]