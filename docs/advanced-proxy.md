# Advanced HTTP/HTTPS Proxy Configuration

This document describes the advanced proxy configuration features available for HTTP and HTTPS routes in Warptail.

## Overview

Warptail now supports nginx-style routing and proxy configuration for HTTP/HTTPS routes, allowing you to:

- Route different paths to different backend servers
- Modify request and response headers  
- Configure proxy timeouts and retry behavior
- Rewrite request paths
- Strip path prefixes

## Configuration Options

### Proxy Settings

```yaml
proxy_settings:
  timeout: 60                    # Request timeout in seconds (default: 30)
  retry_attempts: 3              # Number of retry attempts (default: 3)
  buffer_requests: true          # Buffer requests before forwarding (default: false)
  preserve_host: false           # Preserve original Host header (default: false)
  follow_redirects: true         # Follow HTTP redirects (default: false)
```

### Custom Headers

You can modify request and response headers in three ways:

```yaml
custom_headers:
  add:                           # Add headers only if they don't exist
    X-Forwarded-Proto: "https"
    X-Real-IP: "$remote_addr"
  set:                           # Set headers (overwrite existing)
    X-Custom-Header: "warptail-proxy"
    Cache-Control: "no-cache"
  remove:                        # Remove headers
    - "Server"
    - "X-Powered-By"
```

### Path-based Routing Rules

Configure nginx-style location blocks to route different paths to different backends:

```yaml
rules:
  - path: "/api/"                # Path prefix to match (required)
    target_host: "api-server"    # Backend server (optional, defaults to main machine)
    target_port: 8080           # Backend port (optional, defaults to main port)
    strip_path: true            # Remove matched path from request (optional)
    rewrite: "/v1/"             # Rewrite path prefix (optional, only used with strip_path)
```

## Examples

### Basic API Gateway

Route API calls to a different backend:

```yaml
type: "https"
domain: "myapp.com"
machine:
  address: "web-server"
  port: 3000
proxy_settings:
  rules:
    # Simple path stripping - removes /api/ prefix
    - path: "/api/"
      target_host: "api-server"
      target_port: 8080
      strip_path: true
    
    # Static files - no path modification needed  
    - path: "/static/"
      target_host: "cdn-server"
      target_port: 9000
```

This configuration will:
- Route `https://myapp.com/` to `web-server:3000`  
- Route `https://myapp.com/api/users` to `api-server:8080/users` (path stripped)
- Route `https://myapp.com/static/logo.png` to `cdn-server:9000/static/logo.png` (path preserved)

### Microservices Routing

Route to multiple microservices:

```yaml
type: "https"
domain: "services.com"
machine:
  address: "frontend"
  port: 3000
proxy_settings:
  rules:
    - path: "/auth/"
      target_host: "auth-service"
      target_port: 8001
      strip_path: true
    - path: "/user/"
      target_host: "user-service" 
      target_port: 8002
      strip_path: true
    - path: "/static/"
      target_host: "cdn"
      target_port: 9000
```

### Security Headers

Add security headers to all responses:

```yaml
proxy_settings:
  custom_headers:
    set:
      X-Content-Type-Options: "nosniff"
      X-Frame-Options: "DENY"  
      X-XSS-Protection: "1; mode=block"
      Strict-Transport-Security: "max-age=31536000"
    remove:
      - "Server"
      - "X-Powered-By"
```

### WebSocket Proxying

Route WebSocket connections:

```yaml
rules:
  - path: "/ws/"
    target_host: "websocket-server"
    target_port: 3001
    strip_path: true
proxy_settings:
  preserve_host: true
  buffer_requests: false  # Important for WebSockets
```

## Path Matching

Path matching is done using prefix matching. The longest matching path wins.

- `/api/` matches `/api/users`, `/api/posts`, etc.
- `/api/v1/` matches `/api/v1/users` but not `/api/v2/users`
- Exact paths like `/health` only match that specific path

## Path Rewriting

The `rewrite` and `strip_path` options work together (both are optional):

1. If `strip_path: true`, the matched path prefix is removed
2. If `rewrite` is specified (and `strip_path` is true), it's prepended to the remaining path
3. If only `strip_path: true` (no rewrite), the path is just stripped

Examples:
- **Strip only**: `/api/v1/users` with `path: "/api/", strip_path: true` → `/v1/users`
- **Strip + Rewrite**: `/api/v1/users` with `path: "/api/", strip_path: true, rewrite: "/backend/"` → `/backend/v1/users`
- **No changes**: `/api/v1/users` with `path: "/api/"` → `/api/v1/users` (forwarded as-is)

## Header Variables

Some headers support variable substitution:
- `$remote_addr` - Client IP address
- `$host` - Original Host header
- `$scheme` - http or https

## Best Practices

1. **Order Rules by Specificity**: More specific paths should come first
2. **Use `preserve_host: true`** for applications that check the Host header
3. **Disable `buffer_requests`** for streaming or WebSocket applications
4. **Set appropriate timeouts** for long-running requests
5. **Remove sensitive headers** like `Server` and `X-Powered-By`
6. **Add security headers** for public-facing applications

## Limitations

- Path matching is prefix-based only (no regex support yet)
- Variable substitution is limited to a few predefined variables
- Rules are evaluated in order, first match wins
- Backend servers must be accessible from the Warptail instance