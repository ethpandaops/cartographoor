# Cartographoor

Cartographoor is a service that discovers and maps active Ethereum networks in the ethpandaops ecosystem. It periodically scans configured GitHub repositories and static configuration, discovers network metadata (clients, forks, genesis config, service URLs, blob schedules), and uploads aggregated results to S3 (or an S3-compatible store) as structured JSON files.

Alongside the core network discovery, the binary ships additional generators that consume the discovered data and produce their own artifacts: a Dora-based client **inventory**, **validator-ranges**, and **EIP-7870 reference node** startup commands. A standalone **client library** (`pkg/client`) is also provided for other Go services that want to consume `networks.json`.

## Architecture

Cartographoor follows a modular, provider-based architecture driven by Viper-based YAML configuration with environment variable substitution.

- **Discovery Service** (`pkg/discovery`): orchestrates discovery across providers, manages intervals/scheduling, aggregates results, and reports errors. Also performs client discovery/identification (`clients.go`).
- **Discovery Providers** (`pkg/providers/*`): pluggable sources implementing the `discovery.Provider` interface. Currently `github` (scans `network-configs/` directories) and `static` (hardcoded networks like mainnet/sepolia/hoodi from config).
- **Storage Providers** (`pkg/storage/s3`): upload discovery results to S3 / S3-compatible stores. Supports download too (used by downstream generators).
- **Generators** (consume the data, run as separate subcommands):
  - `pkg/inventory` — generates a network inventory from Dora APIs, with optional DNS validation.
  - `pkg/validatorranges` — downloads `networks.json` and builds validator range data from Ansible inventory files.
  - `pkg/eip7870referencenodes` — builds EIP-7870 reference node startup commands from ethereum-helm-charts + platform repos.
- **Client library** (`pkg/client`): embeddable consumer with `MemoryProvider` and `RedisProvider` implementations of the `client.Provider` interface; fetches and caches `networks.json` for use in other services.

### Provider interface
```go
type Provider interface {
    Name() string
    Discover(ctx context.Context, config Config) (map[string]Network, error)
}
```
Storage providers implement `Upload(ctx context.Context, result discovery.Result) error` (and `Download`/`Initialize`).

### Data flow
1. Viper loads YAML config with `${VAR}` env substitution.
2. Discovery service is created with configured providers (github + static).
3. Providers are called on an interval (or once with `--once`).
4. Results are merged into a single `discovery.Result` (networks, client metadata, per-repository metadata, stats).
5. The result is uploaded to the configured storage backend.

The canonical data model lives in `pkg/discovery/types.go` (`Network`, `Result`, `ClientInfo`, `ForksConfig`, `GenesisConfig`, `ServiceURLs`, `Images`, `BlobSchedule`, etc.). Read it before changing the output shape — downstream consumers (the client library, validator-ranges) depend on it.

## Subcommands

The CLI (`cmd/cartographoor`, Cobra) exposes:

- `cartographoor run` — core discovery loop; uploads `networks.json`. Supports `--once`.
- `cartographoor inventory --config <file>` — generate inventory from Dora APIs.
- `cartographoor validator-ranges --config <file>` — generate validator ranges.
- `cartographoor eip7870-reference-nodes --config <file>` — generate EIP-7870 reference node commands.

Each subcommand is wired up in `cmd/cartographoor/cmd/root.go` and reads the same YAML config format (relevant sections only). In production these run as separate scheduled GitHub Actions workflows (see `.github/workflows/`) against `.github/config.production.yaml`.

## Key Commands

### Build and Run
```bash
make build          # Build the binary into build/
make run            # Build and run with config.example.yaml
make test           # Run all tests (go test -v ./...)
make lint           # Run golangci-lint
make clean          # Remove build artifacts
```

### Testing and Linting
When making code changes, ALWAYS run these before considering the task complete:
- `make test` — ensures all tests pass
- `make lint` — ensures code quality standards are met

If you cannot find these commands, ask the user for the correct commands to run.

## Code Standards

Apply the ethpandaops Go standards (loaded from `~/.claude/ethpandaops/code-standards/go/CLAUDE.md`) plus:

- Target the Go version in `go.mod` / `.tool-versions` (currently Go 1.26).
- Wrap errors with context: `fmt.Errorf("...: %w", err)`. Return early on errors.
- Use structured logging with logrus and meaningful fields.
- Accept `context.Context` as the first parameter for any I/O; propagate it and respect cancellation/timeouts.
- Keep providers stateless and thread-safe; handle rate limits and retries gracefully.
- Acronyms keep consistent case (`URL`, `ID`, `RPC`), not `Url`/`Id`.
- Use Viper for config and support `${VAR}` substitution; document new options in `config.example.yaml`.

## Testing

- Write unit tests for all new functionality; name files `*_test.go`.
- Use table-driven tests for multiple scenarios.
- Mock external dependencies (GitHub API, S3, Redis, Dora). Mocks live alongside their packages (e.g. `pkg/providers/github/mock`, `pkg/client/mocks`) and are generated with `go generate` / mockgen — regenerate them when interfaces change.

## Configuration

- YAML configuration with environment variable substitution (`${VAR}` syntax via `pkg/utils/envsubst.go`).
- A GitHub token is strongly recommended to avoid rate limiting (`discovery.github.token`, typically `${GITHUB_TOKEN}`).
- `config.example.yaml` documents the discovery/storage/inventory/validator-ranges options; `.github/config.production.yaml` is the live production config (also covers `eip7870ReferenceNodes`).
