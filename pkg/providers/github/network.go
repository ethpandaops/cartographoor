package github

import (
	"context"
	"fmt"
	"net/http"
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
	HiveURL      string
	Images       struct {
		URL     string
		Clients []discovery.ClientImage
		Tools   []discovery.ToolImage
	}
}

// checkHiveAvailability checks if the hive is available for a network.
func (p *Provider) checkHiveAvailability(
	ctx context.Context,
	owner, repo, networkName string,
) (string, error) {

	// Example: fusaka-devnets -> fusaka
	devnetid := strings.ReplaceAll(repo, "-devnets", "")

	hiveURL := fmt.Sprintf(
		"https://hive.ethpandaops.io/#/group/%s-%s",
		devnetid,
		networkName,
	)

	// Check if the hive test listing file is available.
	hiveListingURL := fmt.Sprintf(
		"https://hive.ethpandaops.io/%s-%s/listing.jsonl",
		devnetid,
		networkName,
	)

	resp, err := p.httpClient.Get(hiveListingURL)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("hive is not available for network: %s", networkName)
	}

	return hiveURL, nil
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
		HiveURL:     config.HiveURL,
		LastUpdated: time.Now(),
	}

	// If network is active, add service URLs and GenesisConfig
	if config.Status == "active" {
		if config.Domain != "" {
			// Add service URLs
			network.ServiceURLs = p.getServiceURLs(ctx, config.Domain)

			// Add GenesisConfig if we have config files
			if len(config.ConfigFiles) > 0 {
				network.GenesisConfig = p.buildGenesisConfig(config)
			}
		}

		// Add images information if any exists
		if len(config.Images.Clients) > 0 || len(config.Images.Tools) > 0 {
			network.Images = &discovery.Images{
				URL:     config.Images.URL,
				Clients: config.Images.Clients,
				Tools:   config.Images.Tools,
			}
		}

		// Add hive information if any exists.
		if config.HiveURL != "" {
			network.HiveURL = config.HiveURL
		}

		// Try to extract chainId and genesisTime from genesis.json
		chainID, genesisTime, err := p.parseGenesisJSON(ctx, config.Owner, config.Repo, config.Name)
		if err == nil {
			// Set the ChainID in the Network struct
			network.ChainID = chainID

			// Set the GenesisTime in the GenesisConfig struct if it exists
			if network.GenesisConfig != nil {
				network.GenesisConfig.GenesisTime = genesisTime
			}
		} else {
			p.log.WithError(err).WithField("network", config.Name).Debug("Failed to parse genesis.json")
		}
	}

	return network
}

// buildGenesisConfig builds a GenesisConfig from network config files.
func (p *Provider) buildGenesisConfig(config *NetworkConfig) *discovery.GenesisConfig {
	genesisConfig := &discovery.GenesisConfig{
		ConsensusLayer: []discovery.ConfigFile{},
		ExecutionLayer: []discovery.ConfigFile{},
		Metadata:       []discovery.ConfigFile{},
		API:            []discovery.ConfigFile{},
	}

	for _, configPath := range config.ConfigFiles {
		// Always use the config domain prefix for all paths
		url := fmt.Sprintf("https://config.%s%s", config.Domain, configPath)

		configFile := discovery.ConfigFile{
			Path: configPath,
			URL:  url,
		}

		// Categorize files based on path and filename
		if strings.HasPrefix(configPath, "/api/") {
			// API endpoints
			genesisConfig.API = append(genesisConfig.API, configFile)
		} else if strings.HasPrefix(configPath, "/metadata/") {
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
