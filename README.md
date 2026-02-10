# Geo

A self-hosted GeoIP microservice that provides geolocation information for IP addresses with high availability, caching, and multi-provider support.

## Overview

Geo solves common challenges with GeoIP services:

- **Local databases** (MaxMind, DbIP) require regular updates and maintenance
- **Hosted SaaS services** are expensive (per-lookup charges), have SLA requirements, and create vendor lock-in

Geo combines automatic caching, failover support, and multiple provider support into a single service optimized for Kubernetes deployments.

## Features

- **Multiple Providers**: Switch between MaxMind, DbIP, IPStack, and Globio
- **Provider Cascading**: Try multiple providers in sequence with configurable fallback
- **Automatic Database Downloads**: Downloads and caches databases at startup and on schedule
- **Local Caching**: LRU ARC cache to reduce redundant lookups
- **Periodic Refresh**: Background task refreshes databases on configurable schedule
- **ETag-based Updates**: Only downloads databases when content has changed
- **Kubernetes-Ready**: Includes deployment manifests, config maps, and liveness probes
- **Structured Logging**: JSON/text logging with request tracing
- **Sentry Integration**: Optional error tracking via Sentry DSN

---

# User Guide

## Installation

### Docker

```bash
docker pull cloud66/geo:latest
docker run -p 9912:9912 -v /path/to/geo.yml:/app/geo.yml cloud66/geo:latest
```

### Build from Source

Requires Go 1.25+

```bash
git clone https://github.com/cloud66-oss/geo.git
cd geo
go mod download
CGO_ENABLED=0 go build -o geo
```

## Quick Start

```bash
# Run with default configuration
./geo serve

# Run with custom config file
./geo serve --config /path/to/geo.yml

# Run with command-line options
./geo serve --binding 0.0.0.0 --port 9912 --default maxmind
```

The service starts on port 9912 by default.

## Configuration

Geo uses configuration from multiple sources (in order of precedence):

1. Command-line flags
2. Environment variables (prefixed with `GEO_`)
3. Config file (`geo.yml` in current directory, home directory, or `/app`)
4. Default values

### Configuration File

Create a `geo.yml` file:

```yaml
# Default provider to use for lookups
default: maxmind

providers:
  # MaxMind GeoIP2/GeoLite2 databases
  maxmind:
    enabled: true
    # Direct download from MaxMind (recommended)
    # Sign up at https://www.maxmind.com/en/geolite2/signup
    account_id: ""        # MaxMind account ID
    license_key: ""       # MaxMind license key
    editions:
      city: GeoLite2-City       # or GeoIP2-City for paid databases
      asn: GeoLite2-ASN
      anonymous: ""              # e.g. GeoIP2-Anonymous-IP (paid only)
    db:
      city: dbs/geolite2-city.mmdb
      asn: dbs/geolite2-asn.mmdb
      anonymous: ""              # e.g. dbs/geoip2-anonymous.mmdb
    download:
      enabled: true
      # Fallback URLs used when license_key is not set
      # e.g. https://your-host-or-s3/file.mmdb
      city: ""
      asn: ""

  # DbIP databases
  dbip:
    enabled: true
    db:
      city: dbs/dbip-city-lite.mmdb
      asn: dbs/dbip-asn-lite.mmdb
    download:
      enabled: true
      # e.g. https://your-host-or-s3/file.mmdb
      city: ""
      asn: ""

  # IPStack API-based provider
  ipstack:
    enabled: true
    apikey: ""  # Set via environment variable

  # Globio databases (country and ASN)
  globio:
    enabled: false
    db:
      country: dbs/globio-country.mmdb
      asn: dbs/globio-asn.mmdb
    download:
      enabled: false
      country: ""
      asn: ""

  # Cascade provider (multi-provider failover)
  cascade:
    enabled: false
    providers:
      - maxmind
      - ipstack
    stopOnError: false

# Cache configuration
cache:
  enabled: true
  size: 128

# Database refresh interval
refresh: 24h

# Logging
log:
  level: info
  format: json

# Sentry error tracking (optional)
sentry:
  dsn: ""  # e.g. https://key@o123.ingest.sentry.io/456
```

### Environment Variables

All configuration options can be set via environment variables with the `GEO_` prefix:

