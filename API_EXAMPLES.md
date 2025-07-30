# API Usage Examples

This document provides comprehensive examples of using the Go API Proxy, including request/response examples and common use cases.

## Base URL

All examples assume the proxy is running on `http://localhost` (port 80). Adjust the base URL according to your configuration.

## Health Check Endpoint

### Request

```bash
curl -X GET http://localhost/health
```

### Response

```json
{
  "status": "healthy",
  "service": "go-api-proxy"
}
```

**Status Code**: `200 OK`

## Token Endpoint (Filtered)

The `/api/v2/tokens` endpoint returns filtered token data based on the whitelist configuration.

### Basic Request

```bash
curl -X GET http://localhost/api/v2/tokens
```

### Response (Filtered)

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
    },
    {
      "address": "0x7254B7303A9d5d0A2F232eB62B0B27a06E068Ac7",
      "address_hash": "0x7254B7303A9d5d0A2F232eB62B0B27a06E068Ac7",
      "circulating_market_cap": "500000.25",
      "decimals": "6",
      "exchange_rate": "0.85",
      "holders": "750",
      "holders_count": "750",
      "icon_url": "https://example.com/icon2.png",
      "name": "Another Token",
      "symbol": "ANT",
      "total_supply": "500000000000",
      "type": "ERC-20",
      "volume_24h": "25000.30"
    }
  ]
}
```

**Status Code**: `200 OK`

### Request with Query Parameters

```bash
curl -X GET "http://localhost/api/v2/tokens?limit=10&offset=0"
```

Query parameters are forwarded to the backend API, but the response is still filtered by the whitelist.

### Empty Whitelist Response

If no tokens match the whitelist:

```json
{
  "items": []
}
```

**Status Code**: `200 OK`

## Pass-through Endpoints

All other endpoints are proxied directly to the backend without modification.

### Account Information

```bash
curl -X GET http://localhost/api/v2/accounts/0x1234567890abcdef1234567890abcdef12345678
```

**Response**: Direct response from backend API

### Transaction Data

```bash
curl -X GET http://localhost/api/v2/transactions/0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab
```

**Response**: Direct response from backend API

### Block Information

```bash
curl -X GET http://localhost/api/v2/blocks/latest
```

**Response**: Direct response from backend API

## Error Responses

### Backend Unreachable

When the backend API is not accessible:

```bash
curl -X GET http://localhost/api/v2/tokens
```

**Response**:
```json
{
  "error": "Bad Gateway",
  "message": "Backend API is unreachable"
}
```

**Status Code**: `502 Bad Gateway`

### Invalid Backend Response

When the backend returns invalid JSON:

```bash
curl -X GET http://localhost/api/v2/tokens
```

**Response**:
```json
{
  "error": "Bad Gateway",
  "message": "Invalid response from backend API"
}
```

**Status Code**: `502 Bad Gateway`

### Backend Error Response

When the backend returns an error (e.g., 404, 500):

```bash
curl -X GET http://localhost/api/v2/nonexistent-endpoint
```

**Response**: Direct error response from backend API with original status code

## Advanced Usage Examples

### Using with Different HTTP Methods

#### POST Request (Pass-through)

```bash
curl -X POST http://localhost/api/v2/some-endpoint \
  -H "Content-Type: application/json" \
  -d '{"key": "value"}'
```

#### PUT Request (Pass-through)

```bash
curl -X PUT http://localhost/api/v2/some-endpoint/123 \
  -H "Content-Type: application/json" \
  -d '{"updated": "data"}'
```

### Custom Headers

Headers are forwarded to the backend API:

```bash
curl -X GET http://localhost/api/v2/tokens \
  -H "Authorization: Bearer your-token" \
  -H "X-Custom-Header: custom-value"
```

### Request with User Agent

```bash
curl -X GET http://localhost/api/v2/tokens \
  -H "User-Agent: MyApp/1.0"
