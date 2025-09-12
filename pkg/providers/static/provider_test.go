package static

import (
	"context"
	"testing"
	"time"

	"github.com/ethpandaops/cartographoor/pkg/discovery"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockGitHubProvider is a mock implementation of the GitHub provider for testing.
type mockGitHubProvider struct{}

func (p *mockGitHubProvider) Name() string {
	return "mock-github"
}

func (p *mockGitHubProvider) Discover(ctx context.Context, config discovery.Config) (map[string]discovery.Network, error) {
	networks := make(map[string]discovery.Network)

	// Add a mock network
	networks["devnet-1"] = discovery.Network{
		Name:        "devnet-1",
		Repository:  "ethpandaops/mock-devnets",
		Path:        "network-configs/devnet-1",
		Description: "Mock Devnet 1",
		LastUpdated: time.Now(),
		ServiceURLs: &discovery.ServiceURLs{
			Faucet:    "https://faucet.devnet-1.ethpandaops.io",
			JSONRPC:   "https://rpc.devnet-1.ethpandaops.io",
			BeaconRPC: "https://beacon.devnet-1.ethpandaops.io",
		},
	}

	return networks, nil
}

func TestCombinedProviders(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	// Create config with static networks
	config := discovery.Config{}
	config.Static.Networks = []discovery.StaticNetworkConfig{
		{
			Name:        "mainnet",
			Description: "Production Ethereum network",
			ChainID:     1,
			GenesisTime: 1606824023,
			ServiceURLs: map[string]string{
				"ethstats":    "https://ethstats.mainnet.ethpandaops.io",
				"forkmon":     "https://forkmon.mainnet.ethpandaops.io",
				"blobArchive": "https://blob-archive.mainnet.ethpandaops.io",
				"forky":       "https://forky.mainnet.ethpandaops.io",
				"tracoor":     "https://tracoor.mainnet.ethpandaops.io",
			},
			Forks: &discovery.ForksConfig{
				Consensus: map[string]discovery.ForkConfig{
					"electra": {
						Epoch: 364032,
						MinClientVersions: map[string]string{
							"lighthouse": "7.0.0",
							"prysm":      "6.0.0",
						},
					},
				},
			},
		},
		{
			Name:        "sepolia",
			Description: "Smaller testnet for application development with controlled validator set.",
			ChainID:     11155111,
			GenesisTime: 1655726400,
			ServiceURLs: map[string]string{
				"dora":           "https://dora.sepolia.ethpandaops.io",
				"beaconExplorer": "https://dora.sepolia.ethpandaops.io",
				"checkpointSync": "https://checkpoint-sync.sepolia.ethpandaops.io",
				"ethstats":       "https://ethstats.sepolia.ethpandaops.io",
				"forkmon":        "https://forkmon.sepolia.ethpandaops.io",
				"blobArchive":    "https://blob-archive.sepolia.ethpandaops.io",
				"forky":          "https://forky.sepolia.ethpandaops.io",
				"tracoor":        "https://tracoor.sepolia.ethpandaops.io",
			},
		},
		{
			Name:        "holesky",
			Description: "Long-term public testnet designed for staking/validator testing with high validator counts.",
			ChainID:     17000,
			GenesisTime: 1695902400,
			ServiceURLs: map[string]string{
				"dora":           "https://dora.holesky.ethpandaops.io",
				"beaconExplorer": "https://dora.holesky.ethpandaops.io",
				"checkpointSync": "https://checkpoint-sync.holesky.ethpandaops.io",
				"ethstats":       "https://ethstats.holesky.ethpandaops.io",
				"forkmon":        "https://forkmon.holesky.ethpandaops.io",
				"blobArchive":    "https://blob-archive.holesky.ethpandaops.io",
				"forky":          "https://forky.holesky.ethpandaops.io",
				"tracoor":        "https://tracoor.holesky.ethpandaops.io",
			},
		},
		{
			Name:        "hoodi",
			Description: "New public testnet (launched March 2025) designed for validator testing and protocol upgrades, replacing Holesky.",
			ChainID:     560048,
			GenesisTime: 1742213400,
			ServiceURLs: map[string]string{
				"dora":           "https://dora.hoodi.ethpandaops.io",
				"beaconExplorer": "https://dora.hoodi.ethpandaops.io",
				"checkpointSync": "https://checkpoint-sync.hoodi.ethpandaops.io",
				"forkmon":        "https://forkmon.hoodi.ethpandaops.io",
				"forky":          "https://forky.hoodi.ethpandaops.io",
				"tracoor":        "https://tracoor.hoodi.ethpandaops.io",
			},
		},
	}

	// Create discovery service
	service, err := discovery.NewService(log, config)
	require.NoError(t, err)

	// Register both providers
	mockGitHub := &mockGitHubProvider{}
	service.RegisterProvider(mockGitHub)

	staticProvider, err := NewProvider(log)
	require.NoError(t, err)
	service.RegisterProvider(staticProvider)

	// Run discovery
	result, err := service.RunOnce(context.Background())
	require.NoError(t, err)

	// Verify we have networks from both providers
	assert.Len(t, result.Networks, 5, "Should have 5 networks (4 from static, 1 from GitHub mock)")
	assert.Contains(t, result.Networks, "mainnet")
	assert.Contains(t, result.Networks, "sepolia")
	assert.Contains(t, result.Networks, "hoodi")
	assert.Contains(t, result.Networks, "holesky")
	assert.Contains(t, result.Networks, "devnet-1")

	// Verify fork configuration is properly passed through
	mainnet := result.Networks["mainnet"]
	require.NotNil(t, mainnet.Forks)
	require.NotNil(t, mainnet.Forks.Consensus)
	require.Contains(t, mainnet.Forks.Consensus, "electra")
	assert.Equal(t, uint64(364032), mainnet.Forks.Consensus["electra"].Epoch)
	assert.Equal(t, "7.0.0", mainnet.Forks.Consensus["electra"].MinClientVersions["lighthouse"])

	// Verify the providers in the result
	providerNames := make([]string, 0, len(result.Providers))
	for _, p := range result.Providers {
		providerNames = append(providerNames, p.Name)
	}

	assert.Contains(t, providerNames, "static")
	assert.Contains(t, providerNames, "mock-github")
}

func TestProvider_Name(t *testing.T) {
	log := logrus.New()
	provider, err := NewProvider(log)
	require.NoError(t, err)

	assert.Equal(t, "static", provider.Name())
}

func TestProvider_DiscoverWithForks(t *testing.T) {
	log := logrus.New()
	provider, err := NewProvider(log)
	require.NoError(t, err)

	t.Run("network with full fork configuration", func(t *testing.T) {
		config := discovery.Config{}
		config.Static.Networks = []discovery.StaticNetworkConfig{
			{
				Name:        "test-mainnet",
				Description: "Test network with fork config",
				ChainID:     1,
				GenesisTime: 1606824023,
				ServiceURLs: map[string]string{
					"ethstats": "https://ethstats.test.io",
				},
				Forks: &discovery.ForksConfig{
					Consensus: map[string]discovery.ForkConfig{
						"electra": {
							Epoch: 364032,
							MinClientVersions: map[string]string{
								"lighthouse": "7.0.0",
								"prysm":      "6.0.0",
								"teku":       "25.4.1",
							},
						},
					},
				},
			},
		}

		networks, err := provider.Discover(context.Background(), config)
		require.NoError(t, err)
		require.Len(t, networks, 1)

		network := networks["test-mainnet"]
		require.NotNil(t, network.Forks)
		require.NotNil(t, network.Forks.Consensus)
		require.Contains(t, network.Forks.Consensus, "electra")

		electra := network.Forks.Consensus["electra"]
		assert.Equal(t, uint64(364032), electra.Epoch)
		assert.Equal(t, "7.0.0", electra.MinClientVersions["lighthouse"])
		assert.Equal(t, "6.0.0", electra.MinClientVersions["prysm"])
		assert.Equal(t, "25.4.1", electra.MinClientVersions["teku"])
	})

	t.Run("network without fork configuration", func(t *testing.T) {
		config := discovery.Config{}
		config.Static.Networks = []discovery.StaticNetworkConfig{
			{
				Name:        "test-network",
				Description: "Test network without forks",
				ChainID:     1000,
				GenesisTime: 1700000000,
				ServiceURLs: map[string]string{
					"ethstats": "https://ethstats.test.io",
				},
				// No Forks field
			},
		}

		networks, err := provider.Discover(context.Background(), config)
		require.NoError(t, err)
		require.Len(t, networks, 1)

		network := networks["test-network"]
		assert.Nil(t, network.Forks)
	})

	t.Run("fork config without minClientVersions", func(t *testing.T) {
		config := discovery.Config{}
		config.Static.Networks = []discovery.StaticNetworkConfig{
			{
				Name:        "test-holesky",
				Description: "Test network with fork but no client versions",
				ChainID:     17000,
				GenesisTime: 1695902400,
				ServiceURLs: map[string]string{
					"ethstats": "https://ethstats.test.io",
				},
				Forks: &discovery.ForksConfig{
					Consensus: map[string]discovery.ForkConfig{
						"electra": {
							Epoch: 0,
							// No MinClientVersions - this is valid for future/unknown requirements
						},
					},
				},
			},
		}

		networks, err := provider.Discover(context.Background(), config)
		require.NoError(t, err)
		require.Len(t, networks, 1)

		network := networks["test-holesky"]
		require.NotNil(t, network.Forks)
		require.NotNil(t, network.Forks.Consensus)
		require.Contains(t, network.Forks.Consensus, "electra")

		electra := network.Forks.Consensus["electra"]
		assert.Equal(t, uint64(0), electra.Epoch)
		assert.Nil(t, electra.MinClientVersions)
	})

	t.Run("multiple forks in same network", func(t *testing.T) {
		config := discovery.Config{}
		config.Static.Networks = []discovery.StaticNetworkConfig{
			{
				Name:        "test-multi-fork",
				Description: "Test network with multiple forks",
				ChainID:     2000,
				GenesisTime: 1700000000,
				ServiceURLs: map[string]string{
					"ethstats": "https://ethstats.test.io",
				},
				Forks: &discovery.ForksConfig{
					Consensus: map[string]discovery.ForkConfig{
						"electra": {
							Epoch: 100,
							MinClientVersions: map[string]string{
								"lighthouse": "7.0.0",
							},
						},
						"fulu": {
							Epoch: 200,
							MinClientVersions: map[string]string{
								"lighthouse": "8.0.0",
							},
						},
					},
				},
			},
		}

		networks, err := provider.Discover(context.Background(), config)
		require.NoError(t, err)
		require.Len(t, networks, 1)

		network := networks["test-multi-fork"]
		require.NotNil(t, network.Forks)
		require.Len(t, network.Forks.Consensus, 2)

		assert.Contains(t, network.Forks.Consensus, "electra")
		assert.Contains(t, network.Forks.Consensus, "fulu")

		assert.Equal(t, uint64(100), network.Forks.Consensus["electra"].Epoch)
		assert.Equal(t, uint64(200), network.Forks.Consensus["fulu"].Epoch)
	})
}

func TestProvider_Discover(t *testing.T) {
	log := logrus.New()
	provider, err := NewProvider(log)
	require.NoError(t, err)

	// Create config with static networks
	config := discovery.Config{}
	config.Static.Networks = []discovery.StaticNetworkConfig{
		{
			Name:        "mainnet",
			Description: "Production Ethereum network",
			ChainID:     1,
			GenesisTime: 1606824023,
			ServiceURLs: map[string]string{
				"ethstats":    "https://ethstats.mainnet.ethpandaops.io",
				"forkmon":     "https://forkmon.mainnet.ethpandaops.io",
				"blobArchive": "https://blob-archive.mainnet.ethpandaops.io",
				"forky":       "https://forky.mainnet.ethpandaops.io",
				"tracoor":     "https://tracoor.mainnet.ethpandaops.io",
			},
		},
		{
			Name:        "sepolia",
			Description: "Smaller testnet for application development with controlled validator set.",
			ChainID:     11155111,
			GenesisTime: 1655726400,
			ServiceURLs: map[string]string{
				"dora":           "https://dora.sepolia.ethpandaops.io",
				"beaconExplorer": "https://dora.sepolia.ethpandaops.io",
				"checkpointSync": "https://checkpoint-sync.sepolia.ethpandaops.io",
				"ethstats":       "https://ethstats.sepolia.ethpandaops.io",
				"forkmon":        "https://forkmon.sepolia.ethpandaops.io",
				"blobArchive":    "https://blob-archive.sepolia.ethpandaops.io",
				"forky":          "https://forky.sepolia.ethpandaops.io",
				"tracoor":        "https://tracoor.sepolia.ethpandaops.io",
			},
		},
		{
			Name:        "hoodi",
			Description: "New public testnet (launched March 2025) designed for validator testing and protocol upgrades, replacing Holesky.",
			ChainID:     560048,
			GenesisTime: 1742213400,
			ServiceURLs: map[string]string{
				"dora":           "https://dora.hoodi.ethpandaops.io",
				"beaconExplorer": "https://dora.hoodi.ethpandaops.io",
				"checkpointSync": "https://checkpoint-sync.hoodi.ethpandaops.io",
				"forkmon":        "https://forkmon.hoodi.ethpandaops.io",
				"forky":          "https://forky.hoodi.ethpandaops.io",
				"tracoor":        "https://tracoor.hoodi.ethpandaops.io",
			},
		},
	}

	networks, err := provider.Discover(context.Background(), config)
	require.NoError(t, err)

	// Verify we got the expected networks
	assert.Len(t, networks, 3)
	assert.Contains(t, networks, "mainnet")
	assert.Contains(t, networks, "sepolia")
	assert.Contains(t, networks, "hoodi")

	// Verify mainnet network properties
	mainnet := networks["mainnet"]
	assert.Equal(t, "mainnet", mainnet.Name)
	assert.Equal(t, "Production Ethereum network", mainnet.Description)
	assert.WithinDuration(t, time.Now(), mainnet.LastUpdated, 10*time.Second)
	require.NotNil(t, mainnet.ServiceURLs)
	assert.Equal(t, "https://ethstats.mainnet.ethpandaops.io", mainnet.ServiceURLs.Ethstats)
	assert.Equal(t, "https://forkmon.mainnet.ethpandaops.io", mainnet.ServiceURLs.Forkmon)
	assert.Equal(t, "https://blob-archive.mainnet.ethpandaops.io", mainnet.ServiceURLs.BlobArchive)
	assert.Equal(t, "https://forky.mainnet.ethpandaops.io", mainnet.ServiceURLs.Forky)
	assert.Equal(t, "https://tracoor.mainnet.ethpandaops.io", mainnet.ServiceURLs.Tracoor)

	// Verify sepolia network properties
	sepolia := networks["sepolia"]
	assert.Equal(t, "sepolia", sepolia.Name)
	assert.Equal(t, "Smaller testnet for application development with controlled validator set.", sepolia.Description)
	assert.WithinDuration(t, time.Now(), sepolia.LastUpdated, 10*time.Second)
	require.NotNil(t, sepolia.ServiceURLs)
	assert.Equal(t, "https://dora.sepolia.ethpandaops.io", sepolia.ServiceURLs.Dora)
	assert.Equal(t, "https://dora.sepolia.ethpandaops.io", sepolia.ServiceURLs.BeaconExplorer)
	assert.Equal(t, "https://checkpoint-sync.sepolia.ethpandaops.io", sepolia.ServiceURLs.CheckpointSync)
	assert.Equal(t, "https://ethstats.sepolia.ethpandaops.io", sepolia.ServiceURLs.Ethstats)

	// Verify hoodi network properties
	hoodi := networks["hoodi"]
	assert.Equal(t, "hoodi", hoodi.Name)
	assert.Equal(t, "New public testnet (launched March 2025) designed for validator testing and protocol upgrades, replacing Holesky.", hoodi.Description)
	assert.WithinDuration(t, time.Now(), hoodi.LastUpdated, 10*time.Second)
	require.NotNil(t, hoodi.ServiceURLs)
	assert.Equal(t, "https://dora.hoodi.ethpandaops.io", hoodi.ServiceURLs.Dora)
	assert.Equal(t, "https://dora.hoodi.ethpandaops.io", hoodi.ServiceURLs.BeaconExplorer)
	assert.Equal(t, "https://checkpoint-sync.hoodi.ethpandaops.io", hoodi.ServiceURLs.CheckpointSync)
}
