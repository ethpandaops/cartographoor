# Network Status

A service that discovers active networks in the ethpandaops ecosystem.

## Overview

Network Status is a Go application that periodically scans and discovers active Ethereum networks maintained by the ethpandaops team. It aggregates network information and uploads it to S3 as a structured JSON file, making it easier to maintain an up-to-date view of available networks.

## Features

- Periodic discovery of Ethereum networks
- Multiple discovery sources (starting with GitHub repositories)
- Configurable discovery intervals and sources
- Uploads discovered networks to S3 as a `networks.json` file

## How It Works

Network Status uses a configuration file to determine:
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
   git clone https://github.com/ethpandaops/network-status.git
   cd network-status
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

Network Status is configured via a YAML file that specifies discovery sources, intervals, and output settings. See `config.example.yaml` for a complete example with comments.

Key configuration sections:

```yaml
# Discovery configuration
discovery:
  interval: 1h
  github:
    repositories:
      - ethpandaops/dencun-devnets

# S3 storage configuration
storage:
  bucketName: ethpandaops-networks
  key: networks.json
  region: us-east-1
```

## Usage

```bash
# Run with default configuration (continuous mode)
network-status run

# Run with custom configuration
network-status run --config=/path/to/config.yaml

# Run once and exit
network-status run --once

# Run in debug mode 
network-status run --logging.level=debug
```

### Docker

Network Status is available as a Docker image from GitHub Container Registry:

```bash
# Pull the latest image
docker pull ghcr.io/ethpandaops/network-status:latest

# Run with a custom config
docker run -v /path/to/config.yaml:/app/config/config.yaml ghcr.io/ethpandaops/network-status:latest

# Run once and exit
docker run -v /path/to/config.yaml:/app/config/config.yaml ghcr.io/ethpandaops/network-status:latest run --once
```

You can also use Docker Compose:

```yaml
version: '3.8'

services:
  network-status:
    image: ghcr.io/ethpandaops/network-status:latest
    restart: unless-stopped
    volumes:
      - ./config.yaml:/app/config/config.yaml
    environment:
      - NETWORK_STATUS_LOGGING_LEVEL=info
```

## Development

### Requirements

- Go 1.24 or higher
- Make (optional, for using the Makefile)

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

## Output Format

The service produces a JSON file with the following structure:

```json
{
  "networks": [
    {
      "name": "devnet-10",
      "repository": "ethpandaops/dencun-devnets",
      "path": "network-configs/devnet-10",
      "url": "https://github.com/ethpandaops/dencun-devnets/tree/main/network-configs/devnet-10",
      "status": "active",
      "lastUpdated": "2023-05-04T15:30:00Z"
    },
    ...
  ],
  "lastUpdate": "2023-05-04T15:30:00Z",
  "duration": 1.25,
  "providers": ["github"]
}
```

## License

Apache 2.0