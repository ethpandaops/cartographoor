package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ethpandaops/cartographoor/pkg/utils"

	"github.com/ethpandaops/cartographoor/pkg/discovery"
	"github.com/ethpandaops/cartographoor/pkg/providers/github"
	"github.com/ethpandaops/cartographoor/pkg/storage/s3"
)

type runConfig struct {
	Logging struct {
		Level string `mapstructure:"level"`
	} `mapstructure:"logging"`
	ConfigFile string
	Discovery  discovery.Config `mapstructure:"discovery"`
	Storage    s3.Config        `mapstructure:"storage"`
	RunOnce    bool             `mapstructure:"runOnce"`
}

func newRunCmd(log *logrus.Logger) *cobra.Command {
	cfg := &runConfig{}

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run the Cartographoor service",
		Long:  `Run the Cartographoor service to discover Ethereum networks and upload to S3`,
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

			return runService(cmd.Context(), log, cfg)
		},
	}

	// Define flags
	cmd.Flags().StringVar(&cfg.ConfigFile, "config", "", "Path to config file")
	cmd.Flags().StringVar(&cfg.Logging.Level, "logging.level", "info", "Logging level (trace, debug, info, warn, error, fatal, panic)")
	cmd.Flags().BoolVar(&cfg.RunOnce, "once", false, "Run discovery once and exit")

	return cmd
}

func runService(ctx context.Context, log *logrus.Logger, cfg *runConfig) error {
	// Set up context with cancellation
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Create discovery service
	discoveryService, err := discovery.NewService(log, cfg.Discovery)
	if err != nil {
		return err
	}

	// Create S3 storage provider
	storageProvider, err := s3.NewProvider(log, cfg.Storage)
	if err != nil {
		return err
	}

	// Register GitHub provider
	githubProvider, err := github.NewProvider(log)
	if err != nil {
		return err
	}

	discoveryService.RegisterProvider(githubProvider)

	// For run-once mode, we'll use a different approach
	if cfg.RunOnce {
		log.Info("Running in one-time discovery mode")

		return runOnce(ctx, log, discoveryService, storageProvider)
	}

	// Start the service in normal mode (continuous discovery).
	log.WithField("interval", cfg.Discovery.Interval).Info("Starting service in continuous mode")

	// Start discovery service
	if err := discoveryService.Start(ctx); err != nil {
		return err
	}

	// Set up discovery result handler
	discoveryService.OnResult(func(result discovery.Result) {
		log.WithField("networks", len(result.Networks)).Info("Discovered networks")

		// Skip upload if there are no networks or discovery had errors
		if len(result.Networks) == 0 {
			log.Info("No networks discovered, skipping S3 upload")

			return
		}

		// Upload to S3
		if err := storageProvider.Upload(ctx, result); err != nil {
			log.WithError(err).Error("Failed to upload networks to S3")
		}
	})

	// Handle graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-sigCh:
		log.Info("Received shutdown signal")
	case <-ctx.Done():
		log.Info("Context cancelled")
	}

	// Give a short grace period for cleanup
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := discoveryService.Stop(shutdownCtx); err != nil {
		log.WithError(err).Error("Error during discovery service shutdown")
	}

	return nil
}

// runOnce executes a single discovery run and uploads the results.
func runOnce(ctx context.Context, log *logrus.Logger, discoveryService *discovery.Service, storageProvider *s3.Provider) error {
	// Create a context with timeout to ensure we don't hang indefinitely
	runCtx, runCancel := context.WithTimeout(ctx, 5*time.Minute)
	defer runCancel()

	log.Info("Running one-time discovery")

	result, err := discoveryService.RunOnce(runCtx)
	if err != nil {
		return fmt.Errorf("failed to run discovery: %w", err)
	}

	log.WithField("networks", len(result.Networks)).Info("One-time discovery complete")

	// Skip upload if there are no networks
	if len(result.Networks) == 0 {
		log.Info("No networks discovered, skipping S3 upload")

		return nil
	}

	// Upload to S3
	if err := storageProvider.Upload(runCtx, result); err != nil {
		return fmt.Errorf("failed to upload networks to S3: %w", err)
	}

	log.Info("Upload complete, exiting")

	return nil
}

// readConfigWithEnvSubst reads a config file and performs environment variable substitution.
func readConfigWithEnvSubst(v *viper.Viper) error {
	configFile := v.ConfigFileUsed()
	if configFile == "" {
		configFile = v.GetString("config")
	}

	if configFile == "" {
		return nil
	}

	// Read the config file
	configData, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Substitute environment variables
	processedConfig := utils.EnvSubstBytes(configData)

	// Use ReadConfig instead of ReadInConfig to apply our processed config
	err = v.ReadConfig(strings.NewReader(string(processedConfig)))
	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	return nil
}
