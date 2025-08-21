package inventory

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"sync"
	"sync/atomic"

	"github.com/ethpandaops/cartographoor/pkg/discovery"
	"github.com/ethpandaops/cartographoor/pkg/storage/s3"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/semaphore"
)

// Service orchestrates inventory generation.
type Service struct {
	log       *logrus.Entry
	generator *Generator
	storage   *s3.Provider
}

// NewService creates a new inventory service.
func NewService(log *logrus.Entry, storageConfig s3.Config) (*Service, error) {
	// Create S3 storage provider
	storage, err := s3.NewProvider(log.Logger, storageConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage provider: %w", err)
	}

	return &Service{
		log:       log.WithField("component", "inventory_service"),
		generator: NewGenerator(log),
		storage:   storage,
	}, nil
}

// Run executes the inventory generation process.
func (s *Service) Run(ctx context.Context) error {
	s.log.Info("Starting inventory generation")

	// Download the latest networks.json from S3
	networksData, err := s.downloadNetworksJSON(ctx)
	if err != nil {
		return fmt.Errorf("failed to download networks.json: %w", err)
	}

	// Filter for active networks from GitHub repositories with Dora URLs
	activeNetworks := make(map[string]discovery.Network)
	skippedStatic := 0
	skippedNoRepo := 0

	for name, network := range networksData.Networks {
		// Skip networks without repository (static networks)
		if network.Repository == "" {
			s.log.WithField("network", name).Debug("Skipping network without repository (static network)")

			skippedStatic++

			continue
		}

		// Only process active networks with Dora URLs
		if network.Status == "active" &&
			network.ServiceURLs != nil &&
			network.ServiceURLs.Dora != "" {
			activeNetworks[name] = network
			s.log.WithFields(logrus.Fields{
				"network":    name,
				"repository": network.Repository,
			}).Debug("Including network for inventory generation")
		} else {
			skippedNoRepo++
		}
	}

	s.log.WithFields(logrus.Fields{
		"active_networks": len(activeNetworks),
		"skipped_static":  skippedStatic,
		"skipped_other":   skippedNoRepo,
	}).Info("Found active GitHub networks with Dora URLs")

	// Generate inventories for each network concurrently
	inventories := s.generateInventories(ctx, activeNetworks)

	// Upload inventory files to S3
	if err := s.uploadInventories(ctx, inventories); err != nil {
		return fmt.Errorf("failed to upload inventories: %w", err)
	}

	s.log.WithField("inventories_uploaded", len(inventories)).Info("Inventory generation completed")

	return nil
}

// downloadNetworksJSON downloads and parses the networks.json file from S3.
func (s *Service) downloadNetworksJSON(ctx context.Context) (*NetworksResult, error) {
	s.log.Debug("Downloading networks.json from S3")

	// Download the file
	data, err := s.storage.Download(ctx, "networks.json")
	if err != nil {
		return nil, fmt.Errorf("failed to download networks.json: %w", err)
	}

	// Parse the JSON using our simplified struct
	var result NetworksResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse networks.json: %w", err)
	}

	s.log.WithFields(logrus.Fields{
		"networks":   len(result.Networks),
		"clients":    len(result.Clients),
		"metadata":   len(result.NetworkMetadata),
		"lastUpdate": result.LastUpdate,
	}).Debug("Successfully parsed networks.json")

	return &result, nil
}

// generateInventories generates inventory data for all networks concurrently.
func (s *Service) generateInventories(ctx context.Context, networks map[string]discovery.Network) map[string]*InventoryData {
	const maxConcurrent = 5

	sem := semaphore.NewWeighted(maxConcurrent)
	inventories := make(map[string]*InventoryData)
	mu := sync.Mutex{}

	var (
		wg           sync.WaitGroup
		successCount atomic.Int32
		failureCount atomic.Int32
	)

	for name, network := range networks {
		wg.Add(1)

		go func() {
			defer wg.Done()

			// Acquire semaphore
			if err := sem.Acquire(ctx, 1); err != nil {
				s.log.WithError(err).WithField("network", name).Error("Failed to acquire semaphore")
				failureCount.Add(1)

				return
			}
			defer sem.Release(1)

			// Generate inventory for this network
			inventory, err := s.generator.GenerateForNetwork(ctx, network)
			if err != nil {
				s.log.WithError(err).WithField("network", name).Error("Failed to generate inventory")
				failureCount.Add(1)

				return
			}

			// Skip if no inventory was generated (e.g., no Dora URL)
			if inventory == nil {
				s.log.WithField("network", name).Debug("No inventory generated for network")

				return
			}

			// Store the inventory
			mu.Lock()
			inventories[name] = inventory
			mu.Unlock()

			successCount.Add(1)
			s.log.WithField("network", name).Debug("Successfully generated inventory")
		}()
	}

	wg.Wait()

	s.log.WithFields(logrus.Fields{
		"success": successCount.Load(),
		"failure": failureCount.Load(),
		"total":   len(networks),
	}).Info("Completed inventory generation for all networks")

	return inventories
}

// uploadInventories uploads all inventory files to S3.
func (s *Service) uploadInventories(ctx context.Context, inventories map[string]*InventoryData) error {
	var (
		wg          sync.WaitGroup
		mu          sync.Mutex
		uploadError error
	)

	for name, inventory := range inventories {
		wg.Add(1)

		go func() {
			defer wg.Done()

			// Marshal inventory to JSON
			data, err := json.MarshalIndent(inventory, "", "  ")
			if err != nil {
				s.log.WithError(err).WithField("network", name).Error("Failed to marshal inventory")

				mu.Lock()
				if uploadError == nil {
					uploadError = fmt.Errorf("failed to marshal inventory for %s: %w", name, err)
				}
				mu.Unlock()

				return
			}

			// Construct the S3 key
			key := path.Join("inventory", fmt.Sprintf("%s.json", name))

			// Upload to S3
			s.log.WithFields(logrus.Fields{
				"network": name,
				"key":     key,
				"size":    len(data),
			}).Debug("Uploading inventory to S3")

			// Upload the inventory file
			if err := s.uploadInventoryFile(ctx, key, data); err != nil {
				s.log.WithError(err).WithField("network", name).Error("Failed to upload inventory")

				mu.Lock()
				if uploadError == nil {
					uploadError = fmt.Errorf("failed to upload inventory for %s: %w", name, err)
				}
				mu.Unlock()

				return
			}

			s.log.WithField("network", name).Debug("Successfully uploaded inventory")
		}()
	}

	wg.Wait()

	return uploadError
}

// uploadInventoryFile uploads a single inventory file to S3.
func (s *Service) uploadInventoryFile(ctx context.Context, key string, data []byte) error {
	// Upload the raw JSON data to S3
	if err := s.storage.UploadRaw(ctx, key, data, "application/json"); err != nil {
		return fmt.Errorf("failed to upload inventory file: %w", err)
	}

	return nil
}
