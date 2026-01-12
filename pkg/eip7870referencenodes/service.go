package eip7870referencenodes

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/ethpandaops/cartographoor/pkg/storage/s3"
)

// Service orchestrates the generation of EIP-7870 reference node startup commands.
type Service struct {
	config         *Config
	s3Storage      *s3.Provider
	httpClient     *http.Client
	helmParser     *HelmChartParser
	platformParser *PlatformParser
	commandBuilder *CommandBuilder
	logger         logrus.FieldLogger
	githubToken    string
}

// NewService creates a new EIP-7870 reference nodes service.
func NewService(
	log logrus.FieldLogger,
	config *Config,
	s3Storage *s3.Provider,
	githubToken string,
) *Service {
	return &Service{
		config:    config,
		s3Storage: s3Storage,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		helmParser:     NewHelmChartParser(),
		platformParser: NewPlatformParser(config.SecretPatterns),
		commandBuilder: NewCommandBuilder(),
		logger:         log.WithField("module", "eip7870_reference_nodes"),
		githubToken:    githubToken,
	}
}

// Generate generates the EIP-7870 reference node commands and uploads to S3.
func (s *Service) Generate(ctx context.Context) error {
	s.logger.Info("Starting EIP-7870 reference nodes generation")

	clients := s.config.Clients
	if len(clients) == 0 {
		clients = DefaultClients
	}

	result := &Result{
		ReferenceNodes: make(map[string]*ClientCommand, len(clients)),
		Metadata: &Metadata{
			LastUpdated: time.Now().UTC(),
			Sources: SourcesInfo{
				HelmCharts: RepositoryInfo{
					Repository: s.config.HelmChartsRepository.Name,
					Branch:     s.config.HelmChartsRepository.Branch,
				},
				Platform: RepositoryInfo{
					Repository: s.config.PlatformRepository.Name,
					Branch:     s.config.PlatformRepository.Branch,
				},
			},
		},
	}

	// Fetch the 7870-reference feature file once (shared across all clients)
	featureYAML, err := s.fetchPlatformFeatureFile(ctx)
	if err != nil {
		s.logger.WithError(err).Warn("Failed to fetch 7870-reference feature file, continuing without feature overrides")
	}

	// Process each client
	for _, client := range clients {
		s.logger.WithField("client", client).Info("Processing client")

		cmd, err := s.processClient(ctx, client, featureYAML)
		if err != nil {
			s.logger.WithError(err).WithField("client", client).Error("Failed to process client")

			continue
		}

		result.ReferenceNodes[client] = cmd
	}

	if len(result.ReferenceNodes) == 0 {
		return fmt.Errorf("no clients were successfully processed")
	}

	// Upload to S3
	if err := s.uploadResult(ctx, result); err != nil {
		return fmt.Errorf("failed to upload result to S3: %w", err)
	}

	s.logger.WithField("clients", len(result.ReferenceNodes)).Info("EIP-7870 reference nodes generation completed")

	return nil
}

// processClient processes a single client to generate its startup command.
func (s *Service) processClient(ctx context.Context, client string, featureYAML []byte) (*ClientCommand, error) {
	// 1. Fetch and parse base command from helm charts
	baseArgs, err := s.fetchAndParseHelmChart(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch helm chart for %s: %w", client, err)
	}

	// 2. Fetch and parse client overlay from platform
	clientOverlay, err := s.fetchAndParseClientConfig(ctx, client)
	if err != nil {
		s.logger.WithError(err).WithField("client", client).Warn("Failed to fetch client config, using defaults")

		clientOverlay = &ClientOverlay{
			Args: make([]string, 0),
		}
	}

	// 3. Parse feature overlay
	var featureOverlay *FeatureOverlay

	if featureYAML != nil {
		featureOverlay, err = s.platformParser.ParseFeatureConfig(featureYAML, client)
		if err != nil {
			s.logger.WithError(err).WithField("client", client).Warn("Failed to parse feature config")
		}
	}

	// 4. Build the complete command
	cmd := s.commandBuilder.BuildCommand(client, baseArgs, clientOverlay, featureOverlay)

	return cmd, nil
}

