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
var _ Provider = (*MemoryProvider)(nil)

// MemoryProvider is an in-memory implementation of Provider.
// It periodically fetches data from the cartographoor endpoint and stores it in memory.
type MemoryProvider struct {
	config     Config
	log        logrus.FieldLogger
	mu         sync.RWMutex
	networks   map[string]discovery.Network
	clients    map[string]discovery.ClientInfo
	ready      bool
	ticker     *time.Ticker
	done       chan struct{}
	notifyChan chan struct{}
	wg         sync.WaitGroup
}

// NewMemoryProvider creates a new in-memory provider.
func NewMemoryProvider(config Config, log logrus.FieldLogger) (*MemoryProvider, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	if log == nil {
		log = logrus.New()
	}

	return &MemoryProvider{
		config:     config,
		log:        log.WithField("component", "cartographoor_memory"),
		networks:   make(map[string]discovery.Network),
		clients:    make(map[string]discovery.ClientInfo),
		done:       make(chan struct{}),
		notifyChan: make(chan struct{}, 1),
	}, nil
}

// Start initializes the provider and begins periodic refresh.
func (m *MemoryProvider) Start(ctx context.Context) error {
	m.log.Info("Starting memory provider")

	// Initial fetch (blocking)
	if err := m.refresh(ctx); err != nil {
		return fmt.Errorf("initial fetch failed: %w", err)
	}

	// Mark as ready
	m.mu.Lock()
	m.ready = true
	m.mu.Unlock()

	// Start background refresh loop
	m.ticker = time.NewTicker(m.config.RefreshInterval)
	m.wg.Add(1)

	go func() {
		defer m.wg.Done()

		for {
			select {
			case <-ctx.Done():
				return
			case <-m.done:
				return
			case <-m.ticker.C:
				if err := m.refresh(ctx); err != nil {
					m.log.WithError(err).Error("Failed to refresh data")
				}
			}
		}
	}()

	m.log.Info("Memory provider started")

	return nil
}

// Stop gracefully stops the provider.
func (m *MemoryProvider) Stop() error {
	m.log.Info("Stopping memory provider")

	if m.ticker != nil {
		m.ticker.Stop()
	}

	close(m.done)
	m.wg.Wait()

	m.log.Info("Memory provider stopped")

	return nil
}

// Ready returns true if the provider has loaded data.
func (m *MemoryProvider) Ready() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.ready
}

// GetNetworks returns all known networks.
func (m *MemoryProvider) GetNetworks(ctx context.Context) (map[string]discovery.Network, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.ready {
		return nil, fmt.Errorf("provider not ready")
	}

	// Return copy to prevent external modification
	result := make(map[string]discovery.Network, len(m.networks))
	for k, v := range m.networks {
		result[k] = v
	}

	return result, nil
}

// GetNetwork returns a specific network by name.
func (m *MemoryProvider) GetNetwork(ctx context.Context, name string) (discovery.Network, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.ready {
		return discovery.Network{}, false, fmt.Errorf("provider not ready")
	}

	network, ok := m.networks[name]

	return network, ok, nil
}

// GetActiveNetworks returns only active networks.
func (m *MemoryProvider) GetActiveNetworks(ctx context.Context) (map[string]discovery.Network, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.ready {
		return nil, fmt.Errorf("provider not ready")
	}

	result := make(map[string]discovery.Network)

	for k, v := range m.networks {
		if v.Status == "active" {
			result[k] = v
		}
	}

	return result, nil
}

// GetClients returns all known clients.
func (m *MemoryProvider) GetClients(ctx context.Context) (map[string]discovery.ClientInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.ready {
		return nil, fmt.Errorf("provider not ready")
	}

	// Return copy
	result := make(map[string]discovery.ClientInfo, len(m.clients))
	for k, v := range m.clients {
		result[k] = v
	}

	return result, nil
}

// GetClient returns a specific client by name.
func (m *MemoryProvider) GetClient(ctx context.Context, name string) (discovery.ClientInfo, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.ready {
		return discovery.ClientInfo{}, false, fmt.Errorf("provider not ready")
	}

	client, ok := m.clients[name]

	return client, ok, nil
}

// GetClientsByType returns clients filtered by type.
func (m *MemoryProvider) GetClientsByType(ctx context.Context, clientType string) (map[string]discovery.ClientInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.ready {
		return nil, fmt.Errorf("provider not ready")
	}

	result := make(map[string]discovery.ClientInfo)
	normalizedType := strings.ToLower(clientType)

	for k, v := range m.clients {
		if strings.ToLower(v.Type) == normalizedType {
			result[k] = v
		}
	}

	return result, nil
}

// NotifyChannel returns the notification channel.
func (m *MemoryProvider) NotifyChannel() <-chan struct{} {
	return m.notifyChan
}

// refresh fetches new data from the source and updates internal state.
func (m *MemoryProvider) refresh(ctx context.Context) error {
	m.log.Debug("Fetching cartographoor data")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.config.SourceURL, http.NoBody)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "cartographoor-client")

	resp, err := m.config.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetch data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))

		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var result discovery.Result
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	// Update state
	m.mu.Lock()
	m.networks = result.Networks
	m.clients = result.Clients
	m.mu.Unlock()

	// Send non-blocking notification
	select {
	case m.notifyChan <- struct{}{}:
		m.log.Debug("Sent update notification")
	default:
		// Channel already has pending notification
	}

	m.log.WithFields(logrus.Fields{
		"networks": len(result.Networks),
		"clients":  len(result.Clients),
	}).Info("Refreshed cartographoor data")

	return nil
}
