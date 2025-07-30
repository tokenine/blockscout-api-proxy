# Requirements Document

## Introduction

This feature implements a Go-based REST API proxy server that forwards requests to https://exp.co2e.cc/api/v2 and serves responses on HTTP port 80. The proxy includes special filtering functionality for the `/api/v2/tokens` endpoint, where responses are filtered against a whitelist of token addresses stored in a JSON file.

## Requirements

### Requirement 1

**User Story:** As a client application, I want to make HTTP requests to the proxy server on port 80, so that I can access the backend API through a local proxy.

#### Acceptance Criteria

1. WHEN a client makes an HTTP request to the proxy server THEN the server SHALL listen on port 80
2. WHEN a request is received THEN the proxy SHALL forward the request to the configured backend host
3. WHEN the backend host is not configured THEN the proxy SHALL use environment variable for the target host
4. WHEN the backend responds THEN the proxy SHALL return the response to the client

### Requirement 2

**User Story:** As a system administrator, I want to configure the backend API host through environment variables, so that I can easily change the target without code modifications.

#### Acceptance Criteria

1. WHEN the application starts THEN it SHALL read the backend host from an environment variable
2. WHEN the environment variable is not set THEN the application SHALL use https://exp.co2e.cc as the default host
3. WHEN making backend requests THEN the proxy SHALL append /api/v2 to the configured host URL

### Requirement 3

**User Story:** As a security administrator, I want to filter token responses based on a whitelist, so that only approved tokens are returned to clients.

#### Acceptance Criteria

1. WHEN a request is made to `/api/v2/tokens` THEN the proxy SHALL fetch data from the backend API
2. WHEN the backend returns token data THEN the proxy SHALL load the whitelist from a JSON file
3. WHEN filtering tokens THEN the proxy SHALL only include tokens whose address field matches entries in the whitelist
4. WHEN no tokens match the whitelist THEN the proxy SHALL return an empty items array
5. WHEN the whitelist file is missing or invalid THEN the proxy SHALL log an error and return all tokens

### Requirement 4

**User Story:** As a system administrator, I want to maintain a whitelist of approved token addresses, so that I can control which tokens are exposed through the API.

#### Acceptance Criteria

1. WHEN the application starts THEN it SHALL look for a whitelist.json file in the application directory
2. WHEN the whitelist file exists THEN it SHALL contain an array of token addresses
3. WHEN updating the whitelist THEN the changes SHALL take effect on the next request without restarting the server
4. WHEN the whitelist file format is invalid THEN the application SHALL log an error and continue operation

### Requirement 5

**User Story:** As a developer, I want the proxy to handle all other API endpoints transparently, so that the proxy doesn't interfere with normal API operations.

#### Acceptance Criteria

1. WHEN a request is made to any endpoint other than `/api/v2/tokens` THEN the proxy SHALL forward the request unchanged
2. WHEN the backend responds to non-tokens requests THEN the proxy SHALL return the response unchanged
3. WHEN the backend returns an error THEN the proxy SHALL forward the error response to the client
4. WHEN the backend is unreachable THEN the proxy SHALL return an appropriate error response

### Requirement 6

**User Story:** As a system administrator, I want proper error handling and logging, so that I can monitor and troubleshoot the proxy service.

#### Acceptance Criteria

1. WHEN an error occurs THEN the application SHALL log the error with appropriate detail
2. WHEN the backend is unreachable THEN the proxy SHALL return HTTP 502 Bad Gateway
3. WHEN the whitelist file cannot be read THEN the proxy SHALL log a warning and continue
4. WHEN invalid JSON is received from the backend THEN the proxy SHALL return HTTP 502 Bad Gateway