# Network Status

A service that discovers active networks in the ethpandaops ecosystem.

## Overview

Network Status is a Go application that periodically scans and discovers active Ethereum networks maintained by the ethpandaops team. It aggregates network information and serves it as a structured JSON file, making it easier to maintain an up-to-date view of available networks.

## Features

- Periodic discovery of Ethereum networks
- Multiple discovery sources (starting with GitHub repositories)
- Configurable discovery intervals and sources
- Serves a `networks.json` file with current status information

## How It Works

Network Status uses a configuration file to determine:
- Which repositories to scan
- How often to perform discovery
- Where to look for network configurations

The service identifies networks by checking for directories within the `network-configs/` path of specified repositories.

## Example

The service scans repositories like `ethpandaops/dencun-devnets`, which contains networks such as:
- devnet-4, devnet-5, ..., devnet-12
- gsf-1, gsf-2
- msf-1
- sepolia-sf1

## Configuration

Network Status is configured via a YAML file that specifies discovery sources, intervals, and output settings.

## Usage

```bash
# Run with default configuration
network-status

# Run with custom configuration
network-status --config=/path/to/config.yaml
```

## License

Apache 2.0