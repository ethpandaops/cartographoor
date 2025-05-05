package discovery

import (
	"context"
	"time"
)

// Network represents an Ethereum network.
type Network struct {
	Name        string    `json:"name"`
	Repository  string    `json:"repository"`
	Path        string    `json:"path"`
	URL         string    `json:"url,omitempty"`
	Description string    `json:"description,omitempty"`
	Status      string    `json:"status"`
	LastUpdated time.Time `json:"lastUpdated"`
}

// Result represents the result of a discovery operation.
type Result struct {
	Networks   []Network  `json:"networks"`
	LastUpdate time.Time  `json:"lastUpdate"`
	Duration   float64    `json:"duration"`
	Providers  []Provider `json:"providers"`
}

// Config represents the configuration for the discovery service.
type Config struct {
	Interval time.Duration `mapstructure:"interval"`
	GitHub   struct {
		Repositories []string `mapstructure:"repositories"`
		Token        string   `mapstructure:"token"`
	} `mapstructure:"github"`
}

// Provider is the interface that all discovery providers must implement.
type Provider interface {
	// Name returns the name of the provider.
	Name() string

	// Discover discovers networks.
	Discover(ctx context.Context, config Config) ([]Network, error)
}

// ResultHandler is a function that handles discovery results.
type ResultHandler func(Result)