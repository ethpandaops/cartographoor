package github

import (
	"context"
	"fmt"
	"path"
	"strings"

	gh "github.com/google/go-github/v53/github"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"

	"github.com/ethpandaops/cartographoor/pkg/discovery"
)

// Provider implements the discovery.Provider interface for GitHub.
type Provider struct {
	log    *logrus.Logger
	client *gh.Client
}

// NewProvider creates a new GitHub provider.
func NewProvider(log *logrus.Logger) (*Provider, error) {
	log = log.WithField("provider", "github").Logger

	return &Provider{
		log: log,
	}, nil
}

// Name returns the name of the provider.
func (p *Provider) Name() string {
	return "github"
}

// Discover discovers networks using GitHub.
func (p *Provider) Discover(ctx context.Context, config discovery.Config) (map[string]discovery.Network, error) {
	if len(config.GitHub.Repositories) == 0 {
		return nil, fmt.Errorf("no repositories configured")
	}

	// We require a token to be set in production, otherwise we'll just get rate-limited.
	// Skip this check if the client is already set (for testing purposes)
	if config.GitHub.Token == "" && p.client == nil {
		return nil, fmt.Errorf("no GitHub token configured")
	}

	// Create GitHub client
	client := p.getClient(ctx, config.GitHub.Token)

	networks := make(map[string]discovery.Network)

	// Discover networks for each repository
	for _, repoConfig := range config.GitHub.Repositories {
		discoveredNetworks, err := p.discoverRepositoryNetworks(ctx, client, repoConfig)
		if err != nil {
			p.log.WithError(err).WithField("repository", repoConfig.Name).Error("Failed to discover networks in repository")

			continue
		}

		// Add discovered networks to the result
		for name, network := range discoveredNetworks {
			networks[name] = network
		}
	}

	return networks, nil
}

// discoverRepositoryNetworks discovers networks in a specific repository.
func (p *Provider) discoverRepositoryNetworks(
	ctx context.Context,
	client *gh.Client,
	repoConfig discovery.GitHubRepositoryConfig,
) (map[string]discovery.Network, error) {
	var (
		repoPath   = repoConfig.Name
		namePrefix = repoConfig.NamePrefix
		parts      = strings.Split(repoPath, "/")
	)

	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid repository path: %s", repoPath)
	}

	owner, repo := parts[0], parts[1]
	p.log.WithFields(logrus.Fields{
		"owner":      owner,
		"repo":       repo,
		"namePrefix": namePrefix,
	}).Info("Discovering networks in repository")

	// Check if network-configs directory exists
	netConfigPath := networkConfigDir

	_, dirContent, _, err := client.Repositories.GetContents(ctx, owner, repo, netConfigPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get contents of network-configs directory: %w", err)
	}

	networks := make(map[string]discovery.Network)

	// Process directories in network-configs
	for _, content := range dirContent {
		if *content.Type != "dir" {
			continue
		}

		networkConfig := &NetworkConfig{
			Name:         *content.Name,
			PrefixedName: *content.Name,
			Repository:   repoPath,
			Owner:        owner,
			Repo:         repo,
			Path:         path.Join(netConfigPath, *content.Name),
			URL:          *content.HTMLURL,
		}

		// Apply prefix if configured
		if namePrefix != "" {
			networkConfig.PrefixedName = namePrefix + networkConfig.Name
		}

		// Determine network status, configs, domain, and images
		var images *discovery.Images
		networkConfig.Status, networkConfig.ConfigFiles, networkConfig.Domain, images = p.getNetworkDetails(
			ctx, client, owner, repo, networkConfig.Name,
		)

		// Copy images data to network config if available
		if images != nil {
			networkConfig.Images.URL = images.URL
			networkConfig.Images.Clients = images.Clients
			networkConfig.Images.Tools = images.Tools
		}

		// Create network and add to result
		networks[networkConfig.PrefixedName] = p.createNetwork(ctx, networkConfig)
	}

	return networks, nil
}

// getClient returns a GitHub client.
func (p *Provider) getClient(ctx context.Context, token string) *gh.Client {
	if p.client != nil {
		return p.client
	}

	if token != "" {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		)
		tc := oauth2.NewClient(ctx, ts)
		p.client = gh.NewClient(tc)
	} else {
		p.client = gh.NewClient(nil)
	}

	return p.client
}
