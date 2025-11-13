package client

//go:generate mockgen -package mocks -destination mocks/mock_provider.go github.com/ethpandaops/cartographoor/pkg/client Provider
//go:generate mockgen -package mocks -destination mocks/mock_redis.go github.com/ethpandaops/cartographoor/pkg/client RedisClient,LeaderElector

import (
	"context"

	"github.com/ethpandaops/cartographoor/pkg/discovery"
)

// Provider defines the interface for accessing cartographoor network and client data.
// It handles fetching, caching, and providing access to network metadata.
type Provider interface {
	// Start initializes the provider and begins periodic data refresh.
	// Blocks until initial data is loaded or context is cancelled.
	Start(ctx context.Context) error

	// Stop gracefully stops the provider and cleans up resources.
	Stop() error

	// Ready returns true if the provider has successfully loaded data.
	Ready() bool

	// GetNetworks returns all known networks.
	GetNetworks(ctx context.Context) (map[string]discovery.Network, error)

	// GetNetwork returns a specific network by name.
	// Returns empty Network if not found.
	GetNetwork(ctx context.Context, name string) (discovery.Network, bool, error)

	// GetActiveNetworks returns only networks with status "active".
	GetActiveNetworks(ctx context.Context) (map[string]discovery.Network, error)

	// GetClients returns all known Ethereum clients.
	GetClients(ctx context.Context) (map[string]discovery.ClientInfo, error)

	// GetClient returns a specific client by name.
	// Returns empty ClientInfo if not found.
	GetClient(ctx context.Context, name string) (discovery.ClientInfo, bool, error)

	// GetClientsByType returns clients filtered by type ("consensus" or "execution").
	GetClientsByType(ctx context.Context, clientType string) (map[string]discovery.ClientInfo, error)

	// NotifyChannel returns a channel that receives notifications when data is updated.
	// The channel is buffered with size 1 to prevent blocking the refresh loop.
	NotifyChannel() <-chan struct{}
}

// RedisClient defines the interface for Redis operations needed by RedisProvider.
type RedisClient interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string, ttl int) error
}

// LeaderElector defines the interface for leader election needed by RedisProvider.
type LeaderElector interface {
	IsLeader() bool
}
