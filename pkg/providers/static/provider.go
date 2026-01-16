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
			case "cbt":
				serviceURLs.Cbt = value
			case "cbtapi":
				serviceURLs.CbtApi = value
			}
		}

		// Get timing parameters with defaults
		slotsPerEpoch := staticNet.SlotsPerEpoch
		if slotsPerEpoch == 0 {
			slotsPerEpoch = 32 // Default mainnet preset
		}

		slotDurationSeconds := staticNet.SlotDurationSeconds
		if slotDurationSeconds == 0 {
			slotDurationSeconds = 12 // Default mainnet preset
		}

		// Calculate timestamps for consensus forks
		forks := p.calculateForkTimestamps(staticNet.Forks, staticNet.GenesisTime, slotsPerEpoch, slotDurationSeconds)

		// Calculate timestamps for blob schedule
		blobSchedule := p.calculateBlobScheduleTimestamps(staticNet.BlobSchedule, staticNet.GenesisTime, slotsPerEpoch, slotDurationSeconds)

		// Create network from configuration
		network := discovery.Network{
			Name:         staticNet.Name,
			Description:  staticNet.Description,
			Status:       "active", // All configured networks are active by definition
			ChainID:      staticNet.ChainID,
			LastUpdated:  time.Now(),
			ServiceURLs:  serviceURLs,
			Forks:        forks,
			BlobSchedule: blobSchedule,
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

// calculateForkTimestamps calculates timestamps for consensus forks based on epoch and timing parameters.
func (p *Provider) calculateForkTimestamps(
	forks *discovery.ForksConfig,
	genesisTime, slotsPerEpoch, slotDurationSeconds uint64,
) *discovery.ForksConfig {
	if forks == nil {
		return nil
	}

	result := &discovery.ForksConfig{
		Execution: forks.Execution, // Execution forks already have timestamps in config
	}

	if len(forks.Consensus) > 0 {
		result.Consensus = make(map[string]discovery.ConsensusForkConfig, len(forks.Consensus))

		for name, fork := range forks.Consensus {
			// Calculate timestamp if not already set
			timestamp := fork.Timestamp
			if timestamp == 0 && genesisTime > 0 {
				timestamp = genesisTime + (fork.Epoch * slotsPerEpoch * slotDurationSeconds)
			}

			result.Consensus[name] = discovery.ConsensusForkConfig{
				Epoch:             fork.Epoch,
				Timestamp:         timestamp,
				MinClientVersions: fork.MinClientVersions,
			}
		}
	}

	return result
}

// calculateBlobScheduleTimestamps calculates timestamps for blob schedule entries.
func (p *Provider) calculateBlobScheduleTimestamps(
	schedule []discovery.BlobSchedule,
	genesisTime, slotsPerEpoch, slotDurationSeconds uint64,
) []discovery.BlobSchedule {
	if len(schedule) == 0 {
		return nil
	}

	result := make([]discovery.BlobSchedule, len(schedule))

	for i, entry := range schedule {
		// Calculate timestamp if not already set
		timestamp := entry.Timestamp
		if timestamp == 0 && genesisTime > 0 {
			timestamp = genesisTime + (entry.Epoch * slotsPerEpoch * slotDurationSeconds)
		}

		result[i] = discovery.BlobSchedule{
			Epoch:            entry.Epoch,
			Timestamp:        timestamp,
			MaxBlobsPerBlock: entry.MaxBlobsPerBlock,
		}
	}

	return result
}