```

## Testing Scenarios

### Test Whitelist Filtering

1. **Setup**: Create a whitelist with specific addresses
2. **Request**: Call `/api/v2/tokens`
3. **Verify**: Only tokens with whitelisted addresses are returned

```bash
# Example whitelist.json
{
  "addresses": ["0x5db2B3f16E1a28ad4fe1229a2dc01f264a3f0614"]
}

# Test request
curl -X GET http://localhost/api/v2/tokens

# Expected: Only tokens with the whitelisted address
```

### Test Dynamic Whitelist Updates

1. **Initial Request**: Call `/api/v2/tokens` and note the results
2. **Update Whitelist**: Modify `whitelist.json` file
3. **Subsequent Request**: Call `/api/v2/tokens` again
4. **Verify**: Results reflect the updated whitelist (no restart required)

### Test Error Handling

#### Missing Whitelist File

```bash
# Remove whitelist file
mv whitelist.json whitelist.json.backup

# Make request
curl -X GET http://localhost/api/v2/tokens

# Expected: All tokens returned (no filtering)
# Check logs for warning message
```

#### Invalid Whitelist Format

```bash
# Create invalid JSON
echo "invalid json" > whitelist.json

# Make request
curl -X GET http://localhost/api/v2/tokens

# Expected: All tokens returned (no filtering)
# Check logs for error message
```

## Performance Testing

### Load Testing with curl

```bash
# Simple load test
for i in {1..100}; do
  curl -s http://localhost/api/v2/tokens > /dev/null &
done
wait
```

### Concurrent Requests

```bash
# Test concurrent filtering
curl -X GET http://localhost/api/v2/tokens &
curl -X GET http://localhost/api/v2/tokens &
curl -X GET http://localhost/api/v2/tokens &
wait
```

## Integration Examples

### JavaScript/Node.js

```javascript
const axios = require('axios');

async function getFilteredTokens() {
  try {
    const response = await axios.get('http://localhost/api/v2/tokens');
    console.log('Filtered tokens:', response.data.items);
  } catch (error) {
    console.error('Error:', error.response?.data || error.message);
  }
}

getFilteredTokens();
```

### Python

```python
import requests

def get_filtered_tokens():
    try:
        response = requests.get('http://localhost/api/v2/tokens')
        response.raise_for_status()
        tokens = response.json()['items']
        print(f"Found {len(tokens)} filtered tokens")
        return tokens
    except requests.exceptions.RequestException as e:
        print(f"Error: {e}")
        return []

tokens = get_filtered_tokens()
```

### Go

```go
package main

import (
    "encoding/json"
    "fmt"
    "net/http"
)

type TokenResponse struct {
    Items []map[string]interface{} `json:"items"`
}

func getFilteredTokens() error {
    resp, err := http.Get("http://localhost/api/v2/tokens")
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    var tokenResp TokenResponse
    if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
        return err
    }

    fmt.Printf("Found %d filtered tokens\n", len(tokenResp.Items))
    return nil
}
```

## Monitoring and Observability

### Health Check Monitoring

```bash
# Simple health check script
#!/bin/bash
while true; do
  if curl -f http://localhost/health > /dev/null 2>&1; then
    echo "$(date): Service is healthy"
  else
    echo "$(date): Service is down"
  fi
  sleep 30
done
```

### Request Tracing

Each request gets a unique request ID that appears in logs. Use this for tracing requests through the system.

### Metrics Collection

Monitor these key metrics:
- Request count per endpoint
- Response times
- Error rates
- Whitelist file reload frequency
- Backend API response times

## Troubleshooting Common Issues

### No Tokens Returned

**Problem**: `/api/v2/tokens` returns empty array

**Possible Causes**:
1. Whitelist is too restrictive
2. Backend API returns no tokens
3. Token addresses don't match whitelist format

**Debug Steps**:
1. Check whitelist file content
2. Test backend API directly
3. Compare token addresses with whitelist entries

### Slow Response Times

**Problem**: Requests take too long to complete

**Possible Causes**:
1. Backend API is slow
2. Network latency
3. Large whitelist file

**Debug Steps**:
1. Test backend API response time directly
2. Check network connectivity
3. Monitor whitelist file size and load time