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

// runDiscovery runs the discovery process.
func (s *Service) runDiscovery(ctx context.Context) error {
	start := time.Now()
	s.log.Info("Running discovery")

	s.mutex.Lock()
	providers := s.providers
	s.mutex.Unlock()

	if len(providers) == 0 {
		s.log.Warn("No discovery providers registered")
		return nil
	}

	var (
		networks  []Network
		provNames []Provider
		wg        sync.WaitGroup
		mu        sync.Mutex
	)

	// Run discovery for each provider
	for _, provider := range providers {
		wg.Add(1)
		go func(p Provider) {
			defer wg.Done()

			pLog := s.log.WithField("provider", p.Name())
			pLog.Info("Running discovery provider")

			ns, err := p.Discover(ctx, s.config)
			if err != nil {
				pLog.WithError(err).Error("Failed to discover networks")
				return
			}

			pLog.WithField("networks", len(ns)).Info("Discovery complete")

			mu.Lock()
			networks = append(networks, ns...)
			provNames = append(provNames, p)
			mu.Unlock()
		}(provider)
	}

	wg.Wait()

	// Create result
	duration := time.Since(start).Seconds()
	result := Result{
		Networks:   networks,
		LastUpdate: time.Now(),
		Duration:   duration,
		Providers:  provNames,
	}

	s.log.WithFields(logrus.Fields{
		"networks": len(networks),
		"duration": duration,
	}).Info("Discovery complete")

	// Send result to channel
	select {
	case s.resultChan <- result:
	default:
		s.log.Warn("Result channel full, discarding result")
	}

	return nil
}