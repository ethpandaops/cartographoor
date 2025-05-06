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
		Description: "Ethereum Mainnet",
		Status:      "active",
		LastUpdated: time.Now(),
		ServiceURLs: &discovery.ServiceURLs{
			Ethstats:       "https://ethstats.mainnet.ethpandaops.io",
			Forkmon:        "https://forkmon.mainnet.ethpandaops.io",
			CheckpointSync: "https://checkpoint-sync.mainnet.ethpandaops.io",
			BlobArchive:    "https://blob-archive.mainnet.ethpandaops.io",
			Forky:          "https://forky.mainnet.ethpandaops.io",
			Tracoor:        "https://tracoor.mainnet.ethpandaops.io",
		},
	}

	p.log.Info("Discovered Mainnet")

	// Add Sepolia Testnet
	networks["sepolia"] = discovery.Network{
		Name:        "sepolia",
		Description: "Sepolia Testnet",
		Status:      "active",
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

	// Add Hoodi Network
	networks["hoodi"] = discovery.Network{
		Name:        "hoodi",
		Description: "Hoodi Network",
		Status:      "active",
		LastUpdated: time.Now(),
		ServiceURLs: &discovery.ServiceURLs{
			Dora:           "https://dora.hoodi.ethpandaops.io",
			BeaconExplorer: "https://dora.hoodi.ethpandaops.io",
			CheckpointSync: "https://checkpoint-sync.hoodi.ethpandaops.io",
			Ethstats:       "https://ethstats.hoodi.ethpandaops.io",
			Forkmon:        "https://forkmon.hoodi.ethpandaops.io",
			Assertoor:      "https://assertoor.hoodi.ethpandaops.io",
			BlobArchive:    "https://blob-archive.hoodi.ethpandaops.io",
			Forky:          "https://forky.hoodi.ethpandaops.io",
			Tracoor:        "https://tracoor.hoodi.ethpandaops.io",
		},
	}

	p.log.Info("Discovered Hoodi")

	return networks, nil
}
