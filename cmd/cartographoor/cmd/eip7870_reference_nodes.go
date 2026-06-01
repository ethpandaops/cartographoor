package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ethpandaops/cartographoor/pkg/eip7870referencenodes"
	"github.com/ethpandaops/cartographoor/pkg/storage/s3"
)

type eip7870ReferenceNodesConfig struct {
	Logging struct {
		Level string `mapstructure:"level"`
	} `mapstructure:"logging"`
	ConfigFile            string
	Storage               s3.Config                     `mapstructure:"storage"`
	EIP7870ReferenceNodes *eip7870referencenodes.Config `mapstructure:"eip7870ReferenceNodes"`
	GitHubToken           string                        `mapstructure:"githubToken"`
}

func newEIP7870ReferenceNodesCmd(log *logrus.Logger) *cobra.Command {
	cfg := &eip7870ReferenceNodesConfig{}

	cmd := &cobra.Command{
		Use:   "eip7870-reference-nodes",
		Short: "Generate EIP-7870 reference node startup commands",
		Long: `Fetches configuration from ethereum-helm-charts and platform repositories,
generates startup commands for EIP-7870 reference execution clients,
and uploads the result to S3.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
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

			// Bind GitHub token from environment
			if err := v.BindEnv("githubToken", "GITHUB_TOKEN"); err != nil {
				log.WithError(err).Warn("Failed to bind GITHUB_TOKEN environment variable")
			}

			if err := v.Unmarshal(cfg); err != nil {
				return err
			}

			// Set log level
			level, err := logrus.ParseLevel(cfg.Logging.Level)
			if err == nil {
				log.SetLevel(level)
			}

			return runEIP7870ReferenceNodes(cmd.Context(), log, cfg)
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

func runEIP7870ReferenceNodes(ctx context.Context, log *logrus.Logger, cfg *eip7870ReferenceNodesConfig) error {
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

	// Validate config
	if cfg.EIP7870ReferenceNodes == nil {
		return fmt.Errorf("eip7870ReferenceNodes configuration is required")
	}

	// Apply defaults
	cfg.EIP7870ReferenceNodes.SetDefaults()

	// Validate
	if err := cfg.EIP7870ReferenceNodes.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Create S3 storage provider
	storageProvider, err := s3.NewProvider(log, cfg.Storage)
	if err != nil {
		return fmt.Errorf("failed to create S3 storage provider: %w", err)
	}

	// Initialize storage provider
	if initErr := storageProvider.Initialize(ctx); initErr != nil {
		return fmt.Errorf("failed to initialize storage provider: %w", initErr)
	}

	// Get GitHub token
	githubToken := cfg.GitHubToken
	if githubToken == "" {
		githubToken = os.Getenv("GITHUB_TOKEN")
	}

	if githubToken == "" {
		log.Warn("No GitHub token configured, rate limits may apply")
	}

	// Create service
	service := eip7870referencenodes.NewService(
		log,
		cfg.EIP7870ReferenceNodes,
		storageProvider,
		githubToken,
	)

	log.Info("Starting EIP-7870 reference nodes generation")

	// Generate and upload
	if err := service.Generate(ctx); err != nil {
		return fmt.Errorf("EIP-7870 reference nodes generation failed: %w", err)
	}

	log.Info("EIP-7870 reference nodes generation completed successfully")

	return nil
}