```bash
GEO_DEFAULT=maxmind
GEO_API_BINDING=0.0.0.0
GEO_API_PORT=9912
GEO_PROVIDERS_IPSTACK_APIKEY=your-api-key
GEO_PROVIDERS_MAXMIND_ENABLED=true
GEO_PROVIDERS_MAXMIND_ACCOUNT_ID=your-account-id
GEO_PROVIDERS_MAXMIND_LICENSE_KEY=your-license-key
GEO_PROVIDERS_MAXMIND_EDITIONS_CITY=GeoLite2-City
GEO_PROVIDERS_MAXMIND_EDITIONS_ASN=GeoLite2-ASN
GEO_PROVIDERS_MAXMIND_EDITIONS_ANONYMOUS=GeoIP2-Anonymous-IP
GEO_PROVIDERS_DBIP_ENABLED=true
GEO_CACHE_ENABLED=true
GEO_CACHE_SIZE=128
GEO_REFRESH=24h
GEO_LOG_LEVEL=info
GEO_LOG_FORMAT=json
GEO_SENTRY_DSN=https://key@o123.ingest.sentry.io/456
```

## API Reference

### Health Check

```
GET /_ping
```

Returns `pong` with status 200.

### IP Lookup

```
GET /v1/ip/:address?provider=<provider_name>
```

**Parameters:**

| Parameter  | Type  | Required | Description                                                 |
|------------|-------|----------|-------------------------------------------------------------|
| `address`  | path  | Yes      | IPv4 or IPv6 address to lookup                              |
| `provider` | query | No       | Provider override (maxmind, dbip, ipstack, globio, cascade) |

**Example Request:**

```bash
curl http://localhost:9912/v1/ip/8.8.8.8
```

**Example Response:**

```json
{
  "address": "8.8.8.8",
  "source": "maxmind",
  "is_fallback": false,
  "has_city": true,
  "has_asn": true,
  "has_anonymous_ip": true,
  "city": {
    "geoname_id": 5375480,
    "names": {
      "en": "Mountain View"
    }
  },
  "continent": {
    "code": "NA",
    "geoname_id": 6255149,
    "names": {
      "en": "North America"
    }
  },
  "country": {
    "geoname_id": 6252001,
    "iso_code": "US",
    "is_in_european_union": false,
    "names": {
      "en": "United States"
    }
  },
  "location": {
    "latitude": 37.386,
    "longitude": -122.0838,
    "accuracy_radius": 1000,
    "metro_code": 807,
    "time_zone": "America/Los_Angeles"
  },
  "postal": {
    "code": "94035"
  },
  "subdivisions": [
    {
      "geoname_id": 5332921,
      "iso_code": "CA",
      "names": {
        "en": "California"
      }
    }
  ],
  "asn": {
    "autonomous_system_number": 15169,
    "autonomous_system_organization": "GOOGLE"
  },
  "anonymous_ip": {
    "is_anonymous": false,
    "is_anonymous_vpn": false,
    "is_hosting_provider": false,
    "is_public_proxy": false,
    "is_tor_exit_node": false
  }
}
```

**Error Responses:**

| Status | Description                            |
|--------|----------------------------------------|
| 400    | Invalid IP address or unknown provider |
| 500    | Lookup failure                         |

### Using Different Providers

```bash
# Use MaxMind (default)
curl http://localhost:9912/v1/ip/1.1.1.1

# Use IPStack
curl http://localhost:9912/v1/ip/1.1.1.1?provider=ipstack

# Use DbIP
curl http://localhost:9912/v1/ip/1.1.1.1?provider=dbip

# Use Cascade (failover)
curl http://localhost:9912/v1/ip/1.1.1.1?provider=cascade
```

## Providers

### MaxMind

Uses local MaxMind GeoLite2 or GeoIP2 databases for city, ASN, and anonymous IP detection.

**Data provided:** City, Country, Continent, Location, Postal, Subdivisions, ASN, Anonymous IP detection

**Download modes:**

- **Direct download (recommended):** Set `account_id` and `license_key` to download databases directly from MaxMind's API. Databases are served as `.tar.gz` archives and automatically extracted. Uses ETag-based caching to avoid redundant downloads. Free GeoLite2 accounts can be created at [maxmind.com](https://www.maxmind.com/en/geolite2/signup).
- **URL download (fallback):** When no `license_key` is configured, databases are downloaded from the URLs specified in `download.city` and `download.asn`. This is the legacy mode for using a mirror or pre-hosted database files.

