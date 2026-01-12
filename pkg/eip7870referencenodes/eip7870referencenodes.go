// Package eip7870referencenodes generates startup commands for EIP-7870 reference execution nodes.
// It fetches configuration from ethereum-helm-charts and the platform repository,
// merges base commands with client and feature overlays, and outputs JSON for consumption.
package eip7870referencenodes

import "time"

// Result represents the complete output structure for EIP-7870 reference node commands.
type Result struct {
	ReferenceNodes map[string]*ClientCommand `json:"eip7870ReferenceNodes"`
	Metadata       *Metadata                 `json:"metadata"`
}

// ClientCommand represents the startup command information for a single execution client.
type ClientCommand struct {
	Client         string            `json:"client"`
	DisplayName    string            `json:"displayName"`
	Image          ImageInfo         `json:"image"`
	StartupCommand string            `json:"startupCommand"`
	ArgsBreakdown  ArgsBreakdown     `json:"argsBreakdown"`
	EnvVars        map[string]string `json:"envVars,omitempty"`
}

// ImageInfo contains container image information.
type ImageInfo struct {
	Repository string `json:"repository"`
	Tag        string `json:"tag"`
}

// ArgsBreakdown shows the source of each argument category.
type ArgsBreakdown struct {
	Base           []string `json:"base"`
	ClientSpecific []string `json:"clientSpecific"`
	Feature7870    []string `json:"feature7870"`
}

// Metadata contains information about the data sources.
type Metadata struct {
	LastUpdated time.Time   `json:"lastUpdated"`
	Sources     SourcesInfo `json:"sources"`
}

// SourcesInfo contains information about the source repositories.
type SourcesInfo struct {
	HelmCharts RepositoryInfo `json:"helmCharts"`
	Platform   RepositoryInfo `json:"platform"`
}

// RepositoryInfo contains information about a GitHub repository.
type RepositoryInfo struct {
	Repository string `json:"repository"`
	Branch     string `json:"branch"`
	Commit     string `json:"commit,omitempty"`
}

// Config represents the configuration for EIP-7870 reference nodes generation.
type Config struct {
	Enabled bool `mapstructure:"enabled"`

	// HelmChartsRepository is the source for base command templates.
	HelmChartsRepository RepositoryConfig `mapstructure:"helmChartsRepository"`

	// PlatformRepository is the source for client and feature overlays.
	PlatformRepository PlatformRepositoryConfig `mapstructure:"platformRepository"`

	// Clients is the list of execution clients to generate commands for.
	Clients []string `mapstructure:"clients"`

	// SecretPatterns is a list of patterns to redact in the output.
	SecretPatterns []string `mapstructure:"secretPatterns"`

	// Storage configuration for the output file.
	Storage StorageConfig `mapstructure:"storage"`
}

// RepositoryConfig contains GitHub repository configuration.
type RepositoryConfig struct {
	Name   string `mapstructure:"name"`
	Branch string `mapstructure:"branch"`
}

// PlatformRepositoryConfig extends RepositoryConfig with environment.
type PlatformRepositoryConfig struct {
	Name        string `mapstructure:"name"`
	Branch      string `mapstructure:"branch"`
	Environment string `mapstructure:"environment"`
}

// StorageConfig contains S3 storage configuration for the output.
type StorageConfig struct {
	Key string `mapstructure:"key"`
}

// HelmChartValues represents the parsed values from a Helm chart values.yaml.
type HelmChartValues struct {
	// DefaultCommandArgsTemplate is used by some clients (reth, besu, etc.)
	DefaultCommandArgsTemplate string `yaml:"defaultCommandArgsTemplate"`
	// DefaultCommandTemplate is used by clients that inline args (geth, erigon, etc.)
	DefaultCommandTemplate string `yaml:"defaultCommandTemplate"`
	HTTPPort               int    `yaml:"httpPort"`
	WSPort                 int    `yaml:"wsPort"`
	AuthPort               int    `yaml:"authPort"`
	MetricsPort            int    `yaml:"metricsPort"`
	P2PPort                int    `yaml:"p2pPort"`
	P2PNodePort            struct {
		Enabled bool `yaml:"enabled"`
	} `yaml:"p2pNodePort"`
}

// PlatformClientConfig represents the client configuration from platform repo.
// The yaml tag uses "ethereum-node" to match the actual field name in platform YAML files.
type PlatformClientConfig struct {
	//nolint:tagliatelle // Must match external YAML field name "ethereum-node"
	EthereumNode struct {
		Global struct {
			ClientArgs struct {
				Clients struct {
					Execution map[string][]string `yaml:"execution"`
				} `yaml:"clients"`
			} `yaml:"clientArgs"`
		} `yaml:"global"`
	} `yaml:"ethereum-node"`
}

// ClientImageConfig represents the image configuration for a specific client.
type ClientImageConfig struct {
	Enabled bool `yaml:"enabled"`
	Image   struct {
		Repository string `yaml:"repository"`
		Tag        string `yaml:"tag"`
	} `yaml:"image"`
}

// PlatformFeatureConfig represents the 7870-reference feature configuration.
// The yaml tags use exact field names to match platform YAML files.
type PlatformFeatureConfig struct {
	Global struct {
		ClientArgs struct {
			Features map[string]struct {
				Execution map[string][]string `yaml:"execution"`
			} `yaml:"features"`
		} `yaml:"clientArgs"`
	} `yaml:"global"`
	//nolint:tagliatelle // Must match external YAML field name "ethereum-node"
	EthereumNode map[string]struct {
		ExtraEnv []struct {
			Name  string `yaml:"name"`
			Value string `yaml:"value"`
		} `yaml:"extraEnv"`
		Image struct {
			Repository string `yaml:"repository"`
			Tag        string `yaml:"tag"`
		} `yaml:"image"`
		Annotations map[string]string `yaml:"annotations"`
	} `yaml:"ethereum-node"`
}

// ClientDisplayNames maps client identifiers to display names.
var ClientDisplayNames = map[string]string{
	"besu":       "Besu",
	"geth":       "Geth",
	"reth":       "Reth",
	"nethermind": "Nethermind",
	"erigon":     "Erigon",
	"ethrex":     "Ethrex",
}

// DefaultClients is the default list of execution clients to process.
var DefaultClients = []string{"besu", "geth", "reth", "nethermind", "erigon", "ethrex"}

// DefaultSecretPatterns is the default list of patterns to redact.
var DefaultSecretPatterns = []string{
	"<path:/secrets/",
	"argocd-vault-plugin",
}
