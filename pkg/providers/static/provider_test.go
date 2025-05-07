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
		Status:      "active",
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

	// Create discovery service
	service, err := discovery.NewService(log, discovery.Config{})
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
	assert.Len(t, result.Networks, 5, "Should have 5 networks (5 from static, 1 from GitHub mock)")
	assert.Contains(t, result.Networks, "mainnet")
	assert.Contains(t, result.Networks, "sepolia")
	assert.Contains(t, result.Networks, "hoodi")
	assert.Contains(t, result.Networks, "holesky")
	assert.Contains(t, result.Networks, "devnet-1")

	// Verify the providers in the result
	providerNames := make([]string, 0, len(result.Providers))
	for _, p := range result.Providers {
		providerNames = append(providerNames, p.Name())
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

func TestProvider_Discover(t *testing.T) {
	log := logrus.New()
	provider, err := NewProvider(log)
	require.NoError(t, err)

	networks, err := provider.Discover(context.Background(), discovery.Config{})
	require.NoError(t, err)

	// Verify we got the expected networks
	assert.Len(t, networks, 4)
	assert.Contains(t, networks, "mainnet")
	assert.Contains(t, networks, "sepolia")
	assert.Contains(t, networks, "hoodi")

	// Verify mainnet network properties
	mainnet := networks["mainnet"]
	assert.Equal(t, "mainnet", mainnet.Name)
	assert.Equal(t, "Ethereum Mainnet", mainnet.Description)
	assert.Equal(t, "active", mainnet.Status)
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
	assert.Equal(t, "Sepolia Testnet", sepolia.Description)
	assert.Equal(t, "active", sepolia.Status)
	assert.WithinDuration(t, time.Now(), sepolia.LastUpdated, 10*time.Second)
	require.NotNil(t, sepolia.ServiceURLs)
	assert.Equal(t, "https://dora.sepolia.ethpandaops.io", sepolia.ServiceURLs.Dora)
	assert.Equal(t, "https://dora.sepolia.ethpandaops.io", sepolia.ServiceURLs.BeaconExplorer)
	assert.Equal(t, "https://checkpoint-sync.sepolia.ethpandaops.io", sepolia.ServiceURLs.CheckpointSync)
	assert.Equal(t, "https://ethstats.sepolia.ethpandaops.io", sepolia.ServiceURLs.Ethstats)

	// Verify hoodi network properties
	hoodi := networks["hoodi"]
	assert.Equal(t, "hoodi", hoodi.Name)
	assert.Equal(t, "Hoodi Testnet", hoodi.Description)
	assert.Equal(t, "active", hoodi.Status)
	assert.WithinDuration(t, time.Now(), hoodi.LastUpdated, 10*time.Second)
	require.NotNil(t, hoodi.ServiceURLs)
	assert.Equal(t, "https://dora.hoodi.ethpandaops.io", hoodi.ServiceURLs.Dora)
	assert.Equal(t, "https://dora.hoodi.ethpandaops.io", hoodi.ServiceURLs.BeaconExplorer)
	assert.Equal(t, "https://checkpoint-sync.hoodi.ethpandaops.io", hoodi.ServiceURLs.CheckpointSync)
}
