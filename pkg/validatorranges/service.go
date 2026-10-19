package validatorranges

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ethpandaops/cartographoor/pkg/discovery"
	"github.com/ethpandaops/cartographoor/pkg/storage/s3"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/semaphore"
)

// Service handles the generation and upload of validator ranges.
type Service struct {
	fetcher   *Fetcher
	s3Storage *s3.Provider
	config    *Config
	logger    *logrus.Entry
}

// NewService creates a new validator ranges service.
func NewService(s3Storage *s3.Provider, config *Config, logger *logrus.Logger) *Service {
	return &Service{
		fetcher:   NewFetcher(),
		s3Storage: s3Storage,
		config:    config,
		logger:    logger.WithField("module", "validator_ranges"),
	}
}

// GenerateValidatorRanges processes all networks to generate validator range data.
func (s *Service) GenerateValidatorRanges(ctx context.Context, networks map[string]discovery.Network) error {
	s.logger.WithField("networks", len(networks)).Info("Generating validator ranges for networks")

	// Use semaphore to limit concurrency to 5 networks at a time
	sem := semaphore.NewWeighted(5)

	for name, network := range networks {
		if err := sem.Acquire(ctx, 1); err != nil {
			return fmt.Errorf("failed to acquire semaphore: %w", err)
		}

		go func(networkName string, net discovery.Network) {
			defer sem.Release(1)

			if err := s.processNetwork(ctx, networkName, net); err != nil {
				s.logger.WithFields(logrus.Fields{
					"network": networkName,
					"error":   err,
				}).Error("Failed to process network")
			}
		}(name, network)
	}

	// Wait for all networks to be processed
	if err := sem.Acquire(ctx, 5); err != nil {
		return fmt.Errorf("failed to wait for completion: %w", err)
	}

	sem.Release(5)

	s.logger.Info("Validator ranges generation completed")

	return nil
}

// processNetwork handles the processing of a single network.
func (s *Service) processNetwork(ctx context.Context, networkName string, network discovery.Network) error {
	s.logger.WithField("network", networkName).Debug("Processing network")

	aggregatedRanges := s.buildNetworkRanges(ctx, networkName, network)
	if aggregatedRanges == nil {
		s.logger.WithField("network", networkName).Info("No validator ranges found, skipping")

		return nil
	}

	// Upload to S3
	if err := s.uploadToS3(ctx, networkName, aggregatedRanges); err != nil {
		return fmt.Errorf("failed to upload to S3: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"network":    networkName,
		"validators": aggregatedRanges.Validators.TotalCount,
		"nodes":      len(aggregatedRanges.Nodes),
	}).Info("Successfully processed network")

	return nil
}

// buildNetworkRanges gathers every validator range source for a network, or returns nil when none has data.
func (s *Service) buildNetworkRanges(ctx context.Context, networkName string, network discovery.Network) *ValidatorRanges {
	// Fetch from ethpandaops repository
	ethpandaopsRanges, err := s.fetchEthpandaopsRanges(ctx, networkName, network)
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"network": networkName,
			"error":   err,
		}).Warn("Failed to fetch ethpandaops ranges")
		// Continue even if ethpandaops fetch fails
	}

	// Fetch from additional sources if configured
	additionalRanges, err := s.fetchAdditionalRanges(ctx, networkName)
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"network": networkName,
			"error":   err,
		}).Warn("Failed to fetch additional ranges")
		// Continue even if additional fetches fail
	}

	// Combine all ranges
	allRanges := []*ValidatorRanges{}
	if ethpandaopsRanges != nil {
		allRanges = append(allRanges, ethpandaopsRanges)
	}

	allRanges = append(allRanges, additionalRanges...)

	var configRanges map[string][]*ValidatorRange

	configURL := configRangesURL(network)
	if configURL != "" {
		configRanges, err = s.fetchConfigRanges(ctx, configURL)
		if err != nil {
			s.logger.WithFields(logrus.Fields{
				"network": networkName,
				"url":     configURL,
				"error":   err,
			}).Warn("Failed to fetch config service validator ranges")
		}
	}

	if len(allRanges) == 0 && configRanges == nil {
		return nil
	}

	// Aggregate all ranges
	aggregatedRanges := AggregateRanges(allRanges)
	if configRanges != nil {
		ApplyConfigRanges(aggregatedRanges, configRanges, configURL, s.logger.WithField("network", networkName))
	}

	aggregatedRanges.Metadata.NetworkName = networkName

	return aggregatedRanges
}

