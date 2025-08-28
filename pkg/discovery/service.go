package discovery

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// ClientDiscovererInterface defines the interface for client discovery.
type ClientDiscovererInterface interface {
	DiscoverClients(ctx context.Context) (map[string]ClientInfo, error)
}

// Service handles the discovery of networks.
type Service struct {
	log              *logrus.Logger
	config           Config
	providers        []Provider
	resultChan       chan Result
	resultFuncs      []ResultHandler
	ticker           *time.Ticker
	wg               sync.WaitGroup
	mutex            sync.Mutex
	clientDiscoverer ClientDiscovererInterface
}

// NewService creates a new discovery service.
func NewService(log *logrus.Logger, cfg Config) (*Service, error) {
	log = log.WithField("module", "discovery").Logger

	// Set default values if not specified
	if cfg.Interval == 0 {
		cfg.Interval = 1 * time.Hour
	}

	return &Service{
		log:              log,
		config:           cfg,
		providers:        []Provider{},
		resultChan:       make(chan Result, 10),
		resultFuncs:      []ResultHandler{},
		clientDiscoverer: NewClientDiscoverer(log, cfg.GitHub.Token),
	}, nil
}

// RegisterProvider registers a provider with the discovery service.
func (s *Service) RegisterProvider(provider Provider) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.providers = append(s.providers, provider)
	s.log.WithField("provider", provider.Name()).Info("Registered discovery provider")
}

// OnResult registers a function to be called when a discovery result is available.
func (s *Service) OnResult(fn ResultHandler) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.resultFuncs = append(s.resultFuncs, fn)
}

// Start starts the discovery service.
func (s *Service) Start(ctx context.Context) error {
	s.log.WithField("interval", s.config.Interval).Info("Starting discovery service")

	// Start the result processor
	s.wg.Add(1)

	go func() {
		defer s.wg.Done()

		for {
			select {
			case <-ctx.Done():
				return
			case result := <-s.resultChan:
				s.mutex.Lock()
				for _, fn := range s.resultFuncs {
					fn(result)
				}
				s.mutex.Unlock()
			}
		}
	}()

	// Start the ticker
	s.ticker = time.NewTicker(s.config.Interval)
	s.wg.Add(1)

	go func() {
		defer s.wg.Done()

		// Run an initial discovery
		if err := s.runDiscovery(ctx); err != nil {
			s.log.WithError(err).Error("Failed to run initial discovery")
		}

		for {
			select {
			case <-ctx.Done():
				return
			case <-s.ticker.C:
				if err := s.runDiscovery(ctx); err != nil {
					s.log.WithError(err).Error("Failed to run discovery")
				}
			}
		}
	}()

	return nil
}

