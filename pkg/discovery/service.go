package discovery

import (
	"context"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// Service handles the discovery of networks.
type Service struct {
	log         *logrus.Logger
	config      Config
	providers   []Provider
	resultChan  chan Result
	resultFuncs []ResultHandler
	ticker      *time.Ticker
	wg          sync.WaitGroup
	mutex       sync.Mutex
}

// NewService creates a new discovery service.
func NewService(log *logrus.Logger, cfg Config) (*Service, error) {
	log = log.WithField("module", "discovery").Logger

	// Set default values if not specified
	if cfg.Interval == 0 {
		cfg.Interval = 1 * time.Hour
	}

	return &Service{
		log:         log,
		config:      cfg,
		providers:   []Provider{},
		resultChan:  make(chan Result, 10),
		resultFuncs: []ResultHandler{},
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
		return Result{}, err
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
			Networks: make(map[string]Network),
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
		provNames   = make([]Provider, 0)
	)

	// Wait for all provider goroutines to complete
	for i := 0; i < len(providers); i++ {
		select {
		case <-ctx.Done():
			return Result{Networks: allNetworks}, ctx.Err()
		case pr := <-resultCh:
			if pr.err == nil && pr.networks != nil {
				// Merge networks, newer ones will overwrite older ones with the same key
				for key, network := range pr.networks {
					allNetworks[key] = network
				}

				provNames = append(provNames, pr.provider)
			}
		}
	}

	// Create result
	duration := time.Since(start).Seconds()
	result := Result{
		Networks:   allNetworks,
		LastUpdate: time.Now(),
		Duration:   duration,
		Providers:  provNames,
	}

	s.log.WithFields(logrus.Fields{
		"networks": len(allNetworks),
		"duration": duration,
	}).Info("Discovery complete")

	return result, nil
}
