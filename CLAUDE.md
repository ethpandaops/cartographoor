# Cartographoor

Cartographoor is a service that discovers and maps active Ethereum networks in the ethpandaops ecosystem. It periodically scans configured GitHub repositories, discovers network configurations, and uploads the aggregated results to S3 as a structured JSON file.

## Project Structure
Claude MUST read the `.cursor/rules/project_architecture.mdc` file before making any structural changes to the project.

## Code Standards  
Claude MUST read the `.cursor/rules/code_standards.mdc` file before writing any code in this project.

## Development Workflow
Claude MUST read the `.cursor/rules/development_workflow.mdc` file before making changes to build, test, or deployment configurations.

## Component Documentation
Individual components have their own CLAUDE.md files with component-specific rules. Always check for and read component-level documentation when working on specific parts of the codebase.

## Key Commands

### Build and Run
```bash
make build          # Build the binary
make test           # Run all tests
make lint           # Run golangci-lint
make run            # Build and run with example config
```

### Testing and Linting
When making code changes, ALWAYS run these commands before considering the task complete:
- `make test` - Ensures all tests pass
- `make lint` - Ensures code quality standards are met

If you cannot find these commands, ask the user for the correct commands to run.

## Important Notes

### Configuration
- The service uses YAML configuration with environment variable substitution
- GitHub token is REQUIRED to prevent rate limiting
- See `config.example.yaml` for all configuration options

### Provider Pattern
- All discovery providers must implement the `discovery.Provider` interface
- Storage providers handle uploading discovery results
- The system is designed to be easily extensible

### Error Handling
- Always wrap errors with context using `fmt.Errorf` with `%w`
- Use structured logging with logrus
- Handle external API rate limits gracefully

### Testing
- Write unit tests for all new functionality
- Mock external dependencies (GitHub API, S3, etc.)
- Use table-driven tests for multiple scenarios