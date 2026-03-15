package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ethpandaops/cartographoor/pkg/discovery"
	"github.com/sirupsen/logrus"
)

// Compile-time interface check.
var _ Provider = (*RedisProvider)(nil)

const (
	redisNetworksKey = "cartographoor:networks"
	redisClientsKey  = "cartographoor:clients"
)

// RedisProvider is a Redis-backed implementation of Provider.
// It uses leader election to coordinate writes across multiple instances.
type RedisProvider struct {
	config   Config
	log      logrus.FieldLogger
	redis    RedisClient
	elector  LeaderElector
	readyMu  sync.RWMutex
	ready    bool
	ticker   *time.Ticker
	done     chan struct{}
	notifyCh chan struct{}
	wg       sync.WaitGroup
}

// NewRedisProvider creates a new Redis-backed provider.
func NewRedisProvider(
	config Config,
	redis RedisClient,
	elector LeaderElector,
	log logrus.FieldLogger,
) (*RedisProvider, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	if redis == nil {
		return nil, fmt.Errorf("redis client required")
	}

	if elector == nil {
		return nil, fmt.Errorf("leader elector required")
	}

	if log == nil {
		log = logrus.New()
	}

	return &RedisProvider{
		config:   config,
		log:      log.WithField("component", "cartographoor_redis"),
		redis:    redis,
		elector:  elector,
		done:     make(chan struct{}),
		notifyCh: make(chan struct{}, 1),
	}, nil
}

// Start initializes the provider and begins periodic refresh (leader only).
func (r *RedisProvider) Start(ctx context.Context) error {
	r.log.Info("Starting Redis provider")

	// Start refresh loop (leader only)
	r.ticker = time.NewTicker(r.config.RefreshInterval)
	r.wg.Add(1)

	go func() {
		defer r.wg.Done()

		// Immediate refresh if leader
		if r.elector.IsLeader() {
			if err := r.refresh(ctx); err != nil {
				r.log.WithError(err).Error("Initial refresh failed")
			}
		}

		for {
			select {
			case <-ctx.Done():
				return
			case <-r.done:
				return
			case <-r.ticker.C:
				if r.elector.IsLeader() {
					if err := r.refresh(ctx); err != nil {
						r.log.WithError(err).Error("Failed to refresh data")
					}
				}
			}
		}
	}()

	// Wait for initial data
	readyCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-readyCtx.Done():
			return fmt.Errorf("timeout waiting for initial data: %w", readyCtx.Err())
		case <-ticker.C:
			_, err := r.redis.Get(ctx, redisNetworksKey)
			if err == nil {
				r.setReady(true)
				r.log.Info("Redis provider started and ready")

				return nil
			}
		}
	}
}

// Stop gracefully stops the provider.
func (r *RedisProvider) Stop() error {
	r.log.Info("Stopping Redis provider")

	if r.ticker != nil {
		r.ticker.Stop()
	}

	close(r.done)
	r.wg.Wait()

	r.log.Info("Redis provider stopped")

	return nil
}

// Ready returns true if Redis has data available.
func (r *RedisProvider) Ready() bool {
	r.readyMu.RLock()
	defer r.readyMu.RUnlock()

	return r.ready
}

// GetNetworks returns all networks from Redis.
func (r *RedisProvider) GetNetworks(ctx context.Context) (map[string]discovery.Network, error) {
	data, err := r.redis.Get(ctx, redisNetworksKey)
	if err != nil {
		r.log.WithError(err).Debug("No networks in Redis")

		return make(map[string]discovery.Network), fmt.Errorf("get networks from redis: %w", err)
	}

	var networks map[string]discovery.Network
	if unmarshalErr := json.Unmarshal([]byte(data), &networks); unmarshalErr != nil {
		r.log.WithError(unmarshalErr).Error("Failed to unmarshal networks")

		return make(map[string]discovery.Network), fmt.Errorf("unmarshal networks: %w", unmarshalErr)
	}

	return networks, nil
}

// GetNetwork returns a specific network from Redis.
func (r *RedisProvider) GetNetwork(ctx context.Context, name string) (discovery.Network, bool, error) {
	networks, err := r.GetNetworks(ctx)
	if err != nil {
		return discovery.Network{}, false, err
	}

	network, ok := networks[name]

	return network, ok, nil
}

