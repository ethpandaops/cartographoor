package github

import (
	"context"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/ethpandaops/cartographoor/pkg/discovery"
	gh "github.com/google/go-github/v53/github"
)

// NetworkConfig contains configuration for a network.
type NetworkConfig struct {
	Name         string
	PrefixedName string
	Repository   string
	Owner        string
	Repo         string
	Path         string
	URL          string
	Status       string
	ConfigFiles  []string
	Domain       string
}

// getNetworkConfigs gets the config files and domain for an active network.
func (p *Provider) getNetworkConfigs(
	ctx context.Context,
	client *gh.Client,
	owner, repo, kubePath, networkName string,
) ([]string, string) {
	valuesPath := path.Join(kubePath, "config", "values.yaml")
	fileContent, _, _, err := client.Repositories.GetContents(ctx, owner, repo, valuesPath, nil)

	if err == nil && fileContent != nil {
		// Parse values.yaml to extract domain and config files
		return p.parseValuesYaml(ctx, fileContent, networkName)
	}

	return nil, ""
}

// createNetwork creates a discovery.Network from a NetworkConfig.
func (p *Provider) createNetwork(ctx context.Context, config *NetworkConfig) discovery.Network {
	network := discovery.Network{
		Name:        config.Name,
		Repository:  config.Repository,
		Path:        config.Path,
		URL:         config.URL,
		Status:      config.Status,
		LastUpdated: time.Now(),
	}

	// If network is active and has configs, build the GenesisConfig
	if config.Status == "active" && config.Domain != "" && len(config.ConfigFiles) > 0 {
		network.GenesisConfig = p.buildGenesisConfig(config)
	}

	return network
}

// buildGenesisConfig builds a GenesisConfig from network config files.
func (p *Provider) buildGenesisConfig(config *NetworkConfig) *discovery.GenesisConfig {
	genesisConfig := &discovery.GenesisConfig{
		ConsensusLayer: []discovery.ConfigFile{},
		ExecutionLayer: []discovery.ConfigFile{},
		Metadata:       []discovery.ConfigFile{},
	}

	for _, configPath := range config.ConfigFiles {
		url := fmt.Sprintf("https://config.%s%s", config.Domain, configPath)

		configFile := discovery.ConfigFile{
			Path: configPath,
			URL:  url,
		}

		// Categorize files based on path and filename
		if strings.HasPrefix(configPath, "/metadata/") {
			// Add only to metadata section
			genesisConfig.Metadata = append(genesisConfig.Metadata, configFile)
		} else if strings.HasPrefix(configPath, "/cl/") {
			// Consensus layer specific paths
			genesisConfig.ConsensusLayer = append(genesisConfig.ConsensusLayer, configFile)
		} else if strings.HasPrefix(configPath, "/el/") {
			// Execution layer specific paths
			genesisConfig.ExecutionLayer = append(genesisConfig.ExecutionLayer, configFile)
		}
	}

	return genesisConfig
}
