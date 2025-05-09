package static

import (
	"context"
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

// Discover returns hardcoded networks.
func (p *Provider) Discover(ctx context.Context, config discovery.Config) (map[string]discovery.Network, error) {
	p.log.Info("Discovering static networks")

	networks := make(map[string]discovery.Network)

	// Add Ethereum Mainnet
	networks["mainnet"] = discovery.Network{
		Name:        "mainnet",
		Description: "Production Ethereum network",
		Status:      "active",
		ChainID:     1,
		LastUpdated: time.Now(),
		ServiceURLs: &discovery.ServiceURLs{
			Ethstats:    "https://ethstats.mainnet.ethpandaops.io",
			Forkmon:     "https://forkmon.mainnet.ethpandaops.io",
			BlobArchive: "https://blob-archive.mainnet.ethpandaops.io",
			Forky:       "https://forky.mainnet.ethpandaops.io",
			Tracoor:     "https://tracoor.mainnet.ethpandaops.io",
		},
	}

	p.log.Info("Discovered Mainnet")

	// Add Sepolia Testnet
	networks["sepolia"] = discovery.Network{
		Name:        "sepolia",
		Description: "Smaller testnet for application development with controlled validator set.",
		Status:      "active",
		ChainID:     11155111,
		LastUpdated: time.Now(),
		ServiceURLs: &discovery.ServiceURLs{
			Dora:           "https://dora.sepolia.ethpandaops.io",
			BeaconExplorer: "https://dora.sepolia.ethpandaops.io",
			CheckpointSync: "https://checkpoint-sync.sepolia.ethpandaops.io",
			Ethstats:       "https://ethstats.sepolia.ethpandaops.io",
			Forkmon:        "https://forkmon.sepolia.ethpandaops.io",
			BlobArchive:    "https://blob-archive.sepolia.ethpandaops.io",
			Forky:          "https://forky.sepolia.ethpandaops.io",
			Tracoor:        "https://tracoor.sepolia.ethpandaops.io",
		},
	}

	p.log.Info("Discovered Sepolia")

	// Add Holesky Network
	networks["holesky"] = discovery.Network{
		Name:        "holesky",
		Description: "Long-term public testnet designed for staking/validator testing with high validator counts.",
		Status:      "active",
		ChainID:     17000,
		LastUpdated: time.Now(),
		ServiceURLs: &discovery.ServiceURLs{
			Dora:           "https://dora.holesky.ethpandaops.io",
			BeaconExplorer: "https://dora.holesky.ethpandaops.io",
			CheckpointSync: "https://checkpoint-sync.holesky.ethpandaops.io",
			Ethstats:       "https://ethstats.holesky.ethpandaops.io",
			Forkmon:        "https://forkmon.holesky.ethpandaops.io",
			BlobArchive:    "https://blob-archive.holesky.ethpandaops.io",
			Forky:          "https://forky.holesky.ethpandaops.io",
			Tracoor:        "https://tracoor.holesky.ethpandaops.io",
		},
	}

	p.log.Info("Discovered Holesky")

	// Add Hoodi Network
	networks["hoodi"] = discovery.Network{
		Name:        "hoodi",
		Description: "New public testnet (launched March 2025) designed for validator testing and protocol upgrades, replacing Holesky.",
		Status:      "active",
		ChainID:     560048,
		LastUpdated: time.Now(),
		ServiceURLs: &discovery.ServiceURLs{
			Dora:           "https://dora.hoodi.ethpandaops.io",
			BeaconExplorer: "https://dora.hoodi.ethpandaops.io",
			CheckpointSync: "https://checkpoint-sync.hoodi.ethpandaops.io",
			Forkmon:        "https://forkmon.hoodi.ethpandaops.io",
			Forky:          "https://forky.hoodi.ethpandaops.io",
			Tracoor:        "https://tracoor.hoodi.ethpandaops.io",
		},
	}

	p.log.Info("Discovered Hoodi")

	return networks, nil
}
