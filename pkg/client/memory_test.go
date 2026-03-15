package client

import (
	"context"
	"testing"
	"time"

	"github.com/ethpandaops/cartographoor/pkg/discovery"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryProvider(t *testing.T) {
	config := Config{
		SourceURL:       "https://ethpandaops-platform-production-cartographoor.ams3.cdn.digitaloceanspaces.com/networks.json",
		RefreshInterval: 1 * time.Minute,
		RequestTimeout:  30 * time.Second,
	}

	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	provider, err := NewMemoryProvider(config, log)
	require.NoError(t, err)
	require.NotNil(t, provider)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Start provider
	err = provider.Start(ctx)
	require.NoError(t, err)

	defer func() {
		assert.NoError(t, provider.Stop())
	}()

	// Check ready
	assert.True(t, provider.Ready())

	// Get networks
	networks, err := provider.GetNetworks(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, networks)

	// Get active networks
	activeNetworks, err := provider.GetActiveNetworks(ctx)
	require.NoError(t, err)
	t.Logf("Found %d active networks", len(activeNetworks))

	// Get clients
	clients, err := provider.GetClients(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, clients)

	// Get consensus clients
	consensusClients, err := provider.GetClientsByType(ctx, "consensus")
	require.NoError(t, err)
	assert.NotEmpty(t, consensusClients)

	// Verify notification channel
	notifyCh := provider.NotifyChannel()
	assert.NotNil(t, notifyCh)
}

func TestMemoryProviderFiltering(t *testing.T) {
	provider := &MemoryProvider{
		log: logrus.New(),
		networks: map[string]discovery.Network{
			"active-net":   {Name: "active-net", Status: "active"},
			"inactive-net": {Name: "inactive-net", Status: "inactive"},
		},
		clients: map[string]discovery.ClientInfo{
			"lighthouse": {Name: "lighthouse", Type: "consensus"},
			"geth":       {Name: "geth", Type: "execution"},
		},
		ready: true,
	}

	ctx := context.Background()

	// Test GetActiveNetworks
	active, err := provider.GetActiveNetworks(ctx)
	require.NoError(t, err)
	assert.Len(t, active, 1)
	assert.Contains(t, active, "active-net")

	// Test GetClientsByType
	consensus, err := provider.GetClientsByType(ctx, "consensus")
	require.NoError(t, err)
	assert.Len(t, consensus, 1)
	assert.Contains(t, consensus, "lighthouse")

	execution, err := provider.GetClientsByType(ctx, "execution")
	require.NoError(t, err)
	assert.Len(t, execution, 1)
	assert.Contains(t, execution, "geth")
}
