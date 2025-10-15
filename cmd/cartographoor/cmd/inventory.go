package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ethpandaops/cartographoor/pkg/inventory"
	"github.com/ethpandaops/cartographoor/pkg/storage/s3"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type inventoryConfig struct {
	Logging struct {
		Level string `mapstructure:"level"`
	} `mapstructure:"logging"`
	ConfigFile string
	Storage    s3.Config         `mapstructure:"storage"`
	Inventory  inventorySettings `mapstructure:"inventory"`
}

type inventorySettings struct {
	Validation validationSettings `mapstructure:"validation"`
}

type validationSettings struct {
	Enabled                  *bool  `mapstructure:"enabled"`
	DNSTimeout               string `mapstructure:"dnsTimeout"`
	MaxConcurrentValidations int64  `mapstructure:"maxConcurrentValidations"`
}

func newInventoryCmd(log *logrus.Logger) *cobra.Command {
	cfg := &inventoryConfig{}

	cmd := &cobra.Command{
		Use:   "inventory",
		Short: "Generate network inventory from Dora APIs",
		Long:  `Fetches client information from Dora APIs and generates inventory files`,
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

			return runInventory(cmd.Context(), log, cfg)
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

func runInventory(ctx context.Context, log *logrus.Logger, cfg *inventoryConfig) error {
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

	// Parse and prepare inventory configuration
	inventoryCfg, err := prepareInventoryConfig(&cfg.Inventory)
	if err != nil {
		return fmt.Errorf("failed to prepare inventory configuration: %w", err)
	}

	// Create inventory service
	service, err := inventory.NewService(
		log.WithField("component", "inventory"),
		cfg.Storage,
		*inventoryCfg,
	)
	if err != nil {
		return fmt.Errorf("failed to create inventory service: %w", err)
	}

	log.Info("Starting inventory generation")

	// Run the inventory generation
	if err := service.Run(ctx); err != nil {
		return fmt.Errorf("inventory generation failed: %w", err)
	}

	log.Info("Inventory generation completed successfully")

	return nil
}

// prepareInventoryConfig prepares the inventory configuration with defaults.
func prepareInventoryConfig(cfg *inventorySettings) (*inventory.Config, error) {
	// Default values
	const (
		defaultEnabled                  = true
		defaultMaxConcurrentValidations = int64(100)
	)

	inventoryCfg := &inventory.Config{
		Validation: inventory.ValidationConfig{
			Enabled:                  defaultEnabled,
			DNSTimeout:               3 * time.Second,
			MaxConcurrentValidations: defaultMaxConcurrentValidations,
		},
	}

	// Override with configured values if provided
	if cfg.Validation.Enabled != nil {
		inventoryCfg.Validation.Enabled = *cfg.Validation.Enabled
	}

	if cfg.Validation.DNSTimeout != "" {
		timeout, err := time.ParseDuration(cfg.Validation.DNSTimeout)
		if err != nil {
			return nil, fmt.Errorf("invalid dnsTimeout value: %w", err)
		}

		inventoryCfg.Validation.DNSTimeout = timeout
	}

	if cfg.Validation.MaxConcurrentValidations > 0 {
		inventoryCfg.Validation.MaxConcurrentValidations = cfg.Validation.MaxConcurrentValidations
	}

	return inventoryCfg, nil
}