// GetActiveNetworks returns only active networks from Redis.
func (r *RedisProvider) GetActiveNetworks(ctx context.Context) (map[string]discovery.Network, error) {
	networks, err := r.GetNetworks(ctx)
	if err != nil {
		return nil, err
	}

	result := make(map[string]discovery.Network)

	for k, v := range networks {
		if v.Status == "active" {
			result[k] = v
		}
	}

	return result, nil
}

// GetClients returns all clients from Redis.
func (r *RedisProvider) GetClients(ctx context.Context) (map[string]discovery.ClientInfo, error) {
	data, err := r.redis.Get(ctx, redisClientsKey)
	if err != nil {
		r.log.WithError(err).Debug("No clients in Redis")

		return make(map[string]discovery.ClientInfo), fmt.Errorf("get clients from redis: %w", err)
	}

	var clients map[string]discovery.ClientInfo
	if unmarshalErr := json.Unmarshal([]byte(data), &clients); unmarshalErr != nil {
		r.log.WithError(unmarshalErr).Error("Failed to unmarshal clients")

		return make(map[string]discovery.ClientInfo), fmt.Errorf("unmarshal clients: %w", unmarshalErr)
	}

	return clients, nil
}

// GetClient returns a specific client from Redis.
func (r *RedisProvider) GetClient(ctx context.Context, name string) (discovery.ClientInfo, bool, error) {
	clients, err := r.GetClients(ctx)
	if err != nil {
		return discovery.ClientInfo{}, false, err
	}

	client, ok := clients[name]

	return client, ok, nil
}

// GetClientsByType returns clients filtered by type from Redis.
func (r *RedisProvider) GetClientsByType(ctx context.Context, clientType string) (map[string]discovery.ClientInfo, error) {
	clients, err := r.GetClients(ctx)
	if err != nil {
		return nil, err
	}

	result := make(map[string]discovery.ClientInfo)
	normalizedType := strings.ToLower(clientType)

	for k, v := range clients {
		if strings.ToLower(v.Type) == normalizedType {
			result[k] = v
		}
	}

	return result, nil
}

// NotifyChannel returns the notification channel.
func (r *RedisProvider) NotifyChannel() <-chan struct{} {
	return r.notifyCh
}

// setReady updates the ready state.
func (r *RedisProvider) setReady(ready bool) {
	r.readyMu.Lock()
	defer r.readyMu.Unlock()

	r.ready = ready
}

// refresh fetches new data and stores it in Redis (leader only).
func (r *RedisProvider) refresh(ctx context.Context) error {
	if !r.elector.IsLeader() {
		return nil
	}

	r.log.Debug("Leader refreshing cartographoor data")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.config.SourceURL, http.NoBody)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "cartographoor-client")

	resp, err := r.config.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetch data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))

		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var result discovery.Result
	if decodeErr := json.NewDecoder(resp.Body).Decode(&result); decodeErr != nil {
		return fmt.Errorf("decode response: %w", decodeErr)
	}

	// Store networks in Redis
	networksJSON, err := json.Marshal(result.Networks)
	if err != nil {
		return fmt.Errorf("marshal networks: %w", err)
	}

	if setErr := r.redis.Set(ctx, redisNetworksKey, string(networksJSON), 600); setErr != nil {
		return fmt.Errorf("store networks: %w", setErr)
	}

	// Store clients in Redis
	clientsJSON, err := json.Marshal(result.Clients)
	if err != nil {
		return fmt.Errorf("marshal clients: %w", err)
	}

	if err := r.redis.Set(ctx, redisClientsKey, string(clientsJSON), 600); err != nil {
		return fmt.Errorf("store clients: %w", err)
	}

	// Send non-blocking notification
	select {
	case r.notifyCh <- struct{}{}:
		r.log.Debug("Sent update notification")
	default:
	}

	r.log.WithFields(logrus.Fields{
		"networks": len(result.Networks),
		"clients":  len(result.Clients),
	}).Info("Leader refreshed cartographoor data")

	return nil
}
