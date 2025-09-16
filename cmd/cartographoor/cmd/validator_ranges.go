package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ethpandaops/cartographoor/pkg/discovery"
	"github.com/ethpandaops/cartographoor/pkg/storage/s3"
	"github.com/ethpandaops/cartographoor/pkg/validatorranges"
)

type validatorRangesConfig struct {
	Logging struct {
		Level string `mapstructure:"level"`
	} `mapstructure:"logging"`
	ConfigFile      string
	Storage         s3.Config               `mapstructure:"storage"`
	ValidatorRanges *validatorranges.Config `mapstructure:"validatorRanges"`
}

func newValidatorRangesCmd(log *logrus.Logger) *cobra.Command {
	cfg := &validatorRangesConfig{}

	cmd := &cobra.Command{
		Use:   "validator-ranges",
		Short: "Generate validator ranges for discovered networks",
		Long:  `Downloads networks.json and generates validator range data from Ansible inventory files`,
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.New()

			if cfg.ConfigFile != "" {
				v.SetConfigFile(cfg.ConfigFile)

				// Read and process the config file with environment variable substitution
				if err := readConfigWithEnvSubst(v); err != nil {
					return err
				}
			}

			v.SetEnvPrefix("CARTOGRAPHOOR")
			v.AutomaticEnv()

			if err := v.Unmarshal(cfg); err != nil {
				return err
			}

			// Set log level
			level, err := logrus.ParseLevel(cfg.Logging.Level)
			if err == nil {
				log.SetLevel(level)
			}

			return runValidatorRanges(cmd.Context(), log, cfg)
		},
	}

	// Define flags
	cmd.Flags().StringVarP(&cfg.ConfigFile, "config", "c", "", "Path to config file")
	cmd.Flags().StringVar(&cfg.Logging.Level, "logging.level", "info", "Logging level (trace, debug, info, warn, error, fatal, panic)")

	// Mark config as required
	if err := cmd.MarkFlagRequired("config"); err != nil {
		log.WithError(err).Fatal("Failed to mark config flag as required")
	}

	return cmd
}

func runValidatorRanges(ctx context.Context, log *logrus.Logger, cfg *validatorRangesConfig) error {
	// Set up context with cancellation
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Handle signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Info("Received shutdown signal, cancelling context")
		cancel()
	}()

	// Create S3 storage provider
	storageProvider, err := s3.NewProvider(log, cfg.Storage)
	if err != nil {
		return fmt.Errorf("failed to create S3 storage provider: %w", err)
	}

	// Initialize storage provider
	if initErr := storageProvider.Initialize(ctx); initErr != nil {
		return fmt.Errorf("failed to initialize storage provider: %w", initErr)
	}

	log.Info("Downloading networks.json from S3")

	// Download networks.json from S3
	networksData, err := storageProvider.Download(ctx, "networks.json")
	if err != nil {
		return fmt.Errorf("failed to download networks.json: %w", err)
	}

	// Parse networks.json
	var discoveryResult discovery.Result
	if err := json.Unmarshal(networksData, &discoveryResult); err != nil {
		return fmt.Errorf("failed to parse networks.json: %w", err)
	}

	log.WithField("networks", len(discoveryResult.Networks)).Info("Downloaded networks from S3")

	// Create validator ranges service
	service := validatorranges.NewService(storageProvider, cfg.ValidatorRanges, log)

	log.Info("Starting validator ranges generation")

	// Process all active networks
	activeNetworks := make(map[string]discovery.Network)

	for name, network := range discoveryResult.Networks {
		if network.Status == "active" || network.Status == "running" {
			activeNetworks[name] = network
		}
	}

	log.WithField("active_networks", len(activeNetworks)).Info("Processing active networks")

	// Generate validator ranges for all networks
	if err := service.GenerateValidatorRanges(ctx, activeNetworks); err != nil {
		return fmt.Errorf("validator ranges generation failed: %w", err)
	}

	log.Info("Validator ranges generation completed successfully")

	return nil
}
