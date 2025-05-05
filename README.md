# Cartographoor

A service that discovers and maps active Ethereum networks in the ethpandaops ecosystem.

## Overview

Cartographoor is a Go application that periodically scans and discovers active Ethereum networks maintained by the ethpandaops team. It aggregates network information and uploads it to S3 as a structured JSON file, making it easier to maintain an up-to-date view of available networks.

The name "Cartographoor" combines "cartography" (the science of map-making) with the "oor" suffix common in ethpandaops projects, reflecting its purpose of mapping the Ethereum network landscape.

### Architecture

Cartographoor follows a modular architecture with three main components:

1. **Discovery Service**: Coordinates the discovery process and aggregates results
2. **Discovery Providers**: Pluggable components that discover networks from different sources (currently GitHub)
3. **Storage Providers**: Components that store the discovery results (currently S3)

This design allows for easy extension with new discovery sources or storage backends without modifying the core functionality.

## Features

- Periodic discovery of Ethereum networks
- Multiple discovery sources (starting with GitHub repositories)
- Configurable discovery intervals and sources
- Uploads discovered networks to S3 as a `networks.json` file

## How It Works

Cartographoor uses a configuration file to determine:
- Which repositories to scan
- How often to perform discovery
- Where to look for network configurations
- S3 bucket details for storing the results

The service identifies networks by checking for directories within the `network-configs/` path of specified repositories.

## Example

The service scans repositories like `ethpandaops/dencun-devnets`, which contains networks such as:
- devnet-4, devnet-5, ..., devnet-12
- gsf-1, gsf-2
- msf-1
- sepolia-sf1

## Installation

### From Source

1. Clone the repository:
   ```bash
   git clone https://github.com/ethpandaops/cartographoor.git
   cd cartographoor
   ```

2. Build the binary:
   ```bash
   make build
   ```

3. Copy and modify the example configuration:
   ```bash
   cp config.example.yaml config.yaml
   ```

## Configuration

Cartographoor is configured via a YAML file that specifies discovery sources, intervals, and output settings. See `config.example.yaml` for a complete example with comments.

Key configuration sections:

```yaml
# Discovery configuration
discovery:
  interval: 1h
  github:
    repositories:
      # Simple configuration
      - name: ethpandaops/dencun-devnets
      
      # With name prefix (adds "dencun-" to each network name)
      - name: ethpandaops/dencun-devnets
        namePrefix: dencun-

# S3 storage configuration
storage:
  bucketName: ethpandaops-networks
  key: networks.json
  region: us-east-1
```

### Environment Variable Substitution

The configuration file supports environment variable substitution using the `${VAR}` syntax. This allows you to keep sensitive information like API tokens and credentials outside of your configuration files.

For example:

```yaml
storage:
  bucketName: ${S3_BUCKET_NAME}
  region: ${AWS_REGION}
  accessKey: ${AWS_ACCESS_KEY_ID}
  secretKey: ${AWS_SECRET_ACCESS_KEY}
```

When the application starts, it will replace `${VAR}` with the corresponding environment variable value. If the environment variable is not set, the placeholder will remain unchanged.

## Usage

```bash
# Run with default configuration (continuous mode)
cartographoor run

# Run with custom configuration
cartographoor run --config=/path/to/config.yaml

# Run once and exit
cartographoor run --once

# Run in debug mode 
cartographoor run --logging.level=debug
```

### Docker

Cartographoor is available as a Docker image from GitHub Container Registry:

```bash
# Pull the latest image
docker pull ghcr.io/ethpandaops/cartographoor:latest

# Run with a custom config
docker run -v /path/to/config.yaml:/app/config/config.yaml ghcr.io/ethpandaops/cartographoor:latest

# Run once and exit
docker run -v /path/to/config.yaml:/app/config/config.yaml ghcr.io/ethpandaops/cartographoor:latest run --once
```

You can also use Docker Compose:

