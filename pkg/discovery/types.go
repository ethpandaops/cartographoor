package discovery

import (
	"context"
	"time"
)

// Network represents an Ethereum network.
type Network struct {
	Name          string         `json:"name"`
	Repository    string         `json:"repository,omitempty"`
	Path          string         `json:"path,omitempty"`
	URL           string         `json:"url,omitempty"`
	Description   string         `json:"description,omitempty"`
	Status        string         `json:"status"`
	LastUpdated   time.Time      `json:"lastUpdated"`
	ChainID       uint64         `json:"chainId,omitempty"`
	GenesisConfig *GenesisConfig `json:"genesisConfig,omitempty"`
	ServiceURLs   *ServiceURLs   `json:"serviceUrls,omitempty"`
	Images        *Images        `json:"images,omitempty"`
}

// ServiceURLs contains URLs for various network services.
type ServiceURLs struct {
	Faucet         string `json:"faucet,omitempty"`
	JSONRPC        string `json:"jsonRpc,omitempty"`
	BeaconRPC      string `json:"beaconRpc,omitempty"`
	Explorer       string `json:"explorer,omitempty"`
	BeaconExplorer string `json:"beaconExplorer,omitempty"`
	Forkmon        string `json:"forkmon,omitempty"`
	Assertoor      string `json:"assertoor,omitempty"`
	Dora           string `json:"dora,omitempty"`
	CheckpointSync string `json:"checkpointSync,omitempty"`
	Blobscan       string `json:"blobscan,omitempty"`
	Ethstats       string `json:"ethstats,omitempty"`
	DevnetSpec     string `json:"devnetSpec,omitempty"`
	BlobArchive    string `json:"blobArchive,omitempty"`
	Forky          string `json:"forky,omitempty"`
	Tracoor        string `json:"tracoor,omitempty"`
}

// GenesisConfig represents the configuration URLs for a network.
type GenesisConfig struct {
	ConsensusLayer []ConfigFile `json:"consensusLayer,omitempty"`
	ExecutionLayer []ConfigFile `json:"executionLayer,omitempty"`
	Metadata       []ConfigFile `json:"metadata,omitempty"`
	API            []ConfigFile `json:"api,omitempty"`
	GenesisTime    uint64       `json:"genesisTime,omitempty"`
}

// ConfigFile represents a configuration file URL.
type ConfigFile struct {
	Path string `json:"path"`
	URL  string `json:"url"`
}

// Result represents the result of a discovery operation.
type Result struct {
	Networks   map[string]Network `json:"networks"`
	LastUpdate time.Time          `json:"lastUpdate"`
	Duration   float64            `json:"duration"`
	Providers  []Provider         `json:"providers"`
}

// GitHubRepositoryConfig represents the configuration for a GitHub repository source.
type GitHubRepositoryConfig struct {
	Name       string `mapstructure:"name"`
	NamePrefix string `mapstructure:"namePrefix"`
}

// Config represents the configuration for the discovery service.
type Config struct {
	Interval time.Duration `mapstructure:"interval"`
	GitHub   struct {
		Repositories []GitHubRepositoryConfig `mapstructure:"repositories"`
		Token        string                   `mapstructure:"token"`
	} `mapstructure:"github"`
}

// Provider is the interface that all discovery providers must implement.
type Provider interface {
	// Name returns the name of the provider.
	Name() string

	// Discover discovers networks and returns them as a map with network names as keys.
	Discover(ctx context.Context, config Config) (map[string]Network, error)
}

// ResultHandler is a function that handles discovery results.
type ResultHandler func(Result)

// Images contains information about client and tool images used in the network.
type Images struct {
	URL     string        `json:"url,omitempty"`
	Clients []ClientImage `json:"clients,omitempty"`
	Tools   []ToolImage   `json:"tools,omitempty"`
}

// ClientImage represents a client image with name and version.
type ClientImage struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ToolImage represents a tool image with name and version.
type ToolImage struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}