// fetchEthpandaopsRanges fetches validator ranges from the ethpandaops repository.
func (s *Service) fetchEthpandaopsRanges(ctx context.Context, networkName string, network discovery.Network) (*ValidatorRanges, error) {
	// Extract repository from network
	repo := network.Repository
	if repo == "" {
		repo = "ethpandaops/ansible"
	}

	// Build inventory URLs
	urls := s.fetcher.BuildInventoryURLs(repo, inventoryDirName(networkName, repo))

	// Fetch inventory files
	contents, successfulURLs, err := s.fetcher.FetchMultiple(ctx, urls)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch inventory files: %w", err)
	}

	// Parse and aggregate all inventory files
	allRanges := make([]*ValidatorRanges, 0, len(contents))

	for i, content := range contents {
		ranges, err := ParseInventory(content, successfulURLs[i], "ethpandaops", 0)
		if err != nil {
			s.logger.WithFields(logrus.Fields{
				"url":   successfulURLs[i],
				"error": err,
			}).Warn("Failed to parse inventory")

			continue
		}

		allRanges = append(allRanges, ranges)
	}

	if len(allRanges) == 0 {
		return nil, fmt.Errorf("no valid inventory files found")
	}

	// Aggregate all ranges from ethpandaops
	return AggregateRanges(allRanges), nil
}

// fetchAdditionalRanges fetches validator ranges from additional configured sources.
func (s *Service) fetchAdditionalRanges(ctx context.Context, networkName string) ([]*ValidatorRanges, error) {
	if s.config == nil || s.config.AdditionalSources == nil {
		return nil, nil
	}

	sources, exists := s.config.AdditionalSources[networkName]
	if !exists || len(sources) == 0 {
		return nil, nil
	}

	allRanges := make([]*ValidatorRanges, 0, len(sources))

	for _, source := range sources {
		content, err := s.fetcher.FetchInventoryFile(ctx, source.URL)
		if err != nil {
			s.logger.WithFields(logrus.Fields{
				"url":   source.URL,
				"name":  source.Name,
				"error": err,
			}).Warn("Failed to fetch additional source")

			continue
		}

		ranges, err := ParseInventory(content, source.URL, source.Name, source.RangeOffset)
		if err != nil {
			s.logger.WithFields(logrus.Fields{
				"url":   source.URL,
				"name":  source.Name,
				"error": err,
			}).Warn("Failed to parse additional source")

			continue
		}

		allRanges = append(allRanges, ranges)
	}

	return allRanges, nil
}

// fetchConfigRanges fetches the on-chain validator ranges a network's config service publishes.
func (s *Service) fetchConfigRanges(ctx context.Context, url string) (map[string][]*ValidatorRange, error) {
	content, err := s.fetcher.FetchInventoryFile(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch config ranges: %w", err)
	}

	return ParseConfigRanges(content)
}

// uploadToS3 uploads validator ranges data to S3.
func (s *Service) uploadToS3(ctx context.Context, networkName string, data *ValidatorRanges) error {
	// Marshal data to JSON
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal validator ranges: %w", err)
	}

	// Upload to S3
	key := fmt.Sprintf("validator-ranges/%s.json", networkName)

	if err := s.s3Storage.UploadRaw(ctx, key, jsonData, "application/json"); err != nil {
		return fmt.Errorf("failed to upload to S3: %w", err)
	}

	return nil
}

// inventoryDirName maps e.g. glamsterdam-devnet-8 in ethpandaops/glamsterdam-devnets to devnet-8.
func inventoryDirName(networkName, repo string) string {
	repoName := repo[strings.LastIndex(repo, "/")+1:]

	prefix, ok := strings.CutSuffix(repoName, "devnets")
	if !ok || prefix == "" {
		return networkName
	}

	if after, ok := strings.CutPrefix(networkName, prefix); ok && after != "" {
		return after
	}

	return networkName
}
