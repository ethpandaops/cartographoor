package static

import (
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/ethpandaops/cartographoor/pkg/discovery"
	"github.com/sirupsen/logrus"
)

// Provider implements the discovery.Provider interface for static networks.
type Provider struct {
	log *logrus.Logger
}

// NewProvider creates a new static provider.
func NewProvider(log *logrus.Logger) (*Provider, error) {
	log = log.WithField("provider", "static").Logger

	return &Provider{
		log: log,
	}, nil
}

// Name returns the name of the provider.
func (p *Provider) Name() string {
	return "static"
}

// Discover returns networks from configuration.
func (p *Provider) Discover(ctx context.Context, config discovery.Config) (map[string]discovery.Network, error) {
	p.log.Info("Discovering static networks")

	networks := make(map[string]discovery.Network)

	// Process each configured static network
	for _, staticNet := range config.Static.Networks {
		// Map service URLs from config to ServiceURLs struct
		serviceURLs := &discovery.ServiceURLs{}

		for key, value := range staticNet.ServiceURLs {
			switch strings.ToLower(key) {
			case "faucet":
				serviceURLs.Faucet = value
			case "jsonrpc":
				serviceURLs.JSONRPC = value
			case "beaconrpc":
				serviceURLs.BeaconRPC = value
			case "explorer", "etherscan":
				serviceURLs.Explorer = value
			case "beaconexplorer":
				serviceURLs.BeaconExplorer = value
			case "forkmon":
				serviceURLs.Forkmon = value
			case "assertoor":
				serviceURLs.Assertoor = value
			case "dora":
				serviceURLs.Dora = value
			case "checkpointsync":
				serviceURLs.CheckpointSync = value
			case "blobscan":
				serviceURLs.Blobscan = value
			case "ethstats":
				serviceURLs.Ethstats = value
			case "devnetspec":
				serviceURLs.DevnetSpec = value
			case "blobarchive":
				serviceURLs.BlobArchive = value
			case "forky":
				serviceURLs.Forky = value
			case "tracoor":
				serviceURLs.Tracoor = value
			case "syncoor":
				serviceURLs.Syncoor = value
			}
		}

		// Create network from configuration
		network := discovery.Network{
			Name:         staticNet.Name,
			Description:  staticNet.Description,
			Status:       "active", // All configured networks are active by definition
			ChainID:      staticNet.ChainID,
			LastUpdated:  time.Now(),
			ServiceURLs:  serviceURLs,
			Forks:        staticNet.Forks,
			BlobSchedule: staticNet.BlobSchedule,
		}

		// Add genesis config if genesis time is provided
		if staticNet.GenesisTime > 0 || staticNet.ConfigURL != "" {
			u, err := url.Parse(staticNet.ConfigURL)
			if err != nil {
				p.log.Errorf("Error parsing config url for static network %s: %v", staticNet.Name, err)

				continue
			}

			network.GenesisConfig = &discovery.GenesisConfig{
				GenesisTime:  staticNet.GenesisTime,
				GenesisDelay: staticNet.GenesisDelay,
				Metadata: []discovery.ConfigFile{
					{URL: staticNet.ConfigURL, Path: u.Path},
				},
			}
		}

		networks[staticNet.Name] = network

		p.log.WithField("network", staticNet.Name).Info("Discovered static network")
	}

	p.log.WithField("count", len(networks)).Info("Static network discovery complete")

	return networks, nil
}