**Edition IDs:** Configure which MaxMind database editions to download via `editions.city`, `editions.asn`, and `editions.anonymous`. Common values:

| Edition ID | Database | License |
|---|---|---|
| `GeoLite2-City` | City (default) | Free |
| `GeoLite2-ASN` | ASN (default) | Free |
| `GeoIP2-City` | City | Paid |
| `GeoIP2-Anonymous-IP` | Anonymous IP | Paid |

### DbIP

Uses local DbIP databases. A lightweight alternative to MaxMind.

**Data provided:** City, Country, Continent, Location

### IPStack

API-based provider using the IPStack service. Requires an API key.

**Data provided:** City, Country, Continent, Location, ASN, ISP

### Globio

Uses local Globio databases for country and ASN lookups.

**Data provided:** Country, ASN

### Cascade

Meta-provider that tries multiple providers in sequence. Useful for failover scenarios.

```yaml
providers:
  cascade:
    enabled: true
    providers:
      - maxmind
      - ipstack
    stopOnError: false  # Continue to next provider on error
```

## Deployment

### Docker

```bash
docker run -d \
  -p 9912:9912 \
  -e GEO_PROVIDERS_IPSTACK_APIKEY=your-key \
  -v /path/to/geo.yml:/app/geo.yml \
  cloud66/geo:latest
```

### Kubernetes

Create a secret for the IPStack API key:

```bash
kubectl create secret generic geo-ipstack-api-key \
  --namespace=your-namespace \
  --from-literal=api-key=your-api-key
```

Apply the deployment:

```bash
kubectl apply -f deployment/deployment.yml
```

The included Kubernetes manifest provides:
- Deployment with resource limits
- Service on port 9912
- Liveness probe on `/_ping`
- ConfigMap for configuration
- Secret reference for API keys

---

# Developer Guide

## Architecture

```
┌─────────────────────────────────────────────┐
│         HTTP Server (Echo Framework)        │
│  /_ping (healthcheck)                       │
│  /v1/ip/:address (lookup endpoint)          │
└────────────┬────────────────────────────────┘
             │
    ┌────────▼─────────┐
    │  Request Router  │
    │  with Caching    │
    └────────┬─────────┘
             │
    ┌────────▼──────────────────────────────────┐
    │   Provider Layer (IPProvider Interface)   │
    │  - MaxMind (city, ASN, anonymous)         │
    │  - DbIP (city data)                       │
    │  - IPStack (API-based)                    │
    │  - Globio (country + ASN)                  │
    │  - Cascade (multi-provider failover)      │
    └────────┬──────────────────────────────────┘
             │
    ┌────────▼──────────────────────────────────┐
    │     Cache Layer (LRU ARC Cache)           │
    │     Stores IP lookups by provider         │
    └───────────────────────────────────────────┘
```

## Project Structure

```
geo/
├── cmd/                    # CLI command handling
│   ├── root.go            # Root command setup
│   ├── serve.go           # Server command implementation
│   └── serve_test.go      # Server tests
├── provider/              # IP data providers
│   ├── ip_provider.go     # Provider interface
│   ├── max_mind_provider.go
│   ├── db_ip.go
│   ├── ipstack_provider.go
│   ├── globio_provider.go
│   ├── cascade_ip_provider.go
│   └── cascade_ip_provider_test.go
├── cache/                 # Caching layer
│   ├── cache_provider.go  # Cache interface
│   └── local_cache.go     # LRU cache implementation
├── utils/                 # Utilities and shared types
│   ├── ip_info.go         # Data structures
│   ├── container.go       # IoC container
│   ├── errors.go
│   ├── http.go            # Download helpers (URL, MaxMind API, tar.gz extraction)
│   ├── http_test.go       # Download and extraction tests
│   ├── file.go
│   └── echo_zero_logger.go
├── deployment/            # Kubernetes manifests
│   ├── deployment.yml
│   └── deployment-staging.yml
├── geo.yml                # Default configuration
├── main.go                # Entry point
├── Dockerfile
├── cloudbuild.json          # Production Cloud Build config
└── cloudbuild-staging.json  # Staging Cloud Build config
```

