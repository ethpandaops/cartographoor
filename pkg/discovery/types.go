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
	Links         []Link         `json:"links,omitempty"`
	Status        string         `json:"status"`
	LastUpdated   time.Time      `json:"lastUpdated"`
	ChainID       uint64         `json:"chainId,omitempty"`
	GenesisConfig *GenesisConfig `json:"genesisConfig,omitempty"`
	ServiceURLs   *ServiceURLs   `json:"serviceUrls,omitempty"`
	Images        *Images        `json:"images,omitempty"`
	HiveURL       string         `json:"hiveUrl,omitempty"`
	SelfHostedDNS bool           `json:"selfHostedDns"`
	Forks         *ForksConfig   `json:"forks,omitempty"`
}

// Link represents a related link with title and URL.
type Link struct {
	Title string `json:"title" mapstructure:"title"`
	URL   string `json:"url" mapstructure:"url"`
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
	Syncoor        string `json:"syncoor,omitempty"`
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

// RepositoryMetadata contains metadata for a repository.
type RepositoryMetadata struct {
	DisplayName string `json:"displayName"`
	Description string `json:"description,omitempty"`
	Links       []Link `json:"links"`
	Image       string `json:"image,omitempty"`
	Stats       Stats  `json:"stats"`
}

// Stats contains statistics and metrics about repository networks.
type Stats struct {
	TotalNetworks    int      `json:"totalNetworks"`
	ActiveNetworks   int      `json:"activeNetworks"`
	InactiveNetworks int      `json:"inactiveNetworks"`
	NetworkNames     []string `json:"networkNames,omitempty"`
}

// ClientInfo represents details about an Ethereum client.
type ClientInfo struct {
	Name          string `json:"name"`
	DisplayName   string `json:"displayName"`
	Repository    string `json:"repository"`
	Type          string `json:"type"`
	Branch        string `json:"branch"`
	Logo          string `json:"logo"`
	LatestVersion string `json:"latestVersion,omitempty"`
	WebsiteURL    string `json:"websiteUrl,omitempty"`
	DocsURL       string `json:"docsUrl,omitempty"`
}

// ProviderInfo represents serializable information about a provider.
type ProviderInfo struct {
	Name string `json:"name"`
}

// Result represents the result of a discovery operation.
type Result struct {
	NetworkMetadata map[string]RepositoryMetadata `json:"networkMetadata"`
	Networks        map[string]Network            `json:"networks"`
	Clients         map[string]ClientInfo         `json:"clients"`
	LastUpdate      time.Time                     `json:"lastUpdate"`
	Duration        float64                       `json:"duration"`
	Providers       []ProviderInfo                `json:"providers"`
}

// GitHubRepositoryConfig represents the configuration for a GitHub repository source.
type GitHubRepositoryConfig struct {
	Name        string `mapstructure:"name"`
	NamePrefix  string `mapstructure:"namePrefix"`
	DisplayName string `mapstructure:"displayName"`
	Description string `mapstructure:"description"`
	Image       string `mapstructure:"image"`
	Links       []Link `mapstructure:"links"`
}

// StaticNetworkConfig represents the configuration for a static network.
type StaticNetworkConfig struct {
	Name        string            `mapstructure:"name"`
	Description string            `mapstructure:"description"`
	ChainID     uint64            `mapstructure:"chainId"`
	GenesisTime uint64            `mapstructure:"genesisTime"`
	ServiceURLs map[string]string `mapstructure:"serviceUrls"`
	Forks       *ForksConfig      `mapstructure:"forks"`
}

// ForksConfig represents fork configuration for both consensus and execution layers.
type ForksConfig struct {
	Consensus map[string]ForkConfig `mapstructure:"consensus"`
}

// ForkConfig represents configuration for a specific fork.
type ForkConfig struct {
	Epoch             uint64            `mapstructure:"epoch"`
	MinClientVersions map[string]string `mapstructure:"minClientVersions"`
}

// Config represents the configuration for the discovery service.
type Config struct {
	Interval time.Duration `mapstructure:"interval"`
	Static   struct {
		Networks []StaticNetworkConfig `mapstructure:"networks"`
	} `mapstructure:"static"`
	GitHub struct {
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
