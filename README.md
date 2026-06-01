# Cartographoor

A service that discovers and maps active Ethereum networks in the ethpandaops ecosystem.

[![Go Reference](https://pkg.go.dev/badge/github.com/ethpandaops/cartographoor.svg)](https://pkg.go.dev/github.com/ethpandaops/cartographoor)
[![Go Report Card](https://goreportcard.com/badge/github.com/ethpandaops/cartographoor)](https://goreportcard.com/report/github.com/ethpandaops/cartographoor)
[![License](https://img.shields.io/github/license/ethpandaops/cartographoor)](LICENSE)
[![Latest Networks](https://img.shields.io/badge/Latest%20Networks-JSON-blue)](https://ethpandaops-platform-production-cartographoor.ams3.digitaloceanspaces.com/networks.json)

## Overview

Cartographoor is a Go application that periodically scans and discovers active Ethereum networks maintained by the ethpandaops team. It aggregates network metadata and uploads it to S3 as a structured `networks.json` file, making it easy to maintain an up-to-date view of available networks.

The name "Cartographoor" combines "cartography" (the science of map-making) with the "oor" suffix common in ethpandaops projects, reflecting its purpose of mapping the Ethereum network landscape.

### Architecture

Cartographoor follows a modular, provider-based architecture:

1. **Discovery Service** — coordinates discovery, aggregates results, and identifies Ethereum clients.
2. **Discovery Providers** — pluggable sources that discover networks:
   - `github` — scans the `network-configs/` directory of configured repositories.
   - `static` — hardcoded networks (mainnet, sepolia, hoodi, …) defined in config.
3. **Storage Providers** — store the discovery results (currently AWS S3 / S3-compatible stores).

On top of the core discovery, the binary ships additional generators that consume the discovered data and produce their own artifacts (see [Subcommands](#subcommands)).

A standalone Go [client library](#client-library) (`pkg/client`) is also provided for other services that want to consume `networks.json` with caching built in.

This design allows for easy extension with new discovery sources or storage backends without modifying the core functionality.

## Features

- Periodic discovery of Ethereum networks from GitHub repositories and static configuration
- Rich network metadata: chain ID, status, fork schedules, blob schedules, genesis config, service URLs, client/tool images
- Ethereum client discovery (consensus & execution) with versions and metadata
- Configurable discovery intervals and sources
- Uploads to S3 (or any S3-compatible store, e.g. DigitalOcean Spaces, Minio) as `networks.json`
- Additional generators: Dora-based client inventory, validator ranges, and EIP-7870 reference node commands
- Embeddable client library with in-memory and Redis-backed caching

## How It Works

Cartographoor uses a configuration file to determine:
- Which repositories to scan and which static networks to include
- How often to perform discovery
- Where to look for network configurations
- S3 bucket details for storing the results

For GitHub sources, the service identifies networks by checking for directories within the `network-configs/` path of specified repositories, then enriches each network with metadata parsed from its configuration files.

### Requirements

- **GitHub Token**: A GitHub personal access token is strongly recommended to prevent rate limiting when accessing GitHub repositories. Provide it in the configuration file or via the `GITHUB_TOKEN` environment variable.

## Subcommands

The `cartographoor` binary exposes several subcommands:

| Command | Description |
| --- | --- |
| `run` | Core discovery loop; discovers networks and uploads `networks.json`. Supports `--once`. |
| `inventory` | Generates a network inventory from Dora APIs (with optional DNS validation). |
| `validator-ranges` | Downloads `networks.json` and generates validator range data from Ansible inventory files. |
| `eip7870-reference-nodes` | Generates EIP-7870 reference node startup commands from the ethereum-helm-charts and platform repositories. |

In production, each subcommand runs as its own scheduled GitHub Actions workflow (see `.github/workflows/`) against `.github/config.production.yaml`.

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

Cartographoor is configured via a YAML file that specifies discovery sources, intervals, and output settings. See `config.example.yaml` for a complete, commented example, and `.github/config.production.yaml` for the live production configuration.

Key configuration sections:

```yaml
logging:
  level: info

# Discovery configuration
discovery:
  interval: 1h

  # Static networks (e.g. mainnet, sepolia, hoodi)
  static:
    networks:
      - name: mainnet
        description: Production Ethereum network
        chainId: 1
        genesisTime: 1606824023
        configUrl: "https://raw.githubusercontent.com/eth-clients/mainnet/refs/heads/main/metadata/config.yaml"
        serviceUrls:
          beaconExplorer: https://beaconcha.in
        forks:
          consensus:
            altair: { epoch: 74240 }
          execution:
            london: { block: 12965000, timestamp: 1628166822 }

  github:
    # GitHub token is recommended to prevent rate limiting
    token: ${GITHUB_TOKEN}
    repositories:
      # Simple configuration
      - name: ethpandaops/pectra-devnets

      # With metadata (prefix, display name, description, image, links)
      - name: ethpandaops/fusaka-devnets
        namePrefix: fusaka-
        displayName: Fusaka Devnets
        description: "Fusaka upgrade devnets."
        image: https://ethpandaops.io/img/fusaka.jpg
        links:
          - title: "EIP-7594: PeerDAS"
            url: "https://eips.ethereum.org/EIPS/eip-7594"

# S3 storage configuration
storage:
  bucketName: ethpandaops-networks
  key: networks.json
  region: us-east-1
  # endpoint: https://ams3.digitaloceanspaces.com  # for S3-compatible stores
  contentType: application/json
  retryDuration: 5s
  maxRetries: 3

# Inventory generation (used by the `inventory` subcommand)
inventory:
  validation:
    enabled: true
    dnsTimeout: 3s
    maxConcurrentValidations: 100

# Validator ranges (used by the `validator-ranges` subcommand)
validatorRanges:
  additionalSources:
    fusaka-devnet-5:
      - url: https://raw.githubusercontent.com/testinprod-io/fusaka-devnets/refs/heads/main/ansible/inventories/devnet-5/inventory.ini
        name: testinprod
        rangeOffset: 51312
```

### Environment Variable Substitution

The configuration file supports environment variable substitution using the `${VAR}` syntax. This allows you to keep sensitive information like API tokens and credentials outside of your configuration files.

For example:

```yaml
discovery:
  github:
    token: ${GITHUB_TOKEN}

storage:
  bucketName: ${S3_BUCKET_NAME}
  region: ${AWS_REGION}
  accessKey: ${AWS_ACCESS_KEY_ID}
  secretKey: ${AWS_SECRET_ACCESS_KEY}
```

When the application starts, it replaces `${VAR}` with the corresponding environment variable value. If the environment variable is not set, the placeholder remains unchanged.

## Usage

```bash
# Run discovery in continuous mode
cartographoor run --config=config.yaml

# Run discovery once and exit
cartographoor run --config=config.yaml --once

# Run in debug mode
cartographoor run --config=config.yaml --logging.level=debug

# Generate the Dora-based inventory
cartographoor inventory --config=config.yaml

# Generate validator ranges
cartographoor validator-ranges --config=config.yaml

# Generate EIP-7870 reference node commands
cartographoor eip7870-reference-nodes --config=config.yaml
```

### Docker

Cartographoor is available as a Docker image from GitHub Container Registry:

```bash
# Pull the latest image
docker pull ghcr.io/ethpandaops/cartographoor:latest

# Run with a custom config
docker run -v /path/to/config.yaml:/app/config/config.yaml ghcr.io/ethpandaops/cartographoor:latest run --config=/app/config/config.yaml

# Run once and exit
docker run -v /path/to/config.yaml:/app/config/config.yaml ghcr.io/ethpandaops/cartographoor:latest run --config=/app/config/config.yaml --once
```

You can also use Docker Compose:

```yaml
services:
  cartographoor:
    image: ghcr.io/ethpandaops/cartographoor:latest
    restart: unless-stopped
    volumes:
      - ./config.yaml:/app/config/config.yaml
    environment:
      - CARTOGRAPHOOR_LOGGING_LEVEL=info
      - GITHUB_TOKEN=your_github_token_here
    command: run --config=/app/config/config.yaml
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
You can view the generated `networks.json` file by navigating to the `ethpandaops-networks` bucket, or run `./view-networks.sh`.

## Client Library

Other Go services can consume the published `networks.json` via the `pkg/client` package, which fetches and caches the data with periodic refresh. Two implementations of the `client.Provider` interface are available:

- `MemoryProvider` — fetches and caches the data in memory.
- `RedisProvider` — caches via Redis with leader election, so only the leader fetches.

```go
import "github.com/ethpandaops/cartographoor/pkg/client"

provider, err := client.NewMemoryProvider(client.Config{}, log)
if err != nil {
    // handle error
}

if err := provider.Start(ctx); err != nil {
    // handle error
}
defer provider.Stop()

networks, err := provider.GetActiveNetworks(ctx)
```

By default the client fetches from the production endpoint and refreshes every 5 minutes (configurable via `client.Config`).

## Development

### Requirements

- Go 1.26 or higher (see `go.mod` / `.tool-versions`)
- Make (optional, for using the Makefile)

### Project Structure

```
├── cmd/                          # Application entry points
│   └── cartographoor/            # Main CLI application
│       ├── cmd/                  # Cobra command definitions
│       │   ├── root.go           # Root command + subcommand wiring
│       │   ├── run.go            # Discovery `run` command
│       │   ├── inventory.go      # `inventory` command
│       │   ├── validator_ranges.go        # `validator-ranges` command
│       │   └── eip7870_reference_nodes.go # `eip7870-reference-nodes` command
│       └── main.go               # Entry point
├── pkg/                          # Reusable packages
│   ├── discovery/                # Discovery service + data model
│   │   ├── service.go            # Discovery orchestration
│   │   ├── types.go              # Core types (Network, Result, ClientInfo, …)
│   │   └── clients.go            # Ethereum client discovery/identification
│   ├── providers/                # Discovery providers
│   │   ├── github/               # GitHub repository provider
│   │   └── static/               # Static (hardcoded) network provider
│   ├── storage/                  # Storage providers
│   │   └── s3/                   # AWS S3 / S3-compatible storage provider
│   ├── inventory/                # Dora-based inventory generator
│   ├── validatorranges/          # Validator ranges generator
│   ├── eip7870referencenodes/    # EIP-7870 reference node command generator
│   ├── client/                   # Embeddable consumer client library
│   └── utils/                    # Utilities (env var substitution)
├── .github/                      # Workflows + production config
├── Dockerfile                    # Container definition
├── Makefile                      # Build automation
└── config.example.yaml           # Example configuration
```

### Common Commands

```bash
make build    # Build the binary
make run      # Build and run with config.example.yaml
make test     # Run tests
make lint     # Run golangci-lint
make clean    # Clean build artifacts
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

Then register your provider in the `runService` function in `cmd/cartographoor/cmd/run.go` (see how the `github` and `static` providers are registered).

#### Adding a New Storage Provider

To add a new storage backend, create a new package in the `pkg/storage` directory that provides similar functionality to the S3 provider. The key method to implement is:

```go
Upload(ctx context.Context, result discovery.Result) error
```

Then update the `runService` function to use your new storage provider.

## Output Format

The `run` command produces a JSON file with the following structure (fields are omitted when empty):

```json
{
  "networkMetadata": {
    "ethpandaops/fusaka-devnets": {
      "displayName": "Fusaka Devnets",
      "description": "Fusaka upgrade devnets.",
      "image": "https://ethpandaops.io/img/fusaka.jpg",
      "links": [{ "title": "EIP-7594: PeerDAS", "url": "https://eips.ethereum.org/EIPS/eip-7594" }],
      "stats": {
        "totalNetworks": 3,
        "activeNetworks": 1,
        "inactiveNetworks": 2,
        "networkNames": ["fusaka-devnet-5"]
      }
    }
  },
  "networks": {
    "fusaka-devnet-5": {
      "name": "fusaka-devnet-5",
      "repository": "ethpandaops/fusaka-devnets",
      "path": "network-configs/devnet-5",
      "url": "https://github.com/ethpandaops/fusaka-devnets/tree/main/network-configs/devnet-5",
      "status": "active",
      "lastUpdated": "2026-05-04T15:30:00Z",
      "chainId": 7088110746,
      "genesisConfig": {
        "genesisTime": 1234567890,
        "consensusLayer": [{ "path": "config.yaml", "url": "https://..." }]
      },
      "serviceUrls": {
        "dora": "https://dora.fusaka-devnet-5.ethpandaops.io",
        "jsonRpc": "https://rpc.fusaka-devnet-5.ethpandaops.io"
      },
      "images": {
        "clients": [{ "name": "geth", "version": "v1.15.0" }]
      },
      "forks": {
        "consensus": { "fulu": { "epoch": 272640 } },
        "execution": { "prague": { "block": 0, "timestamp": 1234567890 } }
      },
      "blobSchedule": [{ "epoch": 274176, "maxBlobsPerBlock": 15 }],
      "selfHostedDns": false
    }
  },
  "clients": {
    "geth": {
      "name": "geth",
      "displayName": "Geth",
      "repository": "ethereum/go-ethereum",
      "type": "execution",
      "branch": "master",
      "logo": "https://ethpandaops.io/img/clients/geth.jpg",
      "latestVersion": "v1.15.0"
    }
  },
  "lastUpdate": "2026-05-04T15:30:00Z",
  "duration": 1.25,
  "providers": [{ "name": "github" }, { "name": "static" }]
}
```

The `inventory`, `validator-ranges`, and `eip7870-reference-nodes` subcommands each produce their own JSON artifacts uploaded to S3 under their configured keys.

## License

Apache 2.0
