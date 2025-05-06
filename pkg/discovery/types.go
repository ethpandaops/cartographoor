package discovery

import (
	"context"
	"time"
)

// Network represents an Ethereum network.
type Network struct {
	Name          string         `json:"name"`
	Repository    string         `json:"repository"`
	Path          string         `json:"path"`
	URL           string         `json:"url,omitempty"`
	Description   string         `json:"description,omitempty"`
	Status        string         `json:"status"`
	LastUpdated   time.Time      `json:"lastUpdated"`
	GenesisConfig *GenesisConfig `json:"genesisConfig,omitempty"`
}

// GenesisConfig represents the configuration URLs for a network.
type GenesisConfig struct {
	ConsensusLayer []ConfigFile `json:"consensusLayer,omitempty"`
	ExecutionLayer []ConfigFile `json:"executionLayer,omitempty"`
	Metadata       []ConfigFile `json:"metadata,omitempty"`
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