```yaml
version: '3.8'

services:
  cartographoor:
    image: ghcr.io/ethpandaops/cartographoor:latest
    restart: unless-stopped
    volumes:
      - ./config.yaml:/app/config/config.yaml
    environment:
      - CARTOGRAPHOOR_LOGGING_LEVEL=info
```

### Running with Local Minio (S3 Alternative)

For local development or testing, you can use the included Docker Compose setup with Minio, an S3-compatible storage service:

```bash
# Start the full stack (Cartographoor + Minio)
docker-compose up -d

# View logs
docker-compose logs -f

# Run once and exit
docker-compose run --rm cartographoor run --once --config=/app/config/config.yaml
```

This setup includes:
- Minio S3-compatible storage (accessible at http://localhost:9000)
- Minio Console UI (accessible at http://localhost:9001)
- Automatic bucket creation
- Configuration for Cartographoor to use the local Minio

The Minio console is available at http://localhost:9001 with username `minioadmin` and password `minioadmin`. 
You can view the generated `networks.json` file by navigating to the `ethpandaops-networks` bucket.

## Development

### Requirements

- Go 1.24 or higher
- Make (optional, for using the Makefile)

### Project Structure

```
├── cmd/                    # Application entry points
│   └── cartographoor/      # Main CLI application
│       ├── cmd/            # Command definitions using Cobra
│       └── main.go         # Application entry point
├── pkg/                    # Reusable packages
│   ├── discovery/          # Network discovery service
│   │   ├── service.go      # Discovery service implementation
│   │   └── types.go        # Common types used by discovery
│   ├── providers/          # Network discovery providers
│   │   └── github/         # GitHub repository provider
│   ├── storage/            # Storage providers for saving discovered networks
│   │   └── s3/             # AWS S3 storage provider
│   └── utils/              # Utility functions
│       └── envsubst.go     # Environment variable substitution
├── Dockerfile              # Container definition
├── Makefile                # Build automation
└── config.example.yaml     # Example configuration
```

### Common Commands

```bash
# Build the binary
make build

# Run the application
make run

# Run tests
make test

# Clean build artifacts
make clean
```

### Extending Cartographoor

Cartographoor is designed to be extensible. You can add new discovery providers or storage backends by implementing the appropriate interfaces.

#### Adding a New Discovery Provider

To add a new discovery provider, implement the `discovery.Provider` interface:

```go
type Provider interface {
	// Name returns the name of the provider.
	Name() string

	// Discover discovers networks and returns them as a map with network names as keys.
	Discover(ctx context.Context, config Config) (map[string]Network, error)
}
```

Then register your provider in the `runService` function in `cmd/cartographoor/cmd/run.go`.

#### Adding a New Storage Provider

To add a new storage backend, create a new package in the `pkg/storage` directory that provides similar functionality to the S3 provider. The key method to implement is:

```go
Upload(ctx context.Context, result discovery.Result) error
```

Then update the `runService` function to use your new storage provider.

## Output Format

The service produces a JSON file with the following structure:

```json
{
  "networks": {
    "devnet-10": {
      "name": "devnet-10",
      "repository": "ethpandaops/dencun-devnets",
      "path": "network-configs/devnet-10",
      "url": "https://github.com/ethpandaops/dencun-devnets/tree/main/network-configs/devnet-10",
      "status": "active",
      "lastUpdated": "2023-05-04T15:30:00Z"
    },
    "dencun-devnet-4": {
      "name": "devnet-4",
      "repository": "ethpandaops/dencun-devnets",
      "path": "network-configs/devnet-4",
      "url": "https://github.com/ethpandaops/dencun-devnets/tree/main/network-configs/devnet-4",
      "status": "active",
      "lastUpdated": "2023-05-04T15:30:00Z"
    },
    ...
  },
  "lastUpdate": "2023-05-04T15:30:00Z",
  "duration": 1.25,
  "providers": ["github"]
}
```

## License

Apache 2.0