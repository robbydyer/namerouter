# Namerouter
Namerouter is a simple reverse proxy.

## Features
- Differentiation between "internal" and "external" hostnames for the same destination
- Supports SSL and auto-creates certs (using Letsencrypt) for external hosts
- Rate limiting, with ability to tweak config for internal and external access

## Configuration
Config is provided by a YAML config file. 

```yaml
# Email, required for Letsencrypt cert generation
email: me@place.com

# Enable this to auto-create SSL certs for all defined external hostnames
doSSL: true

# Rate limiting
rateLimits:
  # Limits for requests from internal domains
  internal:
    # Requests per second
    rate: 1000
    burst: 1000

  # Limits for requests coming from external domains
  external:
    rate: 10
    burst: 20

# Define the routes.
# Each route has a `destination` and one or both of
# `external` or `internal` hosts. If `doSSL` is true,
# exrternal hosts will have SSL certs created for them.
routes:
  - destination: "http://10.0.0.1:8080"
    external:
      - "app1.example.com"
    internal:
      - "app1.local"
```

### Special Route Options
There are some less often used options for route config:
- `always404` -> Set to true to have requests to these hosts always return a 404
  Example:
  ```yaml
  routes:
    - always404: true
      external:
        - "ignore.example.com"
  ```