// Stop stops the discovery service.
func (s *Service) Stop(ctx context.Context) error {
	s.log.Info("Stopping discovery service")

	if s.ticker != nil {
		s.ticker.Stop()
	}

	done := make(chan struct{})

	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// RunOnce executes a single discovery run and returns the result directly.
func (s *Service) RunOnce(ctx context.Context) (Result, error) {
	result, err := s.executeDiscovery(ctx)
	if err != nil {
		return Result{
			Networks:        make(map[string]Network),
			NetworkMetadata: make(map[string]RepositoryMetadata),
			Clients:         make(map[string]ClientInfo),
		}, err
	}

	return result, nil
}

// runDiscovery runs the discovery process and sends the result to the result channel.
func (s *Service) runDiscovery(ctx context.Context) error {
	result, err := s.executeDiscovery(ctx)
	if err != nil {
		return err
	}

	// Send result to channel
	select {
	case s.resultChan <- result:
	default:
		s.log.Warn("Result channel full, discarding result")
	}

	return nil
}

// executeDiscovery performs the actual discovery process and returns the result.
func (s *Service) executeDiscovery(ctx context.Context) (Result, error) {
	start := time.Now()

	s.log.Info("Running discovery")

	s.mutex.Lock()
	providers := s.providers
	s.mutex.Unlock()

	if len(providers) == 0 {
		s.log.Warn("No discovery providers registered")

		return Result{
			Networks:        make(map[string]Network),
			NetworkMetadata: make(map[string]RepositoryMetadata),
			Clients:         make(map[string]ClientInfo),
		}, nil
	}

	type providerResult struct {
		networks map[string]Network
		provider Provider
		err      error
	}

	// Channel to collect results from provider goroutines
	resultCh := make(chan providerResult, len(providers))

	// Run discovery for each provider
	for _, provider := range providers {
		go func(p Provider) {
			pLog := s.log.WithField("provider", p.Name())
			pLog.Info("Running discovery provider")

			networkMap, err := p.Discover(ctx, s.config)
			if err != nil {
				pLog.WithError(err).Error("Failed to discover networks")

				resultCh <- providerResult{
					networks: nil,
					provider: p,
					err:      err,
				}

				return
			}

			pLog.WithField("networks", len(networkMap)).Info("Discovery complete")

			resultCh <- providerResult{
				networks: networkMap,
				provider: p,
				err:      nil,
			}
		}(provider)
	}

	// Collect results
	var (
		allNetworks = make(map[string]Network)
		provInfos   = make([]ProviderInfo, 0)
	)

	// Wait for all provider goroutines to complete
	for i := 0; i < len(providers); i++ {
		select {
		case <-ctx.Done():
			return Result{
				Networks:        allNetworks,
				NetworkMetadata: make(map[string]RepositoryMetadata),
				Clients:         make(map[string]ClientInfo),
			}, ctx.Err()
		case pr := <-resultCh:
			if pr.err == nil && pr.networks != nil {
				// Merge networks, newer ones will overwrite older ones with the same key
				for key, network := range pr.networks {
					allNetworks[key] = network
				}

				provInfos = append(provInfos, ProviderInfo{Name: pr.provider.Name()})
			}
		}
	}

	// Build repository metadata from config
	networkMetadata := buildNetworkMetadata(s.config, allNetworks)

	// Discover client information
	clientInfo, err := s.clientDiscoverer.DiscoverClients(ctx)
	if err != nil {
		s.log.WithError(err).Warn("Failed to discover client information")

		clientInfo = make(map[string]ClientInfo)
	}

	// Create result
	duration := time.Since(start).Seconds()
	result := Result{
		Networks:        allNetworks,
		NetworkMetadata: networkMetadata,
		Clients:         clientInfo,
		LastUpdate:      time.Now(),
		Duration:        duration,
		Providers:       provInfos,
	}

	s.log.WithFields(logrus.Fields{
		"networks":         len(allNetworks),
		"network_metadata": len(networkMetadata),
		"clients":          len(clientInfo),
		"duration":         duration,
	}).Info("Discovery complete")

	return result, nil
}

// buildNetworkMetadata builds the network metadata from GitHub repository configurations.
func buildNetworkMetadata(config Config, networks map[string]Network) map[string]RepositoryMetadata {
	metadata := make(map[string]RepositoryMetadata)

	// First pass: create metadata entries for each repository
	for _, repo := range config.GitHub.Repositories {
		// Use the repository name prefix (e.g., "eof-") as the key in network_metadata
		// If there's no prefix, use the repo name
		metadataKey := repo.NamePrefix
		if metadataKey == "" {
			// Extract the repo name from the full path
			parts := strings.Split(repo.Name, "/")
			if len(parts) == 2 {
				metadataKey = parts[1]
			} else {
				metadataKey = repo.Name
			}
		} else {
			// Remove trailing dash if present to get a clean key
			metadataKey = strings.TrimSuffix(metadataKey, "-")
		}

		metadata[metadataKey] = RepositoryMetadata{
			DisplayName: repo.DisplayName,
			Description: repo.Description,
			Links:       repo.Links,
			Image:       repo.Image,
			Stats: Stats{
				TotalNetworks:    0,
				ActiveNetworks:   0,
				InactiveNetworks: 0,
				NetworkNames:     []string{},
			},
		}
	}

	// Second pass: gather network statistics
	repoNetworks := make(map[string][]Network)

	// Group networks by repository
	for netName, network := range networks {
		if network.Repository == "" {
			continue
		}

		// Extract repository name prefix from network name
		var metadataKey string

		for _, repo := range config.GitHub.Repositories {
			prefix := repo.NamePrefix
			if prefix != "" && strings.HasPrefix(netName, prefix) {
				metadataKey = strings.TrimSuffix(prefix, "-")

				break
			}

			// If no matching prefix found, try to extract from repository path
			if repo.Name == network.Repository {
				parts := strings.Split(repo.Name, "/")
				if len(parts) == 2 {
					metadataKey = parts[1]
				} else {
					metadataKey = repo.Name
				}

				break
			}
		}

		if metadataKey != "" {
			if _, exists := repoNetworks[metadataKey]; !exists {
				repoNetworks[metadataKey] = []Network{}
			}

			repoNetworks[metadataKey] = append(repoNetworks[metadataKey], network)
		}
	}

	// Calculate statistics for each repository
	for repoKey, nets := range repoNetworks {
		meta, exists := metadata[repoKey]
		if !exists {
			continue
		}

		// Count networks
		meta.Stats.TotalNetworks = len(nets)
		meta.Stats.ActiveNetworks = 0
		meta.Stats.InactiveNetworks = 0

		for _, net := range nets {
			// Collect network name
			meta.Stats.NetworkNames = append(meta.Stats.NetworkNames, net.Name)

			// Count by status
			if net.Status == "active" {
				meta.Stats.ActiveNetworks++
			} else {
				meta.Stats.InactiveNetworks++
			}
		}

		// Update the metadata
		metadata[repoKey] = meta
	}

	return metadata
}