// fetchAndParseHelmChart fetches and parses a helm chart values.yaml.
func (s *Service) fetchAndParseHelmChart(ctx context.Context, client string) ([]string, error) {
	url := fmt.Sprintf(
		"https://raw.githubusercontent.com/%s/%s/charts/%s/values.yaml",
		s.config.HelmChartsRepository.Name,
		s.config.HelmChartsRepository.Branch,
		client,
	)

	data, err := s.fetchURL(ctx, url)
	if err != nil {
		return nil, err
	}

	return s.helmParser.ParseBaseCommand(data, client)
}

// fetchAndParseClientConfig fetches and parses a client configuration from platform.
func (s *Service) fetchAndParseClientConfig(ctx context.Context, client string) (*ClientOverlay, error) {
	url := fmt.Sprintf(
		"https://raw.githubusercontent.com/%s/%s/environments/%s/applications/ethereum-node/values/clients/execution/%s.yaml",
		s.config.PlatformRepository.Name,
		s.config.PlatformRepository.Branch,
		s.config.PlatformRepository.Environment,
		client,
	)

	data, err := s.fetchURL(ctx, url)
	if err != nil {
		return nil, err
	}

	return s.platformParser.ParseClientConfig(data, client)
}

// fetchPlatformFeatureFile fetches the 7870-reference feature file.
func (s *Service) fetchPlatformFeatureFile(ctx context.Context) ([]byte, error) {
	url := fmt.Sprintf(
		"https://raw.githubusercontent.com/%s/%s/environments/%s/applications/ethereum-node/values/features/7870-reference.yaml",
		s.config.PlatformRepository.Name,
		s.config.PlatformRepository.Branch,
		s.config.PlatformRepository.Environment,
	)

	return s.fetchURL(ctx, url)
}

// fetchURL fetches content from a URL.
func (s *Service) fetchURL(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add GitHub token if available
	if s.githubToken != "" {
		req.Header.Set("Authorization", "token "+s.githubToken)
	}

	req.Header.Set("Accept", "application/vnd.github.v3.raw")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d for URL %s", resp.StatusCode, url)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return data, nil
}

// uploadResult uploads the result to S3.
func (s *Service) uploadResult(ctx context.Context, result *Result) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	key := s.config.Storage.Key
	if key == "" {
		key = "eip7870-reference-nodes.json"
	}

	s.logger.WithField("key", key).Info("Uploading result to S3")

	return s.s3Storage.UploadRaw(ctx, key, data, "application/json")
}

// SetDefaults applies default values to the config if not set.
func (c *Config) SetDefaults() {
	if c.HelmChartsRepository.Name == "" {
		c.HelmChartsRepository.Name = "ethpandaops/ethereum-helm-charts"
	}

	if c.HelmChartsRepository.Branch == "" {
		c.HelmChartsRepository.Branch = "master"
	}

	if c.PlatformRepository.Name == "" {
		c.PlatformRepository.Name = "ethpandaops/platform"
	}

	if c.PlatformRepository.Branch == "" {
		c.PlatformRepository.Branch = "master"
	}

	if c.PlatformRepository.Environment == "" {
		c.PlatformRepository.Environment = "production"
	}

	if len(c.Clients) == 0 {
		c.Clients = DefaultClients
	}

	if len(c.SecretPatterns) == 0 {
		c.SecretPatterns = DefaultSecretPatterns
	}

	if c.Storage.Key == "" {
		c.Storage.Key = "eip7870-reference-nodes.json"
	}
}

// Validate validates the config.
func (c *Config) Validate() error {
	if c.HelmChartsRepository.Name == "" {
		return fmt.Errorf("helmChartsRepository.name is required")
	}

	if c.PlatformRepository.Name == "" {
		return fmt.Errorf("platformRepository.name is required")
	}

	if !strings.Contains(c.HelmChartsRepository.Name, "/") {
		return fmt.Errorf("helmChartsRepository.name must be in format owner/repo")
	}

	if !strings.Contains(c.PlatformRepository.Name, "/") {
		return fmt.Errorf("platformRepository.name must be in format owner/repo")
	}

	return nil
}
