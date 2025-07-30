# Configuration Guide

This document provides detailed information about configuring the Go API Proxy.

## Environment Variables

### BACKEND_HOST

- **Description**: The backend API host URL that the proxy will forward requests to
- **Default**: `https://exp.co2e.cc`
- **Format**: Must start with `http://` or `https://`
- **Examples**:
  - `https://api.example.com`
  - `http://localhost:3000`
  - `https://staging.api.company.com`

### PORT

- **Description**: The port number for the proxy server to listen on
- **Default**: `80`
- **Format**: Integer between 1 and 65535
- **Examples**:
  - `80` (default HTTP port)
  - `8080` (common development port)
  - `3000` (alternative development port)
- **Note**: Ports below 1024 typically require root privileges on Unix systems

### WHITELIST_FILE

- **Description**: Path to the JSON file containing the token address whitelist
- **Default**: `whitelist.json`
- **Format**: File path (relative or absolute)
- **Examples**:
  - `whitelist.json` (current directory)
  - `/etc/go-api-proxy/whitelist.json` (absolute path)
  - `config/tokens-whitelist.json` (relative path)
- **Behavior**: If file doesn't exist or is invalid, all tokens are returned

### HTTP_TIMEOUT

- **Description**: HTTP client timeout for backend requests in seconds
- **Default**: `30`
- **Format**: Positive integer
- **Examples**:
  - `30` (30 seconds)
  - `60` (1 minute)
  - `120` (2 minutes)
- **Note**: Affects both connection and read timeouts

## Configuration Examples

### Development Environment

```bash
export BACKEND_HOST="http://localhost:3001"
export PORT="8080"
export WHITELIST_FILE="dev-whitelist.json"
export HTTP_TIMEOUT="10"
```

### Production Environment

```bash
export BACKEND_HOST="https://api.production.com"
export PORT="80"
export WHITELIST_FILE="/etc/go-api-proxy/production-whitelist.json"
export HTTP_TIMEOUT="30"
```

### Docker Environment

```dockerfile
ENV BACKEND_HOST=https://api.example.com
ENV PORT=80
ENV WHITELIST_FILE=/app/config/whitelist.json
ENV HTTP_TIMEOUT=45
```

## Whitelist File Configuration

### File Format

The whitelist file must be valid JSON with the following structure:

```json
{
  "addresses": [
    "0x1234567890abcdef1234567890abcdef12345678",
    "0xabcdef1234567890abcdef1234567890abcdef12"
  ]
}
```

### Address Format

- **Format**: Ethereum-style hexadecimal addresses
- **Length**: 42 characters (including '0x' prefix)
- **Case Sensitivity**: Addresses are matched case-sensitively
- **Validation**: No validation is performed on address format

### Optional Fields

The whitelist file supports optional comment fields for documentation:

```json
{
  "_comment": "Production token whitelist",
  "_description": "Approved tokens for public API access",
  "_last_updated": "2024-01-15",
  "addresses": [
    "0x1234567890abcdef1234567890abcdef12345678"
  ]
}
```

### File Permissions

- **Read Access**: The application needs read access to the whitelist file
- **Write Access**: Not required (file is read-only)
- **Recommended Permissions**: `644` (owner read/write, group/others read)

## Configuration Validation

The application validates configuration on startup:

### Backend Host Validation

- Must not be empty
- Must start with `http://` or `https://`
- No additional URL validation is performed

### Port Validation

- Must not be empty
- Must be a valid integer
- Must be between 1 and 65535

### Whitelist File Validation

- Path must not be empty
- File existence is checked at runtime (not startup)
- JSON format is validated when loaded

### Timeout Validation

- Must be greater than 0
- Converted to `time.Duration` internally

## Runtime Configuration Changes

### Whitelist Updates

- **Dynamic Loading**: Whitelist is reloaded on each request
- **No Restart Required**: Changes take effect immediately
- **Error Handling**: Invalid files are logged but don't crash the server

### Other Configuration

- **Static**: All other configuration is loaded once at startup
- **Restart Required**: Changes require application restart
- **Validation**: Invalid configuration prevents startup

## Configuration Precedence

1. **Environment Variables**: Highest priority
2. **Default Values**: Used when environment variables are not set
3. **No Configuration Files**: Only environment variables and defaults are used

## Best Practices

### Security

- Store sensitive configuration in environment variables
- Use absolute paths for production whitelist files
- Set appropriate file permissions on whitelist files
- Don't commit production configuration to version control

### Performance

- Use reasonable timeout values (30-60 seconds)
- Place whitelist files on fast storage
- Monitor whitelist file size (affects memory usage)

### Monitoring

- Log configuration values at startup (excluding sensitive data)
- Monitor whitelist file changes
- Set up alerts for configuration validation failures

### Development

- Use different configuration for different environments
- Document environment-specific settings
- Test configuration changes in staging first