## Building

```bash
# Standard build
go build -o geo

# Build with version information
CGO_ENABLED=0 go build \
  -ldflags="-X 'github.com/cloud66-oss/geo/utils.Version=1.0.0'" \
  -o geo

# Build Docker image
docker build --build-arg SHORT_SHA=$(git rev-parse --short HEAD) -t geo:latest .
```

## Testing

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run specific package tests
go test -v ./provider/...
go test -v ./cmd/...
```

## Adding a New Provider

1. Create a new file in `provider/` (e.g., `my_provider.go`)

2. Implement the `IPProvider` interface:

```go
package provider

import (
    "context"
    "github.com/cloud66-oss/geo/utils"
)

type MyProvider struct {
    // Provider-specific fields
}

func NewMyProvider() *MyProvider {
    return &MyProvider{}
}

func (p *MyProvider) Name() string {
    return "myprovider"
}

func (p *MyProvider) Start(ctx context.Context) error {
    // Initialize the provider (download databases, connect to APIs, etc.)
    return nil
}

func (p *MyProvider) Lookup(ctx context.Context, address string) (*utils.IPInfo, error) {
    // Perform the IP lookup
    return &utils.IPInfo{
        Address: address,
        Source:  p.Name(),
        // ... populate other fields
    }, nil
}

func (p *MyProvider) Refresh(ctx context.Context) error {
    // Refresh databases or connections
    return nil
}

func (p *MyProvider) Shutdown(ctx context.Context) error {
    // Clean up resources
    return nil
}
```

3. Register the provider in `cmd/serve.go`:

```go
// In the initProviders function
if viper.GetBool("providers.myprovider.enabled") {
    myProvider := provider.NewMyProvider()
    if err := myProvider.Start(ctx); err != nil {
        return err
    }
    container.SetProvider("myprovider", myProvider)
}
```

4. Add configuration options to `geo.yml`:

```yaml
providers:
  myprovider:
    enabled: true
    # Provider-specific options
```

## Key Interfaces

### IPProvider

```go
type IPProvider interface {
    Name() string
    Start(ctx context.Context) error
    Lookup(ctx context.Context, address string) (*utils.IPInfo, error)
    Refresh(ctx context.Context) error
    Shutdown(ctx context.Context) error
}
```

### CacheProvider

```go
type CacheProvider interface {
    Get(key string) (*utils.IPInfo, bool)
    Set(key string, value *utils.IPInfo)
}
```

## Request Flow

1. HTTP request arrives at `/v1/ip/:address`
2. Extract provider name from query params (or use default)
3. If cache enabled, check LRU cache (key: `provider--address`)
4. If cache miss, get provider from container
5. Call `provider.Lookup()` with context and address
6. If cache enabled, store result in cache
7. Return JSON response

## Dependencies

| Package                  | Purpose                  |
|--------------------------|--------------------------|
| `labstack/echo`          | HTTP server framework    |
| `spf13/viper`            | Configuration management |
| `spf13/cobra`            | CLI command parsing      |
| `rs/zerolog`             | Structured logging       |
| `oschwald/geoip2-golang` | MaxMind database reader  |
| `qioalice/ipstack`       | IPStack API client       |
| `hashicorp/golang-lru`   | ARC LRU cache            |
| `getsentry/sentry-go`    | Error tracking           |
| `jinzhu/copier`          | Struct field copying     |

## Environment Setup

```bash
# Clone repository
git clone https://github.com/cloud66-oss/geo.git
cd geo

# Install dependencies
go mod download

# Run in development mode
go run main.go serve --level debug --log-format text

# Run tests
go test ./...
```

## Logging

Geo uses zerolog for structured logging. Configure via:

```yaml
log:
  level: debug  # debug, info, warn, error
  format: text  # json or text
```

Request logs include:
- Request ID (X-Request-ID header)
- Remote IP
- Method and path
- Status code
- Latency

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/my-feature`)
3. Make your changes
4. Run tests (`go test ./...`)
5. Commit your changes (`git commit -am 'Add my feature'`)
6. Push to the branch (`git push origin feature/my-feature`)
7. Create a Pull Request

## License

Apache License 2.0 - see [LICENSE](LICENSE) for details.
