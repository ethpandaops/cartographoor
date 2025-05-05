package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ethpandaops/network-status/pkg/discovery"
	"github.com/ethpandaops/network-status/pkg/providers/github"
	"github.com/ethpandaops/network-status/pkg/storage/s3"
)

type runConfig struct {
	LoggingLevel string `mapstructure:"logging.level"`
	ConfigFile   string
	Discovery    discovery.Config `mapstructure:"discovery"`
	Storage      s3.Config        `mapstructure:"storage"`
}

func newRunCmd(log *logrus.Logger) *cobra.Command {
	cfg := &runConfig{}

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run the network status service",
		Long:  `Run the network status service to discover Ethereum networks and upload to S3`,
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.New()

			if cfg.ConfigFile != "" {
				v.SetConfigFile(cfg.ConfigFile)
				if err := v.ReadInConfig(); err != nil {
					return err
				}
			}

			v.SetEnvPrefix("NETWORK_STATUS")
			v.AutomaticEnv()

			if err := v.Unmarshal(cfg); err != nil {
				return err
			}

			// Set log level
			level, err := logrus.ParseLevel(cfg.LoggingLevel)
			if err == nil {
				log.SetLevel(level)
			}

			return runService(cmd.Context(), log, cfg)
		},
	}

	// Define flags
	cmd.Flags().StringVar(&cfg.ConfigFile, "config", "", "Path to config file")
	cmd.Flags().StringVar(&cfg.LoggingLevel, "logging.level", "info", "Logging level (trace, debug, info, warn, error, fatal, panic)")

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

	// Start discovery service
	if err := discoveryService.Start(ctx); err != nil {
		return err
	}

	// Set up discovery result handler
	discoveryService.OnResult(func(result discovery.Result) {
		log.WithField("networks", len(result.Networks)).Info("Discovered networks")

